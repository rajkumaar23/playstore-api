package api

import (
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
)

// The brand assets must be served with correct content types: browsers
// silently fall back to a generic tab icon when /favicon.svg 404s (the /:key
// route would answer with JSON), and social platforms refuse og:image
// payloads that aren't a real image/png.
func TestStaticBrandAssetsServed(t *testing.T) {
	h := &Handler{}
	cases := []struct {
		name        string
		call        func(*gin.Context)
		wantCT      string
		minBodySize int
	}{
		{"favicon.svg", h.GetFaviconSVG, "image/svg+xml", 100},
		{"apple-touch-icon.png", h.GetAppleTouchIcon, "image/png", 500},
		{"og-cover.png", h.GetOGCover, "image/png", 10000},
		{"favicon.ico", h.GetFavicon, "image/x-icon", 100},
	}
	for _, tc := range cases {
		w, c := newGET("/" + tc.name)
		tc.call(c)
		if w.Code != http.StatusOK {
			t.Errorf("%s: expected 200, got %d", tc.name, w.Code)
		}
		if ct := w.Header().Get("Content-Type"); ct != tc.wantCT {
			t.Errorf("%s: content-type %q, want %q", tc.name, ct, tc.wantCT)
		}
		if w.Body.Len() < tc.minBodySize {
			t.Errorf("%s: body %d bytes, want at least %d", tc.name, w.Body.Len(), tc.minBodySize)
		}
	}
}
