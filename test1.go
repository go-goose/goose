package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
)

const (
	OS_AUTH_URL    = "https://keystone.canonistack.canonical.com:443"
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
	password = getConfig("OS_PASSWORD", "NOVA_PASSWORD")
	project = getConfig("OS_TENANT_NAME", "NOVA_PROJECT_ID")
	return
}

type Credentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type Auth struct {
	Credentials Credentials `json:"passwordCredentials"`
	TenantName string `json:"tenantName"`
}

func NewAuthRequest(username, password, project string) interface{} {
	creds := struct{
		Username string
		Password string
	}{username, password}
	auth := struct{
		Credentials interface{}
		TenantName string
	}{creds, project}
	return struct {
		Auth interface{}
	}{auth}
}

func main() {
	username, password, project := getCredentials()
	client := &http.Client{CheckRedirect: nil}
	url := OS_AUTH_URL + OS_AUTH_TOKENS
	rq := StringMap{
		"auth": Auth {
			Credentials: Credentials{
				Username: username,
				Password: password,
			},
			TenantName: project,
		},
	}
	jrq, err := json.Marshal(rq)
	if err != nil {
		panic("error marshalling body: " + err.Error())
	}
	fmt.Println(string(jrq))
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
	//var jsonResp StringMap
	var jsonResp struct {
		Access struct {
			User struct {
				Id string
				Username string
				Roles []struct {
					Name string
				}
			}
		}
	}
	err = json.Unmarshal(respBody, &jsonResp)
	if err != nil {
		panic("error unmarshalling response body: " + err.Error())
	}
	fmt.Printf("\n\nGot response:\n%+v\n", jsonResp)
	for _, k := range jsonResp.Access.User.Roles {
		fmt.Println(k)
	}
}
