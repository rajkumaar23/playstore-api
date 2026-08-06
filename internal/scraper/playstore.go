package scraper

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"playstore-api/internal/models"
	"regexp"
	"strings"
	"time"
)

type PlaystoreScraper struct {
	httpClient *http.Client
}

// Upper bound on a single Play Store fetch, in case the caller's context
// carries no deadline of its own.
const fetchTimeout = 15 * time.Second

// The payload lives in an inline script that calls AF_initDataCallback with
// key ds:5. Locating that call directly is enough - the surrounding document
// is never needed - so the marker doubles as the start of the regex match.
const dsDataMarker = "AF_initDataCallback({key: 'ds:5'"

const scriptCloseTag = "</script>"

// Compiled once: this used to be rebuilt on every parse. The capture stays
// greedy, so it must only ever be applied to a single script's contents.
var dsDataPattern = regexp.MustCompile(`AF_initDataCallback\({key: 'ds:5', hash: '[^']*', data:(.*), sideChannel: {}}\);`)

func NewPlaystoreScraper() *PlaystoreScraper {
	return &PlaystoreScraper{
		httpClient: &http.Client{Timeout: fetchTimeout},
	}
}

func (s *PlaystoreScraper) FetchHTML(ctx context.Context, packageID, gl string) (string, int, error) {
	playstoreURL := fmt.Sprintf("https://play.google.com/store/apps/details?id=%s&gl=%s", packageID, gl)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, playstoreURL, nil)
	if err != nil {
		return "", http.StatusInternalServerError, fmt.Errorf("error building request for '%s': %w", playstoreURL, err)
	}

	res, err := s.httpClient.Do(req)
	if err != nil {
		return "", http.StatusInternalServerError, fmt.Errorf("error making GET request to '%s': %w", playstoreURL, err)
	}

	defer res.Body.Close()
	if res.StatusCode != 200 {
		return "", res.StatusCode, fmt.Errorf("received non-200 status code for '%s': %s", playstoreURL, res.Status)
	}

	// Read straight into a builder rather than via io.ReadAll: its String()
	// hands back the accumulated bytes without the extra full-size copy that
	// converting a []byte to string would cost on a multi-megabyte page.
	var body strings.Builder
	if res.ContentLength > 0 {
		body.Grow(int(res.ContentLength))
	}

	if _, err := io.Copy(&body, res.Body); err != nil {
		return "", http.StatusInternalServerError, fmt.Errorf("error reading playstore response for '%s': %w", playstoreURL, err)
	}

	return body.String(), http.StatusOK, nil
}

func (s *PlaystoreScraper) Parse(packageID, html string) (*models.PlaystoreData, error) {
	script, err := s.findDataScript(html)
	if err != nil {
		return nil, fmt.Errorf("%w for id = %s", err, packageID)
	}

	extractedText, err := s.extractText(script)
	if err != nil {
		return nil, fmt.Errorf("regex matching failed for id = %s: %w", packageID, err)
	}

	var data []interface{}
	if err := json.Unmarshal([]byte(extractedText), &data); err != nil {
		return nil, fmt.Errorf("json unmarshal failed for id = %s: %w", packageID, err)
	}

	return models.NewPlaystoreData(packageID, data), nil
}

// findDataScript returns the contents of the inline script holding the ds:5
// payload, as a slice of the original document rather than a copy.
//
// Narrowing to that one script matters beyond avoiding the DOM: the capture in
// dsDataPattern is greedy, so running it across a whole minified document
// could swallow everything up to a later AF_initDataCallback call on the same
// line. Bounding the input at the enclosing </script> keeps the match scoped
// the way iterating over parsed script nodes used to.
func (s *PlaystoreScraper) findDataScript(html string) (string, error) {
	start := strings.Index(html, dsDataMarker)
	if start == -1 {
		return "", fmt.Errorf("failed to find <script> tag in HTML")
	}

	script := html[start:]
	if end := strings.Index(script, scriptCloseTag); end != -1 {
		script = script[:end]
	}

	return script, nil
}

func (s *PlaystoreScraper) extractText(input string) (string, error) {
	matches := dsDataPattern.FindStringSubmatch(input)
	expectedMatchCount := 2
	if len(matches) < expectedMatchCount {
		return "", fmt.Errorf("failed to find %d matches, found %d", expectedMatchCount, len(matches))
	}

	return matches[1], nil
}
