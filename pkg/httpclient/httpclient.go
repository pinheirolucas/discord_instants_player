// Package httpclient builds the client every request to myinstants.com goes
// through.
//
// The User-Agent it sets is load-bearing, not cosmetic: myinstants.com sits
// behind Cloudflare, which answers 403 to Go's default Go-http-client/2.0 (and to
// curl's, and to a bare Mozilla/5.0). A User-Agent that names this application
// gets through. Swapping this client back for http.DefaultClient breaks both the
// instant listing and the clip download.
package httpclient

import (
	"net/http"
	"time"
)

// UserAgent identifies the app to myinstants.com.
const UserAgent = "discord_instants_player/1.0"

const timeout = 30 * time.Second

// userAgentTransport sets UserAgent on requests that do not carry one, so a
// caller can still send its own.
type userAgentTransport struct {
	base http.RoundTripper
}

func (t *userAgentTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.Header.Get("User-Agent") != "" {
		return t.base.RoundTrip(req)
	}

	// A RoundTripper must not modify the request it is given.
	req = req.Clone(req.Context())
	req.Header.Set("User-Agent", UserAgent)

	return t.base.RoundTrip(req)
}

// New returns a client that identifies itself with UserAgent.
func New() *http.Client {
	return &http.Client{
		Transport: &userAgentTransport{base: http.DefaultTransport},
		Timeout:   timeout,
	}
}
