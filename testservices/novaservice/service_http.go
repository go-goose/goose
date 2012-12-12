// Nova double testing service - HTTP API implementation

package novaservice

import (
	"encoding/json"
	"io/ioutil"
	"launchpad.net/goose/nova"
	"net/http"
	"strconv"
	"strings"
)

const authToken = "X-Auth-Token"

// response defines a single HTTP response.
type response struct {
	code        int
	body        string
	contentType string
	errorText   string
}

// verbatim real Nova responses
var (
	unauthorizedResponse = response{
		http.StatusUnauthorized,
		`401 Unauthorized

This server could not verify that you are authorized to access the ` +
			`document you requested. Either you supplied the wrong ` +
			`credentials (e.g., bad password), or your browser does ` +
			`not understand how to supply the credentials required.

 Authentication required
`,
		"text/plain; charset=UTF-8",
		"",
	}
	forbiddenResponse = response{
		http.StatusForbidden,
		`{"forbidden": {"message": "Policy doesn't allow compute_extension:` +
			`flavormanage to be performed.", "code": 403}}`,
		"application/json; charset=UTF-8",
		"",
	}
	badRequestResponse = response{
		http.StatusBadRequest,
		`{"badRequest": {"message": "Malformed request url", "code": 400}}`,
		"application/json; charset=UTF-8",
		"",
	}
	badRequest2Response = response{
		http.StatusBadRequest,
		`{"badRequest": {"message": "The server could not comply with the ` +
			`request since it is either malformed or otherwise incorrect.", "code": 400}}`,
		"application/json; charset=UTF-8",
		"",
	}
	notFoundResponse = response{
		http.StatusNotFound,
		`404 Not Found

The resource could not be found.


`,
		"text/plain; charset=UTF-8",
		"",
	}
	notFoundJSONResponse = response{
		http.StatusNotFound,
		`{"itemNotFound": {"message": "The resource could not be found.", "code": 404}}`,
		"application/json; charset=UTF-8",
		"",
	}
	multipleChoicesResponse = response{
		http.StatusMultipleChoices,
		`{"choices": [{"status": "CURRENT", "media-types": [{"base": ` +
			`"application/xml", "type": "application/vnd.openstack.compute+` +
			`xml;version=2"}, {"base": "application/json", "type": "application/` +
			`vnd.openstack.compute+json;version=2"}], "id": "v2.0", "links": ` +
			`[{"href": "$ENDPOINT$$URL$", "rel": "self"}]}]}`,
		"application/json",
		"",
	}
	noVersionResponse = response{
		http.StatusOK,
		`{"versions": [{"status": "CURRENT", "updated": "2011-01-21` +
			`T11:33:21Z", "id": "v2.0", "links": [{"href": "$ENDPOINT$", "rel": "self"}]}]}`,
		"application/json",
		"",
	}
	versionsLinksResponse = response{
		http.StatusOK,
		`{"version": {"status": "CURRENT", "updated": "2011-01-21T11` +
			`:33:21Z", "media-types": [{"base": "application/xml", "type": ` +
			`"application/vnd.openstack.compute+xml;version=2"}, {"base": ` +
			`"application/json", "type": "application/vnd.openstack.compute` +
			`+json;version=2"}], "id": "v2.0", "links": [{"href": "$ENDPOINT$"` +
			`, "rel": "self"}, {"href": "http://docs.openstack.org/api/openstack` +
			`-compute/1.1/os-compute-devguide-1.1.pdf", "type": "application/pdf` +
			`", "rel": "describedby"}, {"href": "http://docs.openstack.org/api/` +
			`openstack-compute/1.1/wadl/os-compute-1.1.wadl", "type": ` +
			`"application/vnd.sun.wadl+xml", "rel": "describedby"}]}}`,
		"application/json",
		"",
	}
	createdResponse = response{
		http.StatusCreated,
		"201 Created",
		"text/plain; charset=UTF-8",
		"",
	}
	noContentResponse = response{
		http.StatusNoContent,
		"",
		"text/plain; charset=UTF-8",
		"",
	}
	errorResponse = response{
		http.StatusInternalServerError,
		`{"internalServerError":{"message":"$ERROR$",code:500}}`,
		"application/json",
		"", // set by sendError()
	}
)

// endpoint returns the current testing server's endpoint URL.
func endpoint() string {
	return hostname + versionPath + "/"
}

// replaceVars replaces $ENDPOINT$, $URL$, and $ERROR$ in the response body
// with their values, taking the original requset into account, and
// returns the result as a []byte.
func (resp response) replaceVars(r *http.Request) []byte {
	url := strings.TrimLeft(r.URL.Path, "/")
	body := resp.body
	body = strings.Replace(body, "$ENDPOINT$", endpoint(), -1)
	body = strings.Replace(body, "$URL$", url, -1)
	if resp.errorText != "" {
		body = strings.Replace(body, "$ERROR$", resp.errorText, -1)
	}
	return []byte(body)
}

// send serializes the response as needed and sends it.
func (resp response) send(w http.ResponseWriter, r *http.Request) {
	if resp.contentType != "" {
		w.Header().Set("Content-Type", resp.contentType)
	}
	var body []byte
	if resp.body != "" {
		body = resp.replaceVars(r)
	}
	// workaround for https://code.google.com/p/go/issues/detail?id=4454
	w.Header().Set("Content-Length", strconv.Itoa(len(body)))
	if resp.code != 0 {
		w.WriteHeader(resp.code)
	}
	if len(body) > 0 {
		w.Write(body)
	}
}

// sendError responds with the given error to the given http request.
func sendError(err error, w http.ResponseWriter, r *http.Request) {
	eresp := errorResponse
	eresp.errorText = err.Error()
	eresp.send(w, r)
}

// sendJSON sends the specified response serialized as JSON.
func sendJSON(code int, resp interface{}, w http.ResponseWriter, r *http.Request) {
	var data []byte
	if resp != nil {
		var err error
		data, err = json.Marshal(resp)
		if err != nil {
			sendError(err, w, r)
			return
		}
	}
	// workaround for https://code.google.com/p/go/issues/detail?id=4454
	w.Header().Set("Content-Length", strconv.Itoa(len(data)))
	w.WriteHeader(code)
	w.Write(data)
}

// handleUnauthorizedNotFound is called for each request to check for
// common errors (X-Auth-Token and trailing slash in URL). Returns
// true if it's OK, false if a response was sent.
func (n *Nova) handleUnauthorizedNotFound(w http.ResponseWriter, r *http.Request) bool {
	path := r.URL.Path
	if r.Header.Get(authToken) != n.token {
		unauthorizedResponse.send(w, r)
		return false
	}
	if strings.HasSuffix(path, "/") && path != "/" {
		notFoundResponse.send(w, r)
		return false
	}
	return true
}

// handle registers the given Nova handler method for the URL prefix.
func (n *Nova) handle(prefix string, handler func(*Nova, http.ResponseWriter, *http.Request)) http.Handler {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if n.handleUnauthorizedNotFound(w, r) {
			handler(n, w, r)
		}
	})
	return http.StripPrefix(prefix, h)
}

// respond returns an http Handler sending the given response.
func (n *Nova) respond(resp response) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if n.handleUnauthorizedNotFound(w, r) {
			resp.send(w, r)
		}
	})
}

// handleFlavors handles the flavors HTTP API.
func (n *Nova) handleFlavors(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		entities := n.allFlavorsAsEntities()
		if len(entities) == 0 {
			sendJSON(http.StatusNoContent, nil, w, r)
			return
		}
		var resp struct {
			Flavors []nova.Entity `json:"flavors"`
		}
		resp.Flavors = entities
		sendJSON(http.StatusOK, resp, w, r)
	case "POST":
		body, err := ioutil.ReadAll(r.Body)
		r.Body.Close()
		if err != nil {
			sendError(err, w, r)
			return
		}
		if len(body) == 0 {
			badRequest2Response.send(w, r)
			return
		}
		var flavor struct {
			Flavor nova.FlavorDetail
		}
		err = json.Unmarshal(body, &flavor)
		if err != nil {
			sendError(err, w, r)
			return
		}
		n.buildFlavorLinks(&flavor.Flavor)
		err = n.addFlavor(flavor.Flavor)
		if err != nil {
			sendError(err, w, r)
			return
		}
		createdResponse.send(w, r)
	case "PUT", "DELETE":
		notFoundResponse.send(w, r)
	default:
		panic("unknown request method: " + r.Method)
	}
}

// handleFlavorsDetail handles the flavors/detail HTTP API.
func (n *Nova) handleFlavorsDetail(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		flavors := n.allFlavors()
		if len(flavors) == 0 {
			sendJSON(http.StatusNoContent, nil, w, r)
			return
		}
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
		panic("unknown request method: " + r.Method)
	}
}

// setupHTTP attaches all the needed handlers to provide the HTTP API.
func (n *Nova) setupHTTP(mux *http.ServeMux) {
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if !n.handleUnauthorizedNotFound(w, r) {
			return
		}
		if r.URL.Path == "/" {
			noVersionResponse.send(w, r)
		} else {
			multipleChoicesResponse.send(w, r)
		}
	})
	urlVersion := "/" + n.versionPath + "/"
	urlTenant := urlVersion + n.tenantId + "/"
	mux.Handle(urlVersion, n.respond(badRequestResponse))
	mux.HandleFunc(urlTenant, func(w http.ResponseWriter, r *http.Request) {
		if !n.handleUnauthorizedNotFound(w, r) {
			return
		}
		// any unknown path
		notFoundResponse.send(w, r)
	})
	mux.Handle(urlTenant+"flavors", n.handle(urlTenant, (*Nova).handleFlavors))
	mux.Handle(urlTenant+"flavors/detail", n.handle(urlTenant, (*Nova).handleFlavorsDetail))
}
