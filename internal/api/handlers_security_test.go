package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"playstore-api/internal/config"
	"playstore-api/internal/models"
	"playstore-api/internal/scraper"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func init() { gin.SetMode(gin.TestMode) }

// fakeCache is an in-memory stand-in for the Redis cache.
type fakeCache struct {
	mu    sync.Mutex
	store map[string]string
}

func newFakeCache() *fakeCache { return &fakeCache{store: map[string]string{}} }

func (f *fakeCache) Get(_ context.Context, key string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if v, ok := f.store[key]; ok {
		return v, nil
	}
	return "", fmt.Errorf("cache miss: %s", key)
}

func (f *fakeCache) Set(_ context.Context, key, value string, _ time.Duration) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.store[key] = value
	return nil
}

func (f *fakeCache) Close() error { return nil }

// dataPayload is the minimal ds:5 array NewPlaystoreData understands:
// data[1][2][0][0] is the app name.
const dataPayload = `[null,[null,null,[["Test App"]]]]`

func pageHTML() string {
	return "<html><head><script>AF_initDataCallback({key: 'ds:5', hash: '1', data:" +
		dataPayload + ", sideChannel: {}});</script></head></html>"
}

type stubOrigin struct {
	mu      sync.Mutex
	dials   int
	handler http.HandlerFunc
}

func (o *stubOrigin) RoundTrip(req *http.Request) (*http.Response, error) {
	o.mu.Lock()
	o.dials++
	o.mu.Unlock()
	rec := httptest.NewRecorder()
	o.handler(rec, req)
	return rec.Result(), nil
}

func (o *stubOrigin) count() int {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.dials
}

// newTestHandler wires the real handler and scraper against a stub origin
// that receives requests with the play.google.com URL intact.
func newTestHandler(t *testing.T, origin http.HandlerFunc) (*Handler, *fakeCache, *stubOrigin) {
	t.Helper()
	st := &stubOrigin{handler: origin}
	s := scraper.NewPlaystoreScraper().WithTransport(st)
	fc := newFakeCache()
	h := NewHandler(s, fc, &config.Config{DefaultGeoLocation: "US", CacheTTL: time.Hour})
	return h, fc, st
}

func newGET(path string) (*httptest.ResponseRecorder, *gin.Context) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, path, nil)
	return w, c
}

// The selector that ends up in the upstream URL must be a strict allowlist:
// delimiter characters that could reshape the outbound request are rejected
// outright, and a malformed locale falls back to the configured default.
func TestGetAllDataRejectsMalformedSelectors(t *testing.T) {
	for _, id := range []string{
		"",
		"com.foo&hl=de",
		"com.foo#frag",
		"com/foo",
		"com.foo bar",
		"com.foo%",
		strings.Repeat("a", 201),
	} {
		h, _, st := newTestHandler(t, func(w http.ResponseWriter, r *http.Request) {
			t.Errorf("malformed id %q reached the origin", id)
		})
		w, c := newGET("/json?id=" + url.QueryEscape(id))
		h.GetAllData(c)
		if w.Code != http.StatusBadRequest {
			t.Errorf("id %q: expected 400, got %d", id, w.Code)
		}
		if st.count() != 0 {
			t.Errorf("id %q: origin was dialed", id)
		}
	}
}

func TestMalformedGeoFallsBackToDefault(t *testing.T) {
	var gotGL string
	h, _, _ := newTestHandler(t, func(w http.ResponseWriter, r *http.Request) {
		gotGL = r.URL.Query().Get("gl")
		w.Write([]byte(pageHTML()))
	})
	w, c := newGET("/json?id=com.example&gl=USA;DROP")
	h.GetAllData(c)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if gotGL != "US" {
		t.Errorf("malformed gl was forwarded upstream: got %q want US", gotGL)
	}
}

// The (packageID, gl) -> key mapping must be injective. Under the old "id-gl"
// join these two selectors collided on one key, letting one poison the
// other's cached entry (and badge) for the whole TTL.
func TestCacheKeyIsInjective(t *testing.T) {
	if cacheKey("com.foo", "bar-US") == cacheKey("com.foo-bar", "US") {
		t.Fatal("colliding selectors still share a cache key")
	}
	// Brute-force sanity over adversarial short components.
	parts := []string{"", "a", "a-", "-a", "a-a", "US", "aUS", "b"}
	seen := map[string][2]string{}
	for _, id := range parts {
		for _, gl := range parts {
			k := cacheKey(id, gl)
			if prior, dup := seen[k]; dup && prior != [2]string{id, gl} {
				t.Fatalf("collision: %v and %v both map to %q", prior, [2]string{id, gl}, k)
			}
			seen[k] = [2]string{id, gl}
		}
	}
}

func TestCacheCollisionServesOwnEntry(t *testing.T) {
	h, fc, st := newTestHandler(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(pageHTML()))
	})
	// Attacker entry under the selector the old join aliased against the victim.
	attacker, _ := json.Marshal(map[string]any{"packageID": "com.foo", "name": "Attacker Badge"})
	fc.store[cacheKey("com.foo", "bar-US")] = string(attacker)

	w, c := newGET("/json?id=com.foo-bar&gl=US")
	h.GetAllData(c)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp models.PlaystoreData
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("bad response: %v", err)
	}
	if resp.PackageID != "com.foo-bar" {
		t.Fatalf("victim was served the colliding entry: %+v", resp)
	}
	if st.count() != 1 {
		t.Errorf("expected one origin fetch for the victim, got %d", st.count())
	}
	if fc.store[cacheKey("com.foo", "bar-US")] != string(attacker) {
		t.Error("victim request overwrote the attacker's entry")
	}
}

// A failed scrape is remembered briefly so a persistently failing key does
// not re-buy a full upstream fetch on every request.
func TestFailureMarksNegativeCache(t *testing.T) {
	h, fc, st := newTestHandler(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	w, c := newGET("/json?id=com.gone&gl=US")
	h.GetAllData(c)
	if w.Code == http.StatusOK {
		t.Fatal("failing scrape reported success")
	}
	if _, ok := fc.store[negativeCachePrefix+cacheKey("com.gone", "US")]; !ok {
		t.Fatal("failed scrape was not negative-cached")
	}

	w2, c2 := newGET("/json?id=com.gone&gl=US")
	h.GetAllData(c2)
	if w2.Code != http.StatusBadGateway {
		t.Errorf("negative-cached key: expected 502, got %d", w2.Code)
	}
	if st.count() != 1 {
		t.Errorf("negative cache did not short-circuit: origin dialed %d times", st.count())
	}
}

func TestNegativeCacheShortCircuitsBeforeScrape(t *testing.T) {
	h, fc, st := newTestHandler(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("origin dialed despite negative cache entry")
	})
	fc.store[negativeCachePrefix+cacheKey("com.bad", "US")] = "1"
	w, c := newGET("/json?id=com.bad&gl=US")
	h.GetAllData(c)
	if w.Code != http.StatusBadGateway {
		t.Fatalf("expected 502, got %d", w.Code)
	}
	if st.count() != 0 {
		t.Errorf("origin dialed %d times", st.count())
	}
}

// Scrapes are detached from the caller, so they must be bounded: once the
// scrape slots are taken, new work is shed with 429 instead of accumulating.
func TestScrapeSemaphoreShedsWhenSaturated(t *testing.T) {
	h, _, _ := newTestHandler(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(pageHTML()))
	})
	for i := 0; i < maxConcurrentScrapes; i++ {
		h.scrapeSem <- struct{}{}
	}
	start := time.Now()
	w, c := newGET("/json?id=com.saturated&gl=US")
	h.GetAllData(c)
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("saturated service accepted unbounded work: got %d", w.Code)
	}
	if time.Since(start) > upstreamWaitTimeout+time.Second {
		t.Errorf("shedding took %s; expected the wait to be capped at %s", time.Since(start), upstreamWaitTimeout)
	}
}

// Upstream shape deviations must degrade to zero values, never panic the
// process from the request path.
func TestShapeDeviationDoesNotPanic(t *testing.T) {
	// name arrives as a JSON number instead of a string.
	badPayload := `[null,[null,null,[[123]]]]`
	h, _, _ := newTestHandler(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("<html><head><script>AF_initDataCallback({key: 'ds:5', hash: '1', data:" +
			badPayload + ", sideChannel: {}});</script></head></html>"))
	})
	w, c := newGET("/json?id=com.wrongtype&gl=US")
	h.GetAllData(c)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 with zeroed field, got %d: %s", w.Code, w.Body.String())
	}
	var resp models.PlaystoreData
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.Name != "" {
		t.Errorf("type-deviated field not zeroed: %q", resp.Name)
	}
}

// GetField used to assert every field to string: keys backed by non-string
// fields (downloadsExact float, screenshots slice) panicked.
func TestGetDataByKeyNonStringFields(t *testing.T) {
	h, fc, _ := newTestHandler(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("cache was warm; origin should not be dialed")
	})
	cached, _ := json.Marshal(map[string]any{
		"packageID":      "com.foo",
		"name":           "N",
		"downloadsExact": 1234567,
		"screenshots":    []string{"a", "b"},
	})
	fc.store[cacheKey("com.foo", "US")] = string(cached)

	cases := []struct {
		key, wantMessage string
	}{
		{"downloadsexact", "1234567"},
		{"screenshots", "a, b"},
		{"name", "N"},
	}
	for _, tc := range cases {
		w, c := newGET("/x?id=com.foo&gl=US")
		c.Params = gin.Params{{Key: "key", Value: tc.key}}
		h.GetDataByKey(c)
		if w.Code != http.StatusOK {
			t.Fatalf("key %q: expected 200, got %d: %s", tc.key, w.Code, w.Body.String())
		}
		var badge struct {
			Label   string `json:"label"`
			Message string `json:"message"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &badge); err != nil {
			t.Fatalf("key %q: %v", tc.key, err)
		}
		if badge.Message != tc.wantMessage {
			t.Errorf("key %q: message %q want %q", tc.key, badge.Message, tc.wantMessage)
		}
	}

	w, c := newGET("/x?id=com.foo&gl=US")
	c.Params = gin.Params{{Key: "key", Value: "nosuchkey"}}
	h.GetDataByKey(c)
	if w.Code != http.StatusNotFound {
		t.Errorf("unknown key: expected 404, got %d", w.Code)
	}
}
