package api

import (
	"bytes"
	"encoding/json"
	"net/http"
)

// verbatimJSON is a gin.Renderer that serialises without Go's default HTML
// escaping. Play's own text legitimately contains <br>, & and similar;
// json.Marshal's \u003c translation leaks implementation detail into the
// public response and surprises consumers diffing against the store.
type verbatimJSON struct{ data any }

func (v verbatimJSON) Render(w http.ResponseWriter) error {
	v.WriteContentType(w)
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v.data); err != nil {
		return err
	}
	_, err := w.Write(bytes.TrimRight(buf.Bytes(), "\n"))
	return err
}

func (v verbatimJSON) WriteContentType(w http.ResponseWriter) {
	if w.Header().Get("Content-Type") == "" {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
	}
}
