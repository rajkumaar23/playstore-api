package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

const apiURL = "https://play.rajkumaar.co.in/json?id=com.dd.doordash"

func isValidDate(date string) error {
	_, err := time.Parse("Jan 2, 2006", date)
	return err
}

func TestAPIResponse(t *testing.T) {
	resp, err := http.Get(apiURL)
	if err != nil {
		t.Errorf("Failed to fetch API data: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("API request failed with status code: %d", resp.StatusCode)
		return
	}

	var data playstoreDataResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		t.Errorf("Failed to decode JSON response: %v", err)
		return
	}

	// Assertions for each field
	assert.Equal(t, data.PackageID, "com.dd.doordash", "Package ID mismatch")
	assert.Equal(t, data.Name, "DoorDash - Food Delivery", "Name mismatch")
	assert.GreaterOrEqual(t, data.Version, "15.188.6", "Version mismatch")
	assert.Equal(t, data.Downloads, "50,000,000+", "Downloads mismatch")
	assert.GreaterOrEqual(t, data.DownloadsExact, 67209570.0, "Downloads Exact mismatch")

	err = isValidDate(data.LastUpdated)
	assert.Nil(t, err, fmt.Errorf("Invalid date format for lastUpdated: %v", err))

	assert.Equal(t, data.LaunchDate, "Mar 26, 2015", "Launch date mismatch")
	assert.Equal(t, data.Developer, "DoorDash", "Developer mismatch")
	assert.True(t, strings.Contains(data.Description, "Delivery anywhere you are."), "Description mismatch")
	assert.Equal(t, data.Category, "Food \u0026 Drink", "Category mismatch")

	_, err = url.ParseRequestURI(data.Logo)
	assert.Nil(t, err, "Invalid URL for logo")

	_, err = url.ParseRequestURI(data.Banner)
	assert.Nil(t, err, "Invalid URL for banner")

	for _, s := range data.Screenshots {
		_, err = url.ParseRequestURI(s)
		assert.Nil(t, err, fmt.Sprintf("Invalid URL for screenshot: %s", s))
	}

	assert.Equal(t, data.PrivacyPolicy, "https://www.doordash.com/privacy/", "Privacy policy mismatch")

	assert.Equal(t, data.Website, "https://www.doordash.com/", "Website mismatch")
	assert.Equal(t, data.SupportEmail, "support@doordash.com", "Support email mismatch")
	assert.True(t, data.Rating > "0.0" && data.Rating < "5.0", "Rating mismatch")
	assert.GreaterOrEqual(t, data.NoOfUsersRated, "4,712,796", "No of users rated mismatch")
}
