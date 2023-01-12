package outbound

import (
	"context"
	"net"
	"net/url"
	"regexp"
	"strings"

	"github.com/sourcegraph/sourcegraph/internal/database"
	"github.com/sourcegraph/sourcegraph/internal/database/basestore"
	"github.com/sourcegraph/sourcegraph/internal/encryption"
	"github.com/sourcegraph/sourcegraph/internal/encryption/keyring"
	"github.com/sourcegraph/sourcegraph/lib/errors"
)

type OutboundWebhookService interface {
	// EnqueueWebhook creates an outbound webhook job for the given webhook
	// event type, optional scope, and payload. In the normal course of events,
	// this will be picked up by the outbound webhook sender worker in short
	// order, and the webhook will be dispatched to any registered webhooks that
	// match the given type and scope.
	Enqueue(ctx context.Context, eventType string, scope *string, payload []byte) error
}

type outboundWebhookService struct {
	store database.OutboundWebhookJobStore
}

// NewOutboundWebhookService instantiates a new outbound webhook service. If key
// is nil, then the outbound webhook key will be used from the default keyring.
func NewOutboundWebhookService(db basestore.ShareableStore, key encryption.Key) OutboundWebhookService {
	if key == nil {
		key = keyring.Default().OutboundWebhookKey
	}

	return &outboundWebhookService{
		store: database.OutboundWebhookJobsWith(db, key),
	}
}

func (s *outboundWebhookService) Enqueue(
	ctx context.Context,
	eventType string,
	scope *string,
	payload []byte,
) error {
	if _, err := s.store.Create(ctx, eventType, scope, payload); err != nil {
		return errors.Wrap(err, "creating webhook job")
	}

	return nil
}

// Based on https://www.ietf.org/archive/id/draft-chapin-rfc2606bis-00.html
const reservedTLDs = "localhost|local|test|example|invalid|localdomain|domain|lan|home|host|corp"

// CheckAddress validates the intended destination address for a webhook, checking that
// it's not invalid, local, a bad IP, or anything else.
func CheckAddress(address string) error {
	// Try to interpret address as a URL, as an IP with a port, or as an IP without a port.
	u, uErr := url.Parse(address)
	ipStr, _, _ := net.SplitHostPort(address)
	ip1 := net.ParseIP(ipStr)
	ip2 := net.ParseIP(address)

	if ip1 != nil || ip2 != nil {
		// Address is likely an IP address
		var ip net.IP
		if ip1 != nil {
			ip = ip1
		} else {
			ip = ip2
		}

		if ip.To4() == nil && ip.To16() == nil {
			return errors.New("Not a valid IPv4 or IPv6 address")
		}

		if ip.IsUnspecified() {
			return errors.New("Must not be an unspecified IP address")
		}

		if ip.IsPrivate() {
			return errors.New("Must not be a private IP address")
		}

		if ip.IsLinkLocalMulticast() || ip.IsLinkLocalUnicast() {
			return errors.New("Must not be a link-local IP address")
		}

		return nil
	} else if uErr != nil {
		return errors.New("Could not parse address")
	} else {
		// Address is likely a URL
		if u.Scheme != "http" && u.Scheme != "https" {
			return errors.New("Must use http or https scheme")
		}

		if u.Hostname() == "" || u.Hostname() == "localhost" {
			return errors.New("Must have a non-local hostname")
		}

		parts := strings.Split(u.Hostname(), ".")
		tld := strings.ToLower(parts[len(parts)-1])
		if match, _ := regexp.MatchString(reservedTLDs, tld); match {
			return errors.New("Must not be a reserved TLD")
		}

		return nil
	}
}
