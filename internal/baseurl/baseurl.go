package baseurl

import "os"

var baseUrl string = "http://localhost:8080"

func init() {
	envUrl, found := os.LookupEnv("SX_BASE_URL")
	if found {
		baseUrl = envUrl
	}
}

func GetBaseUrl() string {
	return baseUrl
}
