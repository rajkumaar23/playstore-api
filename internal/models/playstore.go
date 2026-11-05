package models

import (
	"reflect"
	"strconv"
	"strings"
)

type PlaystoreData struct {
	PackageID           string   `json:"packageID" shields:"Package ID"`
	Name                string   `json:"name" shields:"Name"`
	Version             string   `json:"version" shields:"Version"`
	Downloads           string   `json:"downloads" shields:"Downloads"`
	DownloadsExact      float64  `json:"downloadsExact" shields:"Downloads"`
	LastUpdated         string   `json:"lastUpdated" shields:"Last Updated On"`
	LaunchDate          string   `json:"launchDate" shields:"Launched On"`
	Developer           string   `json:"developer" shields:"Developer"`
	Description         string   `json:"description" shields:"Description"`
	Screenshots         []string `json:"screenshots" shields:"Screenshots"`
	Category            string   `json:"category" shields:"Category"`
	Logo                string   `json:"logo" shields:"Logo"`
	Banner              string   `json:"banner" shields:"Banner"`
	PrivacyPolicy       string   `json:"privacyPolicy" shields:"Privacy Policy"`
	LatestUpdateMessage string   `json:"latestUpdateMessage" shields:"Latest Update Message"`
	Website             string   `json:"website" shields:"Website"`
	SupportEmail        string   `json:"supportEmail" shields:"Support Email"`
	Rating              string   `json:"rating" shields:"Rating"`
	NoOfUsersRated      string   `json:"noOfUsersRated" shields:"# of users rated"`
}

func NewPlaystoreData(packageID string, data []interface{}) *PlaystoreData {
	latestUpdateMessage := ""
	if val, ok := getAttributeFromData[map[string]interface{}](data, 1, 2, 112)["145"]; ok && val != nil {
		if v, ok := val.([]interface{}); ok {
			latestUpdateMessage = getAttributeFromData[string](v, 1, 1)
		}
	}
	if latestUpdateMessage == "" {
		latestUpdateMessage = getAttributeFromData[string](data, 1, 2, 144, 1, 1)
	}

	lastUpdated := ""
	if val, ok := getAttributeFromData[map[string]interface{}](data, 1, 2, 112)["146"]; ok && val != nil {
		if v, ok := val.([]interface{}); ok {
			lastUpdated = getAttributeFromData[string](v, 0, 0)
		}
	}
	if lastUpdated == "" {
		lastUpdated = getAttributeFromData[string](data, 1, 2, 145, 0, 0)
	}

	version := ""
	if val, ok := getAttributeFromData[map[string]interface{}](data, 1, 2, 112)["141"]; ok && val != nil {
		if v, ok := val.([]interface{}); ok {
			version = getAttributeFromData[string](v, 0, 0, 0)
		}
	}
	if version == "" {
		version = getAttributeFromData[string](data, 1, 2, 140, 0, 0, 0)
	}

	return &PlaystoreData{
		PackageID:           packageID,
		LaunchDate:          getAttributeFromData[string](data, 1, 2, 10, 0),
		Name:                getAttributeFromData[string](data, 1, 2, 0, 0),
		Category:            getAttributeFromData[string](data, 1, 2, 79, 0, 0, 0),
		Developer:           getAttributeFromData[string](data, 1, 2, 37, 0),
		Description:         getAttributeFromData[string](data, 1, 2, 72, 0, 1),
		Downloads:           getAttributeFromData[string](data, 1, 2, 13, 0),
		Logo:                getAttributeFromData[string](data, 1, 2, 95, 0, 3, 2),
		Banner:              getAttributeFromData[string](data, 1, 2, 96, 0, 3, 2),
		PrivacyPolicy:       getAttributeFromData[string](data, 1, 2, 99, 0, 5, 2),
		LatestUpdateMessage: latestUpdateMessage,
		LastUpdated:         lastUpdated,
		Version:             version,
		Website:             getAttributeFromData[string](data, 1, 2, 69, 0, 5, 2),
		SupportEmail:        getAttributeFromData[string](data, 1, 2, 69, 1, 0),
		Screenshots:         parseScreenshots(data),
		DownloadsExact:      getAttributeFromData[float64](data, 1, 2, 13, 2),
		Rating:              getAttributeFromData[string](data, 1, 2, 51, 0, 0),
		NoOfUsersRated:      getAttributeFromData[string](data, 1, 2, 51, 2, 0),
	}
}

func (p *PlaystoreData) GetField(key string) (string, string) {
	val := reflect.ValueOf(*p)
	for i := 0; i < val.Type().NumField(); i++ {
		if !strings.EqualFold(key, val.Type().Field(i).Tag.Get("json")) {
			continue
		}
		label := val.Type().Field(i).Tag.Get("shields")
		field := val.Field(i)
		switch field.Kind() {
		case reflect.String:
			return label, field.String()
		case reflect.Float64:
			return label, strconv.FormatFloat(field.Float(), 'f', -1, 64)
		case reflect.Slice:
			if field.Type().Elem().Kind() == reflect.String {
				parts := make([]string, 0, field.Len())
				for j := 0; j < field.Len(); j++ {
					parts = append(parts, field.Index(j).String())
				}
				return label, strings.Join(parts, ", ")
			}
			return "", ""
		}
		return "", ""
	}
	return "", ""
}

func getAttributeFromData[T any](data []interface{}, indices ...int) T {
	var currentData []interface{} = data
	var zero T
	for i, index := range indices {
		if currentData == nil || index >= len(currentData) || currentData[index] == nil {
			return zero
		}
		if i+1 == len(indices) {
			// Checked: an upstream shape change (number where a string is
			// expected, or vice versa) yields the zero value instead of
			// panicking inside the request path.
			v, ok := currentData[index].(T)
			if !ok {
				return zero
			}
			return v
		}
		next, ok := currentData[index].([]interface{})
		if !ok {
			return zero
		}
		currentData = next
	}
	return zero
}

// nestedSlice indexes v as a []interface{} element at index, reporting
// failure instead of panicking when upstream data deviates from the expected
// shape.
func nestedSlice(v interface{}, index int) ([]interface{}, bool) {
	s, ok := v.([]interface{})
	if !ok || index >= len(s) {
		return nil, false
	}
	t, ok := s[index].([]interface{})
	return t, ok
}

func parseScreenshots(data []interface{}) []string {
	var screenshots []string

	l1, ok := nestedSlice(data, 1)
	if !ok {
		return screenshots
	}
	l2, ok := nestedSlice(l1, 2)
	if !ok {
		return screenshots
	}
	l78, ok := nestedSlice(l2, 78)
	if !ok {
		return screenshots
	}
	l0, ok := nestedSlice(l78, 0)
	if !ok {
		return screenshots
	}
	for _, item := range l0 {
		i3, ok := nestedSlice(item, 3)
		if !ok || len(i3) <= 2 {
			continue
		}
		if screenshot, ok := i3[2].(string); ok {
			screenshots = append(screenshots, screenshot)
		}
	}

	return screenshots
}
