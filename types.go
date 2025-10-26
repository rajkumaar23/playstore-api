package main

type playstoreDataResponse struct {
	PackageID           string   `json:"packageID" api:"Package ID"`
	Name                string   `json:"name" api:"Name"`
	Version             string   `json:"version" api:"Version"`
	Downloads           string   `json:"downloads" api:"Downloads"`
	DownloadsExact      float64  `json:"downloadsExact" api:"Downloads"`
	LastUpdated         string   `json:"lastUpdated" api:"Last Updated On"`
	LaunchDate          string   `json:"launchDate" api:"Launched On"`
	Developer           string   `json:"developer" api:"Developer"`
	Description         string   `json:"description" api:"Description"`
	Screenshots         []string `json:"screenshots" api:"Screenshots"`
	Category            string   `json:"category" api:"Category"`
	Logo                string   `json:"logo" api:"Logo"`
	Banner              string   `json:"banner" api:"Banner"`
	PrivacyPolicy       string   `json:"privacyPolicy" api:"Privacy Policy"`
	LatestUpdateMessage string   `json:"latestUpdateMessage" api:"Latest Update Message"`
	Website             string   `json:"website" api:"Website"`
	SupportEmail        string   `json:"supportEmail" api:"Support Email"`
	Rating              string   `json:"rating" api:"Rating"`
	NoOfUsersRated      string   `json:"noOfUsersRated" api:"No of users rated"`
}

func newPlaystoreDataResponse(packageID string, data []interface{}) *playstoreDataResponse {
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

	return &playstoreDataResponse{
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
