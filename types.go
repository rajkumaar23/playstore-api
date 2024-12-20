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
	return &playstoreDataResponse{
		PackageID:           packageID,
		LaunchDate:          getStringFromData(data, 1, 2, 10, 0),
		Name:                getStringFromData(data, 1, 2, 0, 0),
		Category:            getStringFromData(data, 1, 2, 79, 0, 0, 0),
		Developer:           getStringFromData(data, 1, 2, 37, 0),
		Description:         getStringFromData(data, 1, 2, 72, 0, 1),
		Downloads:           getStringFromData(data, 1, 2, 13, 0),
		Logo:                getStringFromData(data, 1, 2, 95, 0, 3, 2),
		Banner:              getStringFromData(data, 1, 2, 96, 0, 3, 2),
		PrivacyPolicy:       getStringFromData(data, 1, 2, 99, 0, 5, 2),
		LastUpdated:         getStringFromData(data, 1, 2, 145, 0, 0),
		LatestUpdateMessage: getStringFromData(data, 1, 2, 144, 1, 1),
		Version:             getStringFromData(data, 1, 2, 140, 0, 0, 0),
		Website:             getStringFromData(data, 1, 2, 69, 0, 5, 2),
		SupportEmail:        getStringFromData(data, 1, 2, 69, 1, 0),
		Screenshots:         parseScreenshots(data),
		DownloadsExact:      getFloat64FromData(data, 1, 2, 13, 2),
		Rating:              getStringFromData(data, 1, 2, 51, 0, 0),
		NoOfUsersRated:      getStringFromData(data, 1, 2, 51, 2, 0),
	}
}

func getStringFromData(data []interface{}, indices ...int) string {
	var currentData []interface{} = data
	for i, index := range indices {
		if currentData == nil || currentData[index] == nil || index >= len(currentData) {
			return ""
		}
		if i+1 == len(indices) {
			return currentData[index].(string)
		}
		currentData = currentData[index].([]interface{})
	}
	return ""
}

func getFloat64FromData(data []interface{}, indices ...int) float64 {
	var currentData []interface{} = data
	for i, index := range indices {
		if currentData == nil || currentData[index] == nil || index >= len(currentData) {
			return 0
		}
		if i+1 == len(indices) {
			return currentData[index].(float64)
		}
		currentData = currentData[index].([]interface{})
	}
	return 0
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
