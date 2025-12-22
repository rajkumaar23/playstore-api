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

	"github.com/PuerkitoBio/goquery"
)

type PlaystoreScraper struct {
	httpClient *http.Client
}

func NewPlaystoreScraper() *PlaystoreScraper {
	return &PlaystoreScraper{
		httpClient: &http.Client{},
	}
}

func (s *PlaystoreScraper) FetchHTML(ctx context.Context, packageID, gl string) (string, int, error) {
	playstoreURL := fmt.Sprintf("https://play.google.com/store/apps/details?id=%s&gl=%s", packageID, gl)
	res, err := http.Get(playstoreURL)
	if err != nil {
		return "", http.StatusInternalServerError, fmt.Errorf("error making GET request to '%s': %w", playstoreURL, err)
	}

	defer res.Body.Close()
	if res.StatusCode != 200 {
		return "", res.StatusCode, fmt.Errorf("received non-200 status code for '%s': %s", playstoreURL, res.Status)
	}

	bodyBytes, err := io.ReadAll(res.Body)
	if err != nil {
		return "", http.StatusInternalServerError, fmt.Errorf("error reading playstore response for '%s': %w", playstoreURL, err)
	}

	return string(bodyBytes), http.StatusOK, nil
}

func (s *PlaystoreScraper) Parse(packageID, html string) (*models.PlaystoreData, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil, fmt.Errorf("error initialising goquery for id = %s: %w", packageID, err)
	}

	scriptSelector := doc.Find("script")
	for i := range scriptSelector.Nodes {
		scriptElement := scriptSelector.Eq(i)
		if strings.Contains(scriptElement.Text(), "AF_initDataCallback({key: 'ds:5'") {
			extractedText, err := s.extractText(scriptElement.Text())
			if err != nil {
				return nil, fmt.Errorf("regex matching failed for id = %s: %w", packageID, err)
			}
			var data []interface{}
			err = json.Unmarshal([]byte(extractedText), &data)
			if err != nil {
				return nil, fmt.Errorf("json unmarshal failed for id = %s: %w", packageID, err)
			}

			return models.NewPlaystoreData(packageID, data), nil
		}
	}

	return nil, fmt.Errorf("failed to find <script> tag in HTML for id = %s", packageID)
}

func (s *PlaystoreScraper) extractText(input string) (string, error) {
	pattern := `AF_initDataCallback\({key: 'ds:5', hash: '[^']*', data:(.*), sideChannel: {}}\);`
	re, err := regexp.Compile(pattern)
	if err != nil {
		return "", fmt.Errorf("regex compilation failed: %w", err)
	}

	matches := re.FindStringSubmatch(input)
	expectedMatchCount := 2
	if len(matches) < expectedMatchCount {
		return "", fmt.Errorf("failed to find %d matches, found %d", expectedMatchCount, len(matches))
	}

	return matches[1], nil
}
