package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

// The public JSON must carry Play's literal text: descriptions contain
// <br> and & and must not arrive as \u003cbr\u003e escapes.
func TestJSONResponsesKeepHTMLCharacters(t *testing.T) {
	h, fc, _ := newTestHandler(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(pageHTML()))
	})
	const desc = "line1<br><b>line2</b> & more"
	payload, err := json.Marshal(map[string]any{
		"packageID":   "com.example.app",
		"name":        "Example & Co",
		"category":    "Food & Drink",
		"description": desc,
	})
	if err != nil {
		t.Fatal(err)
	}
	fc.store[cacheKey("com.example.app", "US")] = string(payload)

	w, c := newGET("/json?id=com.example.app")
	h.GetAllData(c)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	for _, esc := range []string{`\u003c`, `\u003e`, `\u0026`} {
		if strings.Contains(strings.ToLower(body), esc) {
			t.Fatalf("response still HTML-escapes (%s): %s", esc, body)
		}
	}
	if !strings.Contains(body, desc) {
		t.Fatalf("literal markup missing from response: %s", body)
	}
	var got map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}
	if got["description"] != desc {
		t.Fatalf("description changed in transit: %v", got["description"])
	}
	if ct := w.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Fatalf("content-type lost: %q", ct)
	}

	// The shields-format endpoint shares the renderer: an ampersand in a
	// category must reach the badge payload unescaped.
	w, c = newGET("/category?id=com.example.app")
	c.Params = gin.Params{{Key: "key", Value: "category"}}
	h.GetDataByKey(c)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"message":"Food & Drink"`) {
		t.Fatalf("shields payload escaped: %s", w.Body.String())
	}
}
