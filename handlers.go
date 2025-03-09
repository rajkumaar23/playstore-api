package main

import (
	"net/http"
	"os"
	"reflect"
	"strings"

	"github.com/gin-gonic/gin"
)

func getREADME(c *gin.Context) {
	readme, err := os.ReadFile("page.html")
	if err != nil {
		c.Data(http.StatusInternalServerError, "text/html; charset=utf-8", []byte("an internal error occurred"))
	}

	c.Data(http.StatusOK, "text/html; charset=utf-8", readme)
}

func getAllData(c *gin.Context) {
	packageID := c.Query("id")
	if packageID == "" {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"message": "package id is mandatory"})
		return
	}

	resBody, statusCode := fetchHTML(packageID)
	if statusCode == http.StatusNotFound {
		c.IndentedJSON(http.StatusNotFound, gin.H{"message": "package id is invalid"})
		return
	} else if statusCode != 200 {
		c.IndentedJSON(http.StatusBadGateway, gin.H{"message": "an internal error occurred"})
		return
	}

	parsedPlaystoreData, err := parsePlaystoreData(packageID, resBody)
	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"message": "an internal error occurred"})
		return
	}

	c.IndentedJSON(http.StatusOK, *parsedPlaystoreData)
}

func getDataByKey(c *gin.Context) {
	packageID := c.Query("id")
	if packageID == "" {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"message": "package id is mandatory"})
		return
	}
	resBody, statusCode := fetchHTML(packageID)
	if statusCode == http.StatusNotFound {
		c.IndentedJSON(http.StatusNotFound, gin.H{"message": "package id is invalid"})
		return
	} else if statusCode != 200 {
		c.IndentedJSON(http.StatusBadGateway, gin.H{"message": "an internal error occurred"})
		return
	}

	parsedPlaystoreData, err := parsePlaystoreData(packageID, resBody)
	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"message": "an internal error occurred"})
		return
	}

	key := c.Params.ByName("key")
	val := reflect.ValueOf(*parsedPlaystoreData)
	for i := 0; i < val.Type().NumField(); i++ {
		if strings.EqualFold(key, val.Type().Field(i).Tag.Get("json")) {
			c.IndentedJSON(http.StatusOK, gin.H{"schemaVersion": 1, "label": val.Type().Field(i).Tag.Get("api"), "message": val.Field(i).Interface()})
			return
		}
	}

	c.IndentedJSON(http.StatusBadRequest, gin.H{"message": "key is invalid"})
}
