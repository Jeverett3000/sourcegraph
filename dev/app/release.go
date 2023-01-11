// Program release builds and uploads Zip archives containing release artifacts for Sourcegraph App.
//
// This program expects Google Cloud application default credentials set up. Run:
//
//   gcloud auth application-default login
//
// See https://cloud.google.com/docs/authentication/application-default-credentials#personal for
// more information.
//
// Run it as:
//
//   go run ./enterprise/cmd/frontend/internal/registry/scripts/freeze_legacy_extensions.go > enterprise/cmd/frontend/internal/registry/frozen_legacy_extensions.json
//
// Note: In case it's helpful, run this to get the list of extensions from sourcegraph.com's API:
//
//   curl -v -H 'Accept: application/vnd.sourcegraph.api+json;version=20180621' https://sourcegraph.com/.api/registry/extensions

package main

import (
	"archive/zip"
	"bytes"
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"time"

	"cloud.google.com/go/storage"
	"github.com/sourcegraph/sourcegraph/lib/errors"
	"google.golang.org/api/googleapi"
)

var (
	gcpProject     = flag.String("project", "sourcegraph-app-releases", "Google Cloud Platform project in which to create the bucket")
	gcsBucket      = flag.String("bucket", "sourcegraph-app-releases", "Google Cloud Storage bucket where releases are uploaded")
	releaseVersion = flag.String("release-version", "dev", "release version string") // TODO(sqs): make same as in build.sh
	skipBuild      = flag.Bool("skip-build", false, "skip building")
)

func main() {
	log.SetFlags(0)
	flag.Parse()

	artifacts, err := buildArtifacts(*releaseVersion)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("# Uploading artifacts to Google Cloud Storage...")
	if err := uploadArtifacts(*gcpProject, *gcsBucket, artifacts); err != nil {
		log.Fatal(err)
	}
}

type artifact struct {
	filename string
	os, arch string
	data     []byte
}

func buildArtifacts(releaseVersion string) ([]artifact, error) {
	program, err := buildProgram()
	if err != nil {
		return nil, err
	}
	zipData, err := createZipArchive(program)
	if err != nil {
		return nil, err
	}
	zipArtifact := artifact{
		filename: fmt.Sprintf("sourcegraph-%s-%s-%s.zip", releaseVersion, program.os, program.arch),
		os:       program.os,
		arch:     program.arch,
		data:     zipData,
	}

	return []artifact{zipArtifact}, nil
}

func buildProgram() (*artifact, error) {
	if !*skipBuild {
		out, err := exec.Command("enterprise/cmd/sourcegraph/build.sh").CombinedOutput()
		if err != nil {
			return nil, fmt.Errorf("error building: %w\n\n%s", err, out)
		}
	}

	localPath := fmt.Sprintf(".bin/sourcegraph-%s-%s-dist", runtime.GOOS, runtime.GOARCH)
	data, err := os.ReadFile(localPath)
	if err != nil {
		return nil, err
	}

	return &artifact{
		filename: "sourcegraph",
		os:       runtime.GOOS,
		arch:     runtime.GOARCH,
		data:     data,
	}, nil
}

func createZipArchive(program *artifact) ([]byte, error) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	fh := &zip.FileHeader{
		Name:     "sourcegraph",
		Method:   zip.Deflate,
		Modified: time.Now().UTC(),
	}
	fh.SetMode(0755) // executable
	w, err := zw.CreateHeader(fh)
	if err != nil {
		return nil, err
	}
	if _, err = w.Write(program.data); err != nil {
		return nil, err
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func uploadArtifacts(projectName, bucketName string, artifacts []artifact) error {
	ctx := context.Background()
	gcsClient, err := storage.NewClient(ctx)
	if err != nil {
		return errors.WithMessage(err, "failed to initialize Google Cloud Storage client")
	}

	bkt := gcsClient.Bucket(bucketName)
	if err := bkt.Create(ctx, projectName, &storage.BucketAttrs{PredefinedACL: "private"}); err != nil {
		if e, ok := err.(*googleapi.Error); ok && e.Code == http.StatusConflict {
			// Bucket already exists; ignore.
		} else {
			return errors.WithMessage(err, "failed to create GCS bucket")
		}
	}

	for _, artifact := range artifacts {
		log.Printf("# - %s (%.1f MB)", artifact.filename, float64(len(artifact.data))/(1024*1024))
		url, err := uploadArtifact(ctx, bkt, artifact.filename, artifact.data)
		if err != nil {
			return errors.WithMessage(err, "failed to upload artifact")
		}
		log.Printf("#   %s", url)
	}

	return nil
}

func uploadArtifact(ctx context.Context, bkt *storage.BucketHandle, name string, data []byte) (string, error) {
	obj := bkt.Object(name)
	w := obj.NewWriter(ctx)
	if _, err := w.Write(data); err != nil {
		return "", err
	}
	if err := w.Close(); err != nil {
		return "", err
	}

	if _, err := obj.Update(ctx, storage.ObjectAttrsToUpdate{
		PredefinedACL: "publicRead",
		ContentType:   "application/zip",
	}); err != nil {
		return "", err
	}

	attrs, err := obj.Attrs(ctx)
	if err != nil {
		return "", err
	}

	url := "https://storage.googleapis.com/" + attrs.Bucket + "/" + attrs.Name
	return url, nil
}
