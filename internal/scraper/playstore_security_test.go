package scraper

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
)

// rewriteTransport sends every outbound request to a local stub server while
// leaving the request URL (play.google.com) intact, so FetchHTML's URL
// construction can be asserted end to end without touching the network.
type rewriteTransport struct {
	target string // host:port of the stub server
	seen   []*http.Request
}

func (t *rewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	clone := req.Clone(req.Context())
	clone.URL.Scheme = "http"
	clone.URL.Host = t.target
	t.seen = append(t.seen, req)
	return http.DefaultTransport.RoundTrip(clone)
}

func stubScraper(t *testing.T, handler http.HandlerFunc) (*PlaystoreScraper, *rewriteTransport) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	s := NewPlaystoreScraper()
	rt := &rewriteTransport{target: strings.TrimPrefix(srv.URL, "http://")}
	s.httpClient.Transport = rt
	return s, rt
}

// The only selectors in the upstream URL are attacker-supplied query params.
// They must be percent-encoded so injected delimiters ('&', '#') cannot add
// parameters of their own to the request sent to play.google.com.
func TestFetchHTMLEscapesSelectors(t *testing.T) {
	s, rt := stubScraper(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("<html>ok</html>"))
	})

	malicious := "com.example&hl=de&x=#y"
	html, code, err := s.FetchHTML(context.Background(), malicious, "US&extra=1")
	if err != nil || code != http.StatusOK {
		t.Fatalf("fetch failed: code=%d err=%v", code, err)
	}
	if len(rt.seen) != 1 {
		t.Fatalf("expected exactly one upstream request, got %d", len(rt.seen))
	}
	req := rt.seen[0]
	if req.URL.Host != "play.google.com" || req.URL.Scheme != "https" {
		t.Fatalf("request escaped its intended origin: %s://%s", req.URL.Scheme, req.URL.Host)
	}
	q, err := url.ParseQuery(req.URL.RawQuery)
	if err != nil {
		t.Fatalf("unparseable query: %v", err)
	}
	if got := q.Get("id"); got != malicious {
		t.Errorf("id did not round-trip verbatim: got %q want %q", got, malicious)
	}
	if got := q.Get("gl"); got != "US&extra=1" {
		t.Errorf("gl did not round-trip verbatim: got %q", got)
	}
	if _, ok := q["hl"]; ok {
		t.Errorf("injected parameter 'hl' reached the upstream query: %q", req.URL.RawQuery)
	}
	if _, ok := q["x"]; ok {
		t.Errorf("injected parameter 'x' reached the upstream query: %q", req.URL.RawQuery)
	}
	if strings.Contains(req.URL.RawQuery, "#") {
		t.Errorf("fragment delimiter left unescaped: %q", req.URL.RawQuery)
	}
	_ = html
}

// A forged Content-Length must not make the client pre-allocate (or trust)
// memory proportional to an advertised size beyond the bound.
func TestFetchHTMLRejectsForgedContentLength(t *testing.T) {
	s, _ := stubScraper(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", strconv.Itoa(300<<20)) // 300 MiB advertised
		w.Write([]byte("tiny body"))
	})

	_, code, err := s.FetchHTML(context.Background(), "com.example", "US")
	if err == nil {
		t.Fatal("forged 300 MiB Content-Length accepted")
	}
	if code != http.StatusBadGateway {
		t.Fatalf("expected 502 for oversized advertised body, got %d", code)
	}
}

// A chunked (unadvertised) body larger than the bound must be rejected, not
// buffered in full.
func TestFetchHTMLRejectsBodyOverCap(t *testing.T) {
	s, _ := stubScraper(t, func(w http.ResponseWriter, r *http.Request) {
		chunk := strings.Repeat("A", 1<<20)
		w.Header().Set("Transfer-Encoding", "chunked")
		for i := 0; i < (maxFetchBytes>>20)+2; i++ {
			if _, err := w.Write([]byte(chunk)); err != nil {
				return
			}
		}
	})

	_, code, err := s.FetchHTML(context.Background(), "com.example", "US")
	if err == nil {
		t.Fatal("body larger than maxFetchBytes accepted")
	}
	if code != http.StatusBadGateway {
		t.Fatalf("expected 502 for body over cap, got %d", code)
	}
}

func TestFetchHTMLAcceptsBodyUnderCap(t *testing.T) {
	s, _ := stubScraper(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(strings.Repeat("A", (6<<20)+1))) // 6 MiB + 1 byte
	})

	_, code, err := s.FetchHTML(context.Background(), "com.example", "US")
	if err != nil || code != http.StatusOK {
		t.Fatalf("legitimate multi-MB page rejected: code=%d err=%v", code, err)
	}
}

// The single outbound request must never be redirected off play.google.com
// or downgraded to http.
func TestPinUpstream(t *testing.T) {
	via := []*http.Request{{URL: &url.URL{Scheme: "https", Host: "play.google.com"}}}
	tests := []struct {
		name string
		req  *http.Request
		ok   bool
	}{
		{"same origin https", &http.Request{URL: &url.URL{Scheme: "https", Host: "play.google.com"}}, true},
		{"off-host https", &http.Request{URL: &url.URL{Scheme: "https", Host: "evil.example"}}, false},
		{"downgrade to http", &http.Request{URL: &url.URL{Scheme: "http", Host: "play.google.com"}}, false},
		{"subdomain is off-host", &http.Request{URL: &url.URL{Scheme: "https", Host: "attacker.play.google.com"}}, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := pinUpstream(tc.req, via)
			if tc.ok && err != nil {
				t.Fatalf("legitimate redirect blocked: %v", err)
			}
			if !tc.ok && err != http.ErrUseLastResponse {
				t.Fatalf("hostile redirect allowed (err=%v)", err)
			}
		})
	}

	if NewPlaystoreScraper().httpClient.CheckRedirect == nil {
		t.Fatal("production client carries no redirect policy")
	}
}
