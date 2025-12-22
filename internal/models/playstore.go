package models

import (
	"reflect"
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
		if strings.EqualFold(key, val.Type().Field(i).Tag.Get("json")) {
			return val.Type().Field(i).Tag.Get("shields"), val.Field(i).Interface().(string)
		}
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
			return currentData[index].(T)
		}
		currentData = currentData[index].([]interface{})
	}
	return zero
}

func parseScreenshots(data []interface{}) []string {
	var screenshots []string

	if len(data) > 1 && len(data[1].([]interface{})) > 2 && len(data[1].([]interface{})[2].([]interface{})) > 78 {
		screenshotData := data[1].([]interface{})[2].([]interface{})[78].([]interface{})[0].([]interface{})
		for _, item := range screenshotData {
			if len(item.([]interface{})) > 3 && len(item.([]interface{})[3].([]interface{})) > 2 {
				screenshot := item.([]interface{})[3].([]interface{})[2]
				screenshots = append(screenshots, screenshot.(string))
			}
		}
	}

	return screenshots
}
