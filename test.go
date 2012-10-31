package main

import (
	"os"
	"net/http"
	"io/ioutil"
	"fmt"
	"strings"
	"encoding/json"
)

const (
	OS_AUTH_URL = "https://keystone.canonistack.canonical.com:443"
	OS_AUTH_TOKENS = "/v2.0/tokens"
)

type StringMap map[string]interface{}

func getConfig(envVars ...string) (value string) {
	value = ""
	for _, v := range envVars {
		value = os.Getenv(v)
		if value != "" {
			break
		}
	}
	return
}

func getCredentials() (username, password, project string) {
	username = getConfig("OS_USERNAME", "NOVA_USERNAME")
	password  = getConfig("OS_PASSWORD", "NOVA_PASSWORD")
	project  = getConfig("OS_TENANT_NAME", "NOVA_PROJECT_ID")
	return
}

func main() {
	username, password, project := getCredentials()
	client := &http.Client{CheckRedirect: nil}
	url := OS_AUTH_URL + OS_AUTH_TOKENS
	jsonBody := StringMap{
		"auth": StringMap{
			"passwordCredentials": StringMap{
				"username": username,
				"password": password,
			},
			"tenantName": project,
		},
	}
	body, err := json.Marshal(jsonBody)
	if err != nil {
		panic("error marshalling body: " + err.Error())
	}
	bodyReader := strings.NewReader(string(body))
	req, err := http.NewRequest("POST", url, bodyReader)
	if err != nil {
		panic("cannot create request to: " + url + ": " + err.Error())
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Accept", "application/json")
	fmt.Println("Requesting an auth token from " + url)
	resp, err := client.Do(req)
	if err != nil {
		panic("cannot authenticate to: " + url + ": " + err.Error())
	}
	defer resp.Body.Close()
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic("error reading response body: " + err.Error())
	}
	var jsonResp StringMap
	err = json.Unmarshal(respBody, &jsonResp)
	if err != nil {
		panic("error unmarshalling response body: " + err.Error())
	}
	fmt.Printf("\n\nGot response:\n%+v\n", jsonResp)
}
