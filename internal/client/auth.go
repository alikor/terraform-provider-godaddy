package client

import "net/http"

type authTransport struct {
	base      http.RoundTripper
	apiKey    string
	apiSecret string
	userAgent string
}

func (t *authTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	clone := req.Clone(req.Context())
	clone.Header = clone.Header.Clone()
	clone.Header.Set("Authorization", "sso-key "+t.apiKey+":"+t.apiSecret)
	clone.Header.Set("Accept", "application/json")
	clone.Header.Set("Content-Type", "application/json")

	if t.userAgent != "" {
		clone.Header.Set("User-Agent", t.userAgent)
	}

	return t.base.RoundTrip(clone)
}
