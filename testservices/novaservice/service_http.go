// Nova double testing service - HTTP API implementation

package novaservice

import (
	"encoding/json"
	"fmt"
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
	badRequestSGResponse = response{
		http.StatusBadRequest,
		`{"badRequest": {"message": "Security group id should be integer", "code": 400}}`,
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
	notFoundJSONSGResponse = response{
		http.StatusNotFound,
		`{"itemNotFound": {"message": "Security group $ID$ not found.", "code": 404}}`,
		"application/json; charset=UTF-8",
		"",
	}
	notFoundJSONSGRResponse = response{
		http.StatusNotFound,
		`{"itemNotFound": {"message": "Rule ($ID$) not found.", "code": 404}}`,
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
	noContentResponse = response{
		http.StatusNoContent,
		"",
		"text/plain; charset=UTF-8",
		"",
	}
	acceptedResponse = response{
		http.StatusAccepted,
		"",
		"text/plain; charset=UTF-8",
		"",
	}
	notImplementedResponse = response{
		http.StatusNotImplemented,
		"501 Not Implemented",
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

// replaceVars replaces $ENDPOINT$, $URL$, $ID, and $ERROR$ in the
// response body with their values, taking the original requset into
// account, and returns the result as a []byte.
func (resp response) replaceVars(r *http.Request) []byte {
	url := strings.TrimLeft(r.URL.Path, "/")
	body := resp.body
	body = strings.Replace(body, "$ENDPOINT$", endpoint(), -1)
	body = strings.Replace(body, "$URL$", url, -1)
	if resp.errorText != "" {
		body = strings.Replace(body, "$ERROR$", resp.errorText, -1)
	}
	if slash := strings.LastIndex(url, "/"); slash != -1 {
		body = strings.Replace(body, "$ID$", url[slash+1:], -1)
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

// getId extracts and returns the last part of r.URL.Path, after the
// last slash (if any), which is used as ID of an API object,
// stripping first the given prefix from the path.
func getId(prefix string, r *http.Request) string {
	path := strings.Replace(r.URL.Path, prefix, "", 1)
	if slash := strings.LastIndex(path, "/"); slash != -1 {
		return path[slash+1:]
	}
	return ""
}

// handleFlavors handles the flavors HTTP API.
func (n *Nova) handleFlavors(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		if flavorId := getId("flavors", r); flavorId != "" {
			flavor, err := n.flavor(flavorId)
			if err != nil {
				notFoundResponse.send(w, r)
				return
			}
			var resp struct {
				Flavor nova.FlavorDetail `json:"flavor"`
			}
			resp.Flavor = *flavor
			sendJSON(http.StatusOK, resp, w, r)
			return
		}
		entities := n.allFlavorsAsEntities()
		var resp struct {
			Flavors []nova.Entity `json:"flavors"`
		}
		resp.Flavors = entities
		if len(entities) == 0 {
			resp.Flavors = []nova.Entity{}
		}
		sendJSON(http.StatusOK, resp, w, r)
	case "POST":
		if flavorId := getId("flavors", r); flavorId != "" {
			notFoundResponse.send(w, r)
			return
		}
		body, err := ioutil.ReadAll(r.Body)
		defer r.Body.Close()
		if err != nil {
			sendError(err, w, r)
			return
		}
		if len(body) == 0 {
			badRequest2Response.send(w, r)
			return
		}
		notImplementedResponse.send(w, r)
	case "PUT":
		if flavorId := getId("flavors", r); flavorId != "" {
			notFoundJSONResponse.send(w, r)
			return
		}
		notFoundResponse.send(w, r)
	case "DELETE":
		if flavorId := getId("flavors", r); flavorId != "" {
			forbiddenResponse.send(w, r)
			return
		}
		notFoundResponse.send(w, r)
	default:
		panic("unknown request method: " + r.Method)
	}
}

// handleFlavorsDetail handles the flavors/detail HTTP API.
func (n *Nova) handleFlavorsDetail(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		if flavorId := getId("flavors/detail", r); flavorId != "" {
			notFoundResponse.send(w, r)
			return
		}
		flavors := n.allFlavors()
		var resp struct {
			Flavors []nova.FlavorDetail `json:"flavors"`
		}
		resp.Flavors = flavors
		if len(flavors) == 0 {
			resp.Flavors = []nova.FlavorDetail{}
		}
		sendJSON(http.StatusOK, resp, w, r)
	case "POST":
		notFoundResponse.send(w, r)
	case "PUT":
		if flavorId := getId("flavors/detail", r); flavorId != "" {
			notFoundResponse.send(w, r)
			return
		}
		notFoundJSONResponse.send(w, r)
	case "DELETE":
		if flavorId := getId("flavors/detail", r); flavorId != "" {
			notFoundResponse.send(w, r)
			return
		}
		forbiddenResponse.send(w, r)
	default:
		panic("unknown request method: " + r.Method)
	}
}

// handleServerActions handles the servers/<id/action HTTP API.
func (n *Nova) handleServerActions(server nova.ServerDetail, w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil || len(body) == 0 {
		notFoundResponse.send(w, r)
		return
	}
	var action map[string]interface{}
	err = json.Unmarshal(body, &action)
	if err != nil {
		sendError(err, w, r)
		return
	}
	if ac := action["addSecurityGroup"]; ac != nil {
		name := ac.(map[string]interface{})["name"].(string)
		group, err := n.securityGroupByName(name)
		if err != nil || n.hasServerSecurityGroup(server.Id, group.Id) {
			notFoundResponse.send(w, r)
			return
		}
		err = n.addServerSecurityGroup(server.Id, group.Id)
		if err != nil {
			sendError(err, w, r)
			return
		}
		noContentResponse.send(w, r)
		return
	}
	if ac := action["removeSecurityGroup"]; ac != nil {
		name := ac.(map[string]interface{})["name"].(string)
		group, err := n.securityGroupByName(name)
		if err != nil || !n.hasServerSecurityGroup(server.Id, group.Id) {
			notFoundResponse.send(w, r)
			return
		}
		err = n.removeServerSecurityGroup(server.Id, group.Id)
		if err != nil {
			sendError(err, w, r)
			return
		}
		noContentResponse.send(w, r)
		return
	}
	if ac := action["addFloatingIp"]; ac != nil {
		addr := ac.(map[string]interface{})["address"].(string)
		if n.hasServerFloatingIP(server.Id, addr) {
			notFoundResponse.send(w, r)
			return
		}
		fip, err := n.floatingIPByAddr(addr)
		if err != nil {
			notFoundResponse.send(w, r)
			return
		}
		if err = n.addServerFloatingIP(server.Id, fip.Id); err != nil {
			sendError(err, w, r)
			return
		}
		noContentResponse.send(w, r)
		return
	}
	if ac := action["removeFloatingIp"]; ac != nil {
		addr := ac.(map[string]interface{})["address"].(string)
		if !n.hasServerFloatingIP(server.Id, addr) {
			notFoundResponse.send(w, r)
			return
		}
		fip, err := n.floatingIPByAddr(addr)
		if err != nil {
			notFoundResponse.send(w, r)
			return
		}
		if err = n.removeServerFloatingIP(server.Id, fip.Id); err != nil {
			sendError(err, w, r)
			return
		}
		noContentResponse.send(w, r)
		return
	}
	panic("unknown server action: " + string(body))
}

// handleServers handles the servers HTTP API.
func (n *Nova) handleServers(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		if serverId := getId("servers", r); serverId != "" {
			groups := false
			if serverId == "os-security-groups" {
				// handle GET /servers/<id>/os-security-groups
				serverId = strings.Replace(r.URL.Path, "servers/", "", 1)
				serverId = serverId[:strings.Index(serverId, "/")]
				groups = true
			}
			server, err := n.server(serverId)
			if err != nil {
				notFoundJSONResponse.send(w, r)
				return
			}
			if groups {
				var resp struct {
					Groups []nova.SecurityGroup `json:"security_groups"`
				}
				srvGroups := n.allServerSecurityGroups(serverId)
				resp.Groups = srvGroups
				if len(srvGroups) == 0 {
					resp.Groups = []nova.SecurityGroup{}
				}
				sendJSON(http.StatusOK, resp, w, r)
			} else {
				var resp struct {
					Server nova.ServerDetail `json:"server"`
				}
				resp.Server = *server
				sendJSON(http.StatusOK, resp, w, r)
			}
			return
		}
		entities := n.allServersAsEntities()
		var resp struct {
			Servers []nova.Entity `json:"servers"`
		}
		resp.Servers = entities
		if len(entities) == 0 {
			resp.Servers = []nova.Entity{}
		}
		sendJSON(http.StatusOK, resp, w, r)
	case "POST":
		if serverId := getId("servers", r); serverId != "" {
			if serverId == "action" {
				// handle POST /servers/<id>/action
				serverId = strings.Replace(r.URL.Path, "servers/", "", 1)
				serverId = serverId[:strings.Index(serverId, "/")]
				if server, err := n.server(serverId); err == nil {
					n.handleServerActions(*server, w, r)
					return
				}
			}
			notFoundResponse.send(w, r)
			return
		}
		body, err := ioutil.ReadAll(r.Body)
		defer r.Body.Close()
		if err != nil {
			sendError(err, w, r)
			return
		}
		if len(body) == 0 {
			badRequest2Response.send(w, r)
			return
		}
		notImplementedResponse.send(w, r)
	case "PUT":
		if serverId := getId("servers", r); serverId != "" {
			badRequest2Response.send(w, r)
			return
		}
		notFoundResponse.send(w, r)
	case "DELETE":
		if serverId := getId("servers", r); serverId != "" {
			_, err := n.server(serverId)
			if err != nil {
				notFoundJSONResponse.send(w, r)
				return
			}
			err = n.removeServer(serverId)
			if err != nil {
				sendError(err, w, r)
				return
			}
			noContentResponse.send(w, r)
			return
		}
		notFoundResponse.send(w, r)
	default:
		panic("unknown request method: " + r.Method)
	}
}

// handleServersDetail handles the servers/detail HTTP API.
func (n *Nova) handleServersDetail(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		if serverId := getId("servers/detail", r); serverId != "" {
			notFoundResponse.send(w, r)
			return
		}
		servers := n.allServers()
		var resp struct {
			Servers []nova.ServerDetail `json:"servers"`
		}
		resp.Servers = servers
		if len(servers) == 0 {
			resp.Servers = []nova.ServerDetail{}
		}
		sendJSON(http.StatusOK, resp, w, r)
	case "POST":
		notFoundResponse.send(w, r)
	case "PUT":
		if serverId := getId("servers/detail", r); serverId != "" {
			notFoundResponse.send(w, r)
			return
		}
		badRequest2Response.send(w, r)
	case "DELETE":
		if serverId := getId("servers/detail", r); serverId != "" {
			notFoundResponse.send(w, r)
			return
		}
		notFoundJSONResponse.send(w, r)
	default:
		panic("unknown request method: " + r.Method)
	}
}

// processGroupId extracts and validates group ID from the given
// request, returning -1 when an error response was sent, 0 when no ID
// was present, and the parsed valid ID on success.
func (n *Nova) processGroupId(w http.ResponseWriter, r *http.Request) int {
	if groupId := getId("os-security-groups", r); groupId != "" {
		id, err := strconv.Atoi(groupId)
		if err != nil {
			badRequestSGResponse.send(w, r)
			return -1
		}
		if _, err = n.securityGroup(id); err != nil {
			notFoundJSONSGResponse.send(w, r)
			return -1
		}
		return id
	}
	return 0
}

// handleSecurityGroups handles the os-security-groups HTTP API.
func (n *Nova) handleSecurityGroups(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		if groupId := n.processGroupId(w, r); groupId > 0 {
			group, err := n.securityGroup(groupId)
			if err != nil {
				sendError(err, w, r)
				return
			}
			var resp struct {
				Group nova.SecurityGroup `json:"security_group"`
			}
			resp.Group = *group
			sendJSON(http.StatusOK, resp, w, r)
		} else if groupId == 0 {
			groups := n.allSecurityGroups()
			var resp struct {
				Groups []nova.SecurityGroup `json:"security_groups"`
			}
			resp.Groups = groups
			if len(groups) == 0 {
				resp.Groups = []nova.SecurityGroup{}
			}
			sendJSON(http.StatusOK, resp, w, r)
		}
	case "POST":
		if groupId := getId("os-security-groups", r); groupId != "" {
			notFoundResponse.send(w, r)
			return
		}
		body, err := ioutil.ReadAll(r.Body)
		defer r.Body.Close()
		if err != nil || len(body) == 0 {
			badRequest2Response.send(w, r)
			return
		}
		var req struct {
			Group struct {
				Name        string
				Description string
			} `json:"security_group"`
		}
		if err = json.Unmarshal(body, &req); err != nil {
			sendError(err, w, r)
		} else {
			nextId := len(n.allSecurityGroups()) + 1
			err = n.addSecurityGroup(nova.SecurityGroup{
				Id:          nextId,
				Name:        req.Group.Name,
				Description: req.Group.Description,
			})
			if err != nil {
				sendError(err, w, r)
				return
			}
			group, err := n.securityGroup(nextId)
			if err != nil {
				sendError(err, w, r)
				return
			}
			var resp struct {
				Group nova.SecurityGroup `json:"security_group"`
			}
			resp.Group = *group
			sendJSON(http.StatusOK, resp, w, r)
		}
	case "PUT":
		if groupId := getId("os-security-groups", r); groupId != "" {
			notFoundJSONResponse.send(w, r)
			return
		}
		notFoundResponse.send(w, r)
	case "DELETE":
		if groupId := n.processGroupId(w, r); groupId > 0 {
			err := n.removeSecurityGroup(groupId)
			if err != nil {
				sendError(err, w, r)
				return
			}
			noContentResponse.send(w, r)
		} else if groupId == 0 {
			notFoundResponse.send(w, r)
		}
	default:
		panic("unknown request method: " + r.Method)
	}
}

// handleSecurityGroupRules handles the os-security-group-rules HTTP API.
func (n *Nova) handleSecurityGroupRules(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		notFoundJSONResponse.send(w, r)
	case "POST":
		if ruleId := getId("os-security-group-rules", r); ruleId != "" {
			notFoundResponse.send(w, r)
			return
		}
		body, err := ioutil.ReadAll(r.Body)
		defer r.Body.Close()
		if err != nil || len(body) == 0 {
			badRequest2Response.send(w, r)
			return
		}
		var req struct {
			Rule nova.RuleInfo `json:"security_group_rule"`
		}
		if err = json.Unmarshal(body, &req); err != nil {
			sendError(err, w, r)
		} else {
			nextId := len(n.rules) + 1
			err = n.addSecurityGroupRule(nextId, req.Rule)
			if err != nil {
				sendError(err, w, r)
				return
			}
			rule, err := n.securityGroupRule(nextId)
			if err != nil {
				sendError(err, w, r)
				return
			}
			var resp struct {
				Rule nova.SecurityGroupRule `json:"security_group_rule"`
			}
			resp.Rule = *rule
			sendJSON(http.StatusOK, resp, w, r)
		}
	case "PUT":
		if ruleId := getId("os-security-group-rules", r); ruleId != "" {
			notFoundJSONResponse.send(w, r)
			return
		}
		notFoundResponse.send(w, r)
	case "DELETE":
		if ruleId := getId("os-security-group-rules", r); ruleId != "" {
			id, err := strconv.Atoi(ruleId)
			if err != nil {
				// weird, but this is how nova responds
				badRequestSGResponse.send(w, r)
				return
			}
			if _, err = n.securityGroupRule(id); err != nil {
				notFoundJSONSGRResponse.send(w, r)
				return
			}
			if err = n.removeSecurityGroupRule(id); err != nil {
				sendError(err, w, r)
				return
			}
			noContentResponse.send(w, r)
			return
		}
		notFoundResponse.send(w, r)
	default:
		panic("unknown request method: " + r.Method)
	}
}

// handleFloatingIPs handles the os-floating-ips HTTP API.
func (n *Nova) handleFloatingIPs(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		if ipId := getId("os-floating-ips", r); ipId != "" {
			nId, err := strconv.Atoi(ipId)
			if err != nil {
				notFoundJSONResponse.send(w, r)
				return
			}
			fip, err := n.floatingIP(nId)
			if err != nil {
				notFoundJSONResponse.send(w, r)
				return
			}
			var resp struct {
				IP nova.FloatingIP `json:"floating_ip"`
			}
			resp.IP = *fip
			sendJSON(http.StatusOK, resp, w, r)
			return
		}
		fips := n.allFloatingIPs()
		var resp struct {
			IPs []nova.FloatingIP `json:"floating_ips"`
		}
		resp.IPs = fips
		if len(fips) == 0 {
			resp.IPs = []nova.FloatingIP{}
		}
		sendJSON(http.StatusOK, resp, w, r)
	case "POST":
		if ipId := getId("os-floating-ips", r); ipId != "" {
			notFoundResponse.send(w, r)
			return
		}
		nextId := len(n.allFloatingIPs()) + 1
		addr := fmt.Sprintf("10.0.0.%d", nextId)
		fip := nova.FloatingIP{Id: nextId, IP: addr, Pool: "nova"}
		err := n.addFloatingIP(fip)
		if err != nil {
			sendError(err, w, r)
			return
		}
		var resp struct {
			IP nova.FloatingIP `json:"floating_ip"`
		}
		resp.IP = fip
		sendJSON(http.StatusOK, resp, w, r)
	case "PUT":
		if ipId := getId("os-floating-ips", r); ipId != "" {
			notFoundJSONResponse.send(w, r)
			return
		}
		notFoundResponse.send(w, r)
	case "DELETE":
		if ipId := getId("os-floating-ips", r); ipId != "" {
			// weird, but true - even on success 404 is returned
			nId, err := strconv.Atoi(ipId)
			if err == nil {
				if err := n.removeFloatingIP(nId); err == nil {
					acceptedResponse.send(w, r)
					return
				}
			}
			notFoundJSONResponse.send(w, r)
		}
		notFoundResponse.send(w, r)
	default:
		panic("unknown request method: " + r.Method)
	}
}

var handlersMap = map[string]func(*Nova, http.ResponseWriter, *http.Request){
	"flavors":                 (*Nova).handleFlavors,
	"flavors/detail":          (*Nova).handleFlavorsDetail,
	"servers":                 (*Nova).handleServers,
	"servers/detail":          (*Nova).handleServersDetail,
	"os-security-groups":      (*Nova).handleSecurityGroups,
	"os-security-group-rules": (*Nova).handleSecurityGroupRules,
	"os-floating-ips":         (*Nova).handleFloatingIPs,
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
		path := strings.Replace(r.URL.Path, urlTenant, "", 1)
		if !n.handleUnauthorizedNotFound(w, r) {
			return
		}
		// forward subpaths to registered handlers
		if strings.HasPrefix(path, "servers/") && strings.Index(path, "detail") == -1 {
			// forward /servers/<id>/os-security-groups (or ../action)
			n.handle(urlTenant, (*Nova).handleServers).ServeHTTP(w, r)
			return
		}
		// handle other, e.g. /.../flavors/xyz -> handleFlavors
		if slash := strings.LastIndex(path, "/"); slash != -1 {
			if handler, ok := handlersMap[path[:slash]]; ok {
				n.handle(urlTenant, handler).ServeHTTP(w, r)
				return
			}
		}
		// any unknown path
		fmt.Printf("unknown path: %q\n", path)
		notFoundResponse.send(w, r)
	})
	for pathSuffix, handler := range handlersMap {
		mux.Handle(urlTenant+pathSuffix, n.handle(urlTenant, handler))
	}
}
