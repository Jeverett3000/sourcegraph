package registry

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	frontendregistry "github.com/sourcegraph/sourcegraph/cmd/frontend/registry/api"
	registry "github.com/sourcegraph/sourcegraph/cmd/frontend/registry/client"
	"github.com/sourcegraph/sourcegraph/enterprise/cmd/frontend/internal/registry/stores"
	"github.com/sourcegraph/sourcegraph/internal/conf"
	"github.com/sourcegraph/sourcegraph/internal/database"
	"github.com/sourcegraph/sourcegraph/internal/errcode"
	"github.com/sourcegraph/sourcegraph/internal/jsonc"
	"github.com/sourcegraph/sourcegraph/schema"
)

func init() {
	frontendregistry.HandleRegistry = handleRegistry
}

func registryList(ctx context.Context, db database.DB, opt stores.ExtensionsListOptions) ([]*registry.Extension, error) {
	if useFrozenRegistryData(ctx, db) {
		return frontendregistry.FilterRegistryExtensions(getFrozenRegistryData(), opt.Query), nil
	}

	vs, err := stores.Extensions(db).List(ctx, opt)
	if err != nil {
		return nil, err
	}

	xs, err := toRegistryAPIExtensionBatch(ctx, db, vs)
	if err != nil {
		return nil, err
	}

	ys := make([]*registry.Extension, 0, len(xs))
	for _, x := range xs {
		// To be safe, ensure that the JSON can be safely unmarshaled by API clients. If not,
		// skip this extension.
		if x.Manifest != nil {
			var o schema.SourcegraphExtensionManifest
			if err := jsonc.Unmarshal(*x.Manifest, &o); err != nil {
				continue
			}
		}
		ys = append(ys, x)
	}
	return ys, nil
}

func registryGetByUUID(ctx context.Context, db database.DB, uuid string) (*registry.Extension, error) {
	if useFrozenRegistryData(ctx, db) {
		return frontendregistry.FindRegistryExtension(getFrozenRegistryData(), "uuid", uuid), nil
	}

	x, err := stores.Extensions(db).GetByUUID(ctx, uuid)
	if err != nil {
		return nil, err
	}
	return toRegistryAPIExtension(ctx, db, x)
}

func registryGetByExtensionID(ctx context.Context, db database.DB, extensionID string) (*registry.Extension, error) {
	if useFrozenRegistryData(ctx, db) {
		return frontendregistry.FindRegistryExtension(getFrozenRegistryData(), "extensionID", extensionID), nil
	}

	x, err := stores.Extensions(db).GetByExtensionID(ctx, extensionID)
	if err != nil {
		return nil, err
	}
	return toRegistryAPIExtension(ctx, db, x)
}

func toRegistryAPIExtension(ctx context.Context, db database.DB, v *stores.Extension) (*registry.Extension, error) {
	release, err := getLatestRelease(ctx, stores.Releases(db), v.NonCanonicalExtensionID, v.ID, "release")
	if err != nil {
		return nil, err
	}

	if release == nil {
		return newExtension(v, nil, time.Time{}), nil
	}

	return newExtension(v, &release.Manifest, release.CreatedAt), nil
}

func toRegistryAPIExtensionBatch(ctx context.Context, db database.DB, vs []*stores.Extension) ([]*registry.Extension, error) {
	releasesByExtensionID, err := getLatestForBatch(ctx, db, vs)
	if err != nil {
		return nil, err
	}

	var extensions []*registry.Extension
	for _, v := range vs {
		release, ok := releasesByExtensionID[v.ID]
		if !ok {
			extensions = append(extensions, newExtension(v, nil, time.Time{}))
		} else {
			extensions = append(extensions, newExtension(v, &release.Manifest, release.CreatedAt))
		}
	}
	return extensions, nil
}

func newExtension(v *stores.Extension, manifest *string, publishedAt time.Time) *registry.Extension {
	baseURL := strings.TrimSuffix(conf.Get().ExternalURL, "/")
	return &registry.Extension{
		UUID:        v.UUID,
		ExtensionID: v.NonCanonicalExtensionID,
		Name:        v.Name,
		Manifest:    manifest,
		CreatedAt:   v.CreatedAt,
		UpdatedAt:   v.UpdatedAt,
		PublishedAt: publishedAt,
		URL:         baseURL + frontendregistry.ExtensionURL(v.NonCanonicalExtensionID),
	}
}

type responseRecorder struct {
	http.ResponseWriter
	code int
}

func (r *responseRecorder) WriteHeader(code int) {
	r.code = code
	r.ResponseWriter.WriteHeader(code)
}

// handleRegistry serves the external HTTP API for the extension registry.
func handleRegistry(db database.DB) func(w http.ResponseWriter, r *http.Request) (err error) {
	return func(w http.ResponseWriter, r *http.Request) (err error) {
		recorder := &responseRecorder{ResponseWriter: w, code: http.StatusOK}
		w = recorder

		var operation string
		defer func(began time.Time) {
			seconds := time.Since(began).Seconds()
			if err != nil && recorder.code == http.StatusOK {
				recorder.code = http.StatusInternalServerError
			}
			code := strconv.Itoa(recorder.code)
			registryRequestsDuration.WithLabelValues(operation, code).Observe(seconds)
		}(time.Now())

		if conf.Extensions() == nil {
			w.WriteHeader(http.StatusNotFound)
			return nil
		}

		// Identify this response as coming from the registry API.
		w.Header().Set(registry.MediaTypeHeaderName, registry.MediaType)

		// The response differs based on some request headers, and we need to tell caches which ones.
		//
		// Accept, User-Agent: because these encode the registry client's API version, and responses are
		// not cacheable across versions.
		w.Header().Set("Vary", "Accept, User-Agent")

		// Validate API version.
		if v := r.Header.Get("Accept"); v != registry.AcceptHeader {
			http.Error(w, fmt.Sprintf("invalid Accept header: expected %q", registry.AcceptHeader), http.StatusBadRequest)
			return nil
		}

		// This handler can be mounted at either /.internal or /.api.
		urlPath := r.URL.Path
		switch {
		case strings.HasPrefix(urlPath, "/.internal"):
			urlPath = strings.TrimPrefix(urlPath, "/.internal")
		case strings.HasPrefix(urlPath, "/.api"):
			urlPath = strings.TrimPrefix(urlPath, "/.api")
		}

		const extensionsPath = "/registry/extensions"
		var result any
		switch {
		case urlPath == extensionsPath:
			operation = "list"

			opt := stores.ExtensionsListOptions{
				Query: r.URL.Query().Get("q"),
			}
			xs, err := registryList(r.Context(), db, opt)
			if err != nil {
				return err
			}
			result = xs

		case urlPath == extensionsPath+"/featured":
			operation = "featured"
			result = []struct{}{}

		case strings.HasPrefix(urlPath, extensionsPath+"/"):
			var (
				spec = strings.TrimPrefix(urlPath, extensionsPath+"/")
				x    *registry.Extension
				err  error
			)
			switch {
			case strings.HasPrefix(spec, "uuid/"):
				operation = "get-by-uuid"
				x, err = registryGetByUUID(r.Context(), db, strings.TrimPrefix(spec, "uuid/"))
			case strings.HasPrefix(spec, "extension-id/"):
				operation = "get-by-extension-id"
				x, err = registryGetByExtensionID(r.Context(), db, strings.TrimPrefix(spec, "extension-id/"))
			default:
				w.WriteHeader(http.StatusNotFound)
				return nil
			}
			if x == nil || err != nil {
				if x == nil || errcode.IsNotFound(err) {
					w.Header().Set("Cache-Control", "max-age=5, private")
					http.Error(w, "extension not found", http.StatusNotFound)
					return nil
				}
				return err
			}
			result = x

		default:
			w.WriteHeader(http.StatusNotFound)
			return nil
		}

		w.Header().Set("Cache-Control", "max-age=120, private")
		return json.NewEncoder(w).Encode(result)
	}
}

var (
	registryRequestsDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name: "src_registry_requests_duration_seconds",
		Help: "Seconds spent handling a request to the HTTP registry API",
	}, []string{"operation", "code"})
)
