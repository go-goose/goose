// Nova double testing service - HTTP API implementation

package novaservice

import (
	"encoding/json"
	"io/ioutil"
	"launchpad.net/goose/nova"
	"net/http"
	"strings"
)

const authToken = "X-Auth-Token"

var versionPrefix = "v2"   // version part of endpoint URL
var tenantId = "tenant_id" // tenant ID part of the endpoint URL

// response defines a single HTTP response.
type response struct {
	code        int
	body        string
	contentType string
	errorText   string
}

// verbatim real Nova responses
var unauthorizedResponse = response{
	http.StatusUnauthorized,
	`401 Unauthorized

This server could not verify that you are authorized to access the document you requested. Either you supplied the wrong credentials (e.g., bad password), or your browser does not understand how to supply the credentials required.

 Authentication required
`,
	"text/plain; charset=UTF-8",
	"",
}
var forbiddenResponse = response{
	http.StatusForbidden,
	`{"forbidden": {"message": "Policy doesn't allow compute_extension:flavormanage to be performed.", "code": 403}}`,
	"application/json; charset=UTF-8",
	"",
}
var badRequestResponse = response{
	http.StatusBadRequest,
	`{"badRequest": {"message": "Malformed request url", "code": 400}}`,
	"application/json; charset=UTF-8",
	"",
}
var badRequest2Response = response{
	http.StatusBadRequest,
	`{"badRequest": {"message": "The server could not comply with the request since it is either malformed or otherwise incorrect.", "code": 400}}`,
	"application/json; charset=UTF-8",
	"",
}
var notFoundResponse = response{
	http.StatusNotFound,
	`404 Not Found

The resource could not be found.


`,
	"text/plain; charset=UTF-8",
	"",
}
var notFoundJSONResponse = response{
	http.StatusNotFound,
	`{"itemNotFound": {"message": "The resource could not be found.", "code": 404}}`,
	"application/json; charset=UTF-8",
	"",
}
var multipleChoicesResponse = response{
	http.StatusMultipleChoices,
	`{"choices": [{"status": "CURRENT", "media-types": [{"base": "application/xml", "type": "application/vnd.openstack.compute+xml;version=2"}, {"base": "application/json", "type": "application/vnd.openstack.compute+json;version=2"}], "id": "v2.0", "links": [{"href": "$ENDPOINT$$URL$", "rel": "self"}]}]}`,
	"application/json",
	"",
}
var noVersionResponse = response{
	http.StatusOK,
	`{"versions": [{"status": "CURRENT", "updated": "2011-01-21T11:33:21Z", "id": "v2.0", "links": [{"href": "$ENDPOINT$", "rel": "self"}]}]}`,
	"application/json",
	"",
}
var versionsLinksResponse = response{
	http.StatusOK,
	`{"version": {"status": "CURRENT", "updated": "2011-01-21T11:33:21Z", "media-types": [{"base": "application/xml", "type": "application/vnd.openstack.compute+xml;version=2"}, {"base": "application/json", "type": "application/vnd.openstack.compute+json;version=2"}], "id": "v2.0", "links": [{"href": "$ENDPOINT$", "rel": "self"}, {"href": "http://docs.openstack.org/api/openstack-compute/1.1/os-compute-devguide-1.1.pdf", "type": "application/pdf", "rel": "describedby"}, {"href": "http://docs.openstack.org/api/openstack-compute/1.1/wadl/os-compute-1.1.wadl", "type": "application/vnd.sun.wadl+xml", "rel": "describedby"}]}}`,
	"application/json",
	"",
}
var createdResponse = response{
	http.StatusCreated,
	"201 Created",
	"text/plain; charset=UTF-8",
	"",
}
var errorResponse = response{
	http.StatusInternalServerError,
	`{"internalServerError":{"message":"$ERROR$",code:500}}`,
	"application/json",
	"", // set by sendError()
}

// endpoint returns the current testing server's endpoint URL.
func endpoint() string {
	return hostname + versionPrefix
}

// replaceVars replaces any $<varname>$ chunks in the response body
// with their values, taking the original requset into account, and
// returns the result as a string.
func (resp response) replaceVars(r *http.Request) string {
	url := strings.TrimLeft(r.URL.Path, "/")
	body := resp.body
	body = strings.Replace(body, "$ENDPOINT$", endpoint(), 1)
	body = strings.Replace(body, "$URL$", url, 1)
	if resp.errorText != "" {
		body = strings.Replace(body, "$ERROR$", resp.errorText, 1)
	}
	return body
}

// send serializes the response as needed and sends it.
func (resp response) send(w http.ResponseWriter, r *http.Request) {
	if resp.contentType != "" {
		w.Header().Set("Content-Type", resp.contentType)
	}
	if resp.body == "" {
		// workaround https://code.google.com/p/go/issues/detail?id=4454
		w.Header().Set("Content-Length", "0")
	}
	if resp.code != 0 {
		w.WriteHeader(resp.code)
	}
	if resp.body != "" {
		body := resp.replaceVars(r)
		w.Write([]byte(body))
	}
}

// sendError is a shortcut to errorResponse.send(w, r), but correctly
// handling the error message.
func sendError(err error, w http.ResponseWriter, r *http.Request) {
	eresp := errorResponse
	eresp.errorText = err.Error()
	eresp.send(w, r)
}

// sendJSON sends the specified response serialized as JSON.
func sendJSON(code int, resp interface{}, w http.ResponseWriter, r *http.Request) {
	data := []byte{}
	if resp != nil {
		var err error
		data, err = json.Marshal(resp)
		if err != nil {
			sendError(err, w, r)
		}
	}
	if len(data) == 0 {
		// workaround https://code.google.com/p/go/issues/detail?id=4454
		w.Header().Set("Content-Length", "0")
	}
	w.WriteHeader(code)
	w.Write(data)
}

// handleFlavors provides the HTTP flavors API processing.
func (n *Nova) handleFlavors(method, cmd string, w http.ResponseWriter, r *http.Request) {
	flavors := n.allFlavors()
	if len(flavors) == 0 && method == "GET" {
		sendJSON(http.StatusNoContent, nil, w, r)
		return
	}
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		// not all requests have body, just ignore it
		body = nil
	}
	defer r.Body.Close()
	switch {
	case strings.HasPrefix(cmd, "flavors/detail"):
		switch method {
		case "GET":
			var resp struct {
				Flavors []nova.FlavorDetail `json:"flavors"`
			}
			resp.Flavors = flavors
			sendJSON(http.StatusOK, resp, w, r)
		case "POST":
			notFoundResponse.send(w, r)
		case "PUT":
			notFoundJSONResponse.send(w, r)
		case "DELETE":
			forbiddenResponse.send(w, r)
		default:
			panic("unknown request method: " + method)
		}
	case strings.HasPrefix(cmd, "flavors"):
		switch method {
		case "GET":
			if strings.Index(cmd, "/") != -1 {
				notFoundResponse.send(w, r)
			} else {
				var resp struct {
					Flavors []nova.Entity `json:"flavors"`
				}
				resp.Flavors = n.allFlavorsAsEntities()
				sendJSON(http.StatusOK, resp, w, r)
			}
		case "POST":
			if strings.Index(cmd, "/") != -1 {
				notFoundResponse.send(w, r)
			} else if len(body) == 0 {
				badRequest2Response.send(w, r)
			} else {
				var flavor struct {
					Flavor nova.FlavorDetail
				}
				err = json.Unmarshal(body, &flavor)
				if err != nil {
					sendError(err, w, r)
				}
				n.buildFlavorLinks(&flavor.Flavor)
				err = n.addFlavor(flavor.Flavor)
				if err != nil {
					sendError(err, w, r)
				}
				createdResponse.send(w, r)
			}
		case "PUT":
			fallthrough
		case "DELETE":
			notFoundResponse.send(w, r)
		default:
			panic("unknown request method: " + method)
		}
	default:
		panic("unknown request: " + cmd)
	}
}

// ServeHTTP is the main entry point in the HTTP request processing.
func (n *Nova) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get(authToken) != token {
		unauthorizedResponse.send(w, r)
		return
	}
	path := strings.TrimLeft(r.URL.Path, "/")
	if path == "" {
		noVersionResponse.send(w, r)
		return
	}
	urlparts := strings.Split(path, "/")
	if len(urlparts) > 0 && urlparts[0] != versionPrefix {
		multipleChoicesResponse.send(w, r)
		return
	}
	if len(urlparts) >= 2 && urlparts[1] != tenantId {
		badRequestResponse.send(w, r)
		return
	}
	if strings.HasSuffix(path, "/") {
		notFoundResponse.send(w, r)
		return
	}
	cmd := strings.Join(urlparts[2:], "/")
	switch op := urlparts[2]; op {
	case "flavors":
		n.handleFlavors(r.Method, cmd, w, r)
	default:
		notFoundResponse.send(w, r)
	}
}
