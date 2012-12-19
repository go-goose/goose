// Nova double testing service - HTTP API implementation

package novaservice

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"launchpad.net/goose/nova"
	"net/http"
	"path"
	"strconv"
	"strings"
)

const authToken = "X-Auth-Token"

// errorResponse defines a single HTTP error response.
type errorResponse struct {
	code        int
	body        string
	contentType string
	errorText   string
}

// verbatim real Nova responses (as errors).
var (
	errUnauthorized = &errorResponse{
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
	errForbidden = &errorResponse{
		http.StatusForbidden,
		`{"forbidden": {"message": "Policy doesn't allow compute_extension:` +
			`flavormanage to be performed.", "code": 403}}`,
		"application/json; charset=UTF-8",
		"",
	}
	errBadRequest = &errorResponse{
		http.StatusBadRequest,
		`{"badRequest": {"message": "Malformed request url", "code": 400}}`,
		"application/json; charset=UTF-8",
		"",
	}
	errBadRequest2 = &errorResponse{
		http.StatusBadRequest,
		`{"badRequest": {"message": "The server could not comply with the ` +
			`request since it is either malformed or otherwise incorrect.", "code": 400}}`,
		"application/json; charset=UTF-8",
		"",
	}
	errBadRequestSG = &errorResponse{
		http.StatusBadRequest,
		`{"badRequest": {"message": "Security group id should be integer", "code": 400}}`,
		"application/json; charset=UTF-8",
		"",
	}
	errNotFound = &errorResponse{
		http.StatusNotFound,
		`404 Not Found

The resource could not be found.


`,
		"text/plain; charset=UTF-8",
		"",
	}
	errNotFoundJSON = &errorResponse{
		http.StatusNotFound,
		`{"itemNotFound": {"message": "The resource could not be found.", "code": 404}}`,
		"application/json; charset=UTF-8",
		"",
	}
	errNotFoundJSONSG = &errorResponse{
		http.StatusNotFound,
		`{"itemNotFound": {"message": "Security group $ID$ not found.", "code": 404}}`,
		"application/json; charset=UTF-8",
		"",
	}
	errNotFoundJSONSGR = &errorResponse{
		http.StatusNotFound,
		`{"itemNotFound": {"message": "Rule ($ID$) not found.", "code": 404}}`,
		"application/json; charset=UTF-8",
		"",
	}
	errMultipleChoices = &errorResponse{
		http.StatusMultipleChoices,
		`{"choices": [{"status": "CURRENT", "media-types": [{"base": ` +
			`"application/xml", "type": "application/vnd.openstack.compute+` +
			`xml;version=2"}, {"base": "application/json", "type": "application/` +
			`vnd.openstack.compute+json;version=2"}], "id": "v2.0", "links": ` +
			`[{"href": "$ENDPOINT$$URL$", "rel": "self"}]}]}`,
		"application/json",
		"",
	}
	errNoVersion = &errorResponse{
		http.StatusOK,
		`{"versions": [{"status": "CURRENT", "updated": "2011-01-21` +
			`T11:33:21Z", "id": "v2.0", "links": [{"href": "$ENDPOINT$", "rel": "self"}]}]}`,
		"application/json",
		"",
	}
	errVersionsLinks = &errorResponse{
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
	errNoContent = &errorResponse{
		http.StatusNoContent,
		"",
		"text/plain; charset=UTF-8",
		"",
	}
	errAccepted = &errorResponse{
		http.StatusAccepted,
		"",
		"text/plain; charset=UTF-8",
		"",
	}
	errNotImplemented = &errorResponse{
		http.StatusNotImplemented,
		"501 Not Implemented",
		"text/plain; charset=UTF-8",
		"",
	}
	errInternal = &errorResponse{
		http.StatusInternalServerError,
		`{"internalServerError":{"message":"$ERROR$",code:500}}`,
		"application/json",
		"",
	}
)

func (e *errorResponse) Error() string {
	return e.errorText
}

// endpoint returns the current testing server's endpoint URL.
func endpoint() string {
	return hostname + versionPath + "/"
}

// replaceVars replaces $ENDPOINT$, $URL$, $ID$, and $ERROR$ in the
// error response body with their values, taking the original request
// into account, and returns the result as a []byte.
func (e *errorResponse) replaceVars(r *http.Request) []byte {
	url := strings.TrimLeft(r.URL.Path, "/")
	body := e.body
	body = strings.Replace(body, "$ENDPOINT$", endpoint(), -1)
	body = strings.Replace(body, "$URL$", url, -1)
	if e.Error() != "" {
		body = strings.Replace(body, "$ERROR$", e.Error(), -1)
	}
	if slash := strings.LastIndex(url, "/"); slash != -1 {
		body = strings.Replace(body, "$ID$", url[slash+1:], -1)
	}
	return []byte(body)
}

func (e *errorResponse) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if e.contentType != "" {
		w.Header().Set("Content-Type", e.contentType)
	}
	var body []byte
	if e.body != "" {
		body = e.replaceVars(r)
	}
	// workaround for https://code.google.com/p/go/issues/detail?id=4454
	w.Header().Set("Content-Length", strconv.Itoa(len(body)))
	if e.code != 0 {
		w.WriteHeader(e.code)
	}
	if len(body) > 0 {
		w.Write(body)
	}
}

type novaHandler struct {
	n      *Nova
	method func(n *Nova, w http.ResponseWriter, r *http.Request) error
}

func (h *novaHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	// handle invalid X-Auth-Token header
	if r.Header.Get(authToken) != h.n.token {
		errUnauthorized.ServeHTTP(w, r)
		return
	}
	// handle trailing slash in the path
	if strings.HasSuffix(path, "/") && path != "/" {
		errNotFound.ServeHTTP(w, r)
		return
	}
	err := h.method(h.n, w, r)
	if err == nil {
		return
	}
	resp, _ := err.(http.Handler)
	if resp == nil {
		resp := errInternal
		resp.errorText = err.Error()
	}
	resp.ServeHTTP(w, r)
}

// sendError converts the given error and sends it as errInternal
// error response, returning the error (as a shortcut for handlers).
func sendError(err error, w http.ResponseWriter, r *http.Request) error {
	resp := errInternal
	resp.errorText = err.Error()
	resp.ServeHTTP(w, r)
	return err
}

// sendJSON sends the specified response serialized as JSON, returning
// nil (as a shortcut for handlers) or an error (when marshaling fails).
func sendJSON(code int, resp interface{}, w http.ResponseWriter, r *http.Request) error {
	var data []byte
	if resp != nil {
		var err error
		data, err = json.Marshal(resp)
		if err != nil {
			return sendError(err, w, r)
		}
	}
	// workaround for https://code.google.com/p/go/issues/detail?id=4454
	w.Header().Set("Content-Length", strconv.Itoa(len(data)))
	w.WriteHeader(code)
	w.Write(data)
	return nil
}

func (n *Nova) handler(method func(n *Nova, w http.ResponseWriter, r *http.Request) error) http.Handler {
	return &novaHandler{n, method}
}

func (n *Nova) handleRoot(w http.ResponseWriter, r *http.Request) error {
	if r.URL.Path == "/" {
		return errNoVersion
	}
	return errMultipleChoices
}

// handleFlavors handles the flavors HTTP API.
func (n *Nova) handleFlavors(w http.ResponseWriter, r *http.Request) error {
	switch r.Method {
	case "GET":
		if flavorId := path.Base(r.URL.Path); flavorId != "flavors" {
			flavor, err := n.flavor(flavorId)
			if err != nil {
				return errNotFound
			}
			var resp struct {
				Flavor nova.FlavorDetail `json:"flavor"`
			}
			resp.Flavor = *flavor
			return sendJSON(http.StatusOK, resp, w, r)
		}
		entities := n.allFlavorsAsEntities()
		var resp struct {
			Flavors []nova.Entity `json:"flavors"`
		}
		resp.Flavors = entities
		if len(entities) == 0 {
			resp.Flavors = []nova.Entity{}
		}
		return sendJSON(http.StatusOK, resp, w, r)
	case "POST":
		if flavorId := path.Base(r.URL.Path); flavorId != "flavors" {
			return errNotFound
		}
		body, err := ioutil.ReadAll(r.Body)
		defer r.Body.Close()
		if err != nil {
			return sendError(err, w, r)
		}
		if len(body) == 0 {
			return errBadRequest2
		}
		return errNotImplemented
	case "PUT":
		if flavorId := path.Base(r.URL.Path); flavorId != "flavors" {
			return errNotFoundJSON
		}
		return errNotFound
	case "DELETE":
		if flavorId := path.Base(r.URL.Path); flavorId != "flavors" {
			return errForbidden
		}
		return errNotFound
	default:
		panic("unknown request method: " + r.Method)
	}
	return nil
}

// handleFlavorsDetail handles the flavors/detail HTTP API.
func (n *Nova) handleFlavorsDetail(w http.ResponseWriter, r *http.Request) error {
	switch r.Method {
	case "GET":
		if flavorId := path.Base(r.URL.Path); flavorId != "detail" {
			return errNotFound
		}
		flavors := n.allFlavors()
		var resp struct {
			Flavors []nova.FlavorDetail `json:"flavors"`
		}
		resp.Flavors = flavors
		if len(flavors) == 0 {
			resp.Flavors = []nova.FlavorDetail{}
		}
		return sendJSON(http.StatusOK, resp, w, r)
	case "POST":
		return errNotFound
	case "PUT":
		if flavorId := path.Base(r.URL.Path); flavorId != "detail" {
			return errNotFound
		}
		return errNotFoundJSON
	case "DELETE":
		if flavorId := path.Base(r.URL.Path); flavorId != "detail" {
			return errNotFound
		}
		return errForbidden
	default:
		panic("unknown request method: " + r.Method)
	}
	return nil
}

// handleServerActions handles the servers/<id>/action HTTP API.
func (n *Nova) handleServerActions(server *nova.ServerDetail, w http.ResponseWriter, r *http.Request) error {
	if server == nil {
		return errNotFound
	}
	body, err := ioutil.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil || len(body) == 0 {
		return errNotFound
	}
	var action struct {
		AddSecurityGroup *struct {
			Name string
		}
		RemoveSecurityGroup *struct {
			Name string
		}
		AddFloatingIP *struct {
			Address string
		}
		RemoveFloatingIP *struct {
			Address string
		}
	}
	if err := json.Unmarshal(body, &action); err != nil {
		return sendError(err, w, r)
	}
	switch {
	case action.AddSecurityGroup != nil:
		name := action.AddSecurityGroup.Name
		group, err := n.securityGroupByName(name)
		if err != nil || n.hasServerSecurityGroup(server.Id, group.Id) {
			return errNotFound
		}
		if err = n.addServerSecurityGroup(server.Id, group.Id); err != nil {
			return sendError(err, w, r)
		}
		return errNoContent
	case action.RemoveSecurityGroup != nil:
		name := action.RemoveSecurityGroup.Name
		group, err := n.securityGroupByName(name)
		if err != nil || !n.hasServerSecurityGroup(server.Id, group.Id) {
			return errNotFound
		}
		if err = n.removeServerSecurityGroup(server.Id, group.Id); err != nil {
			return sendError(err, w, r)
		}
		return errNoContent
	case action.AddFloatingIP != nil:
		addr := action.AddFloatingIP.Address
		if n.hasServerFloatingIP(server.Id, addr) {
			return errNotFound
		}
		fip, err := n.floatingIPByAddr(addr)
		if err != nil {
			return errNotFound
		}
		if err = n.addServerFloatingIP(server.Id, fip.Id); err != nil {
			return sendError(err, w, r)
		}
		return errNoContent
	case action.RemoveFloatingIP != nil:
		addr := action.RemoveFloatingIP.Address
		if !n.hasServerFloatingIP(server.Id, addr) {
			return errNotFound
		}
		fip, err := n.floatingIPByAddr(addr)
		if err != nil {
			return errNotFound
		}
		if err = n.removeServerFloatingIP(server.Id, fip.Id); err != nil {
			return sendError(err, w, r)
		}
		return errNoContent
	default:
		panic("unknown server action: " + string(body))
	}
	return nil
}

// handleServers handles the servers HTTP API.
func (n *Nova) handleServers(w http.ResponseWriter, r *http.Request) error {
	switch r.Method {
	case "GET":
		if suffix := path.Base(r.URL.Path); suffix != "servers" {
			groups := false
			serverId := ""
			if suffix == "os-security-groups" {
				// handle GET /servers/<id>/os-security-groups
				serverId = path.Base(strings.Replace(r.URL.Path, "/os-security-groups", "", 1))
				groups = true
			} else {
				serverId = suffix
			}
			server, err := n.server(serverId)
			if err != nil {
				return errNotFoundJSON
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
				return sendJSON(http.StatusOK, resp, w, r)
			}
			var resp struct {
				Server nova.ServerDetail `json:"server"`
			}
			resp.Server = *server
			return sendJSON(http.StatusOK, resp, w, r)
		}
		entities := n.allServersAsEntities()
		var resp struct {
			Servers []nova.Entity `json:"servers"`
		}
		resp.Servers = entities
		if len(entities) == 0 {
			resp.Servers = []nova.Entity{}
		}
		return sendJSON(http.StatusOK, resp, w, r)
	case "POST":
		if suffix := path.Base(r.URL.Path); suffix != "servers" {
			serverId := ""
			if suffix == "action" {
				// handle POST /servers/<id>/action
				serverId = path.Base(strings.Replace(r.URL.Path, "/action", "", 1))
				server, _ := n.server(serverId)
				return n.handleServerActions(server, w, r)
			} else {
				serverId = suffix
			}
			return errNotFound
		}
		body, err := ioutil.ReadAll(r.Body)
		defer r.Body.Close()
		if err != nil {
			return sendError(err, w, r)
		}
		if len(body) == 0 {
			return errBadRequest2
		}
		return errNotImplemented
	case "PUT":
		if serverId := path.Base(r.URL.Path); serverId != "servers" {
			return errBadRequest2
		}
		return errNotFound
	case "DELETE":
		if serverId := path.Base(r.URL.Path); serverId != "servers" {
			if _, err := n.server(serverId); err != nil {
				return errNotFoundJSON
			}
			if err := n.removeServer(serverId); err != nil {
				return sendError(err, w, r)
			}
			return errNoContent
		}
		return errNotFound
	default:
		panic("unknown request method: " + r.Method)
	}
	return nil
}

// handleServersDetail handles the servers/detail HTTP API.
func (n *Nova) handleServersDetail(w http.ResponseWriter, r *http.Request) error {
	switch r.Method {
	case "GET":
		if serverId := path.Base(r.URL.Path); serverId != "detail" {
			return errNotFound
		}
		servers := n.allServers()
		var resp struct {
			Servers []nova.ServerDetail `json:"servers"`
		}
		resp.Servers = servers
		if len(servers) == 0 {
			resp.Servers = []nova.ServerDetail{}
		}
		return sendJSON(http.StatusOK, resp, w, r)
	case "POST":
		return errNotFound
	case "PUT":
		if serverId := path.Base(r.URL.Path); serverId != "detail" {
			return errNotFound
		}
		return errBadRequest2
	case "DELETE":
		if serverId := path.Base(r.URL.Path); serverId != "detail" {
			return errNotFound
		}
		return errNotFoundJSON
	default:
		panic("unknown request method: " + r.Method)
	}
	return nil
}

// processGroupId extracts and validates group ID from the given
// request, returning the group (if valid); nil and no error (no group
// ID was present in the path); or nil and an error (the error
// response was sent in this case)
func (n *Nova) processGroupId(w http.ResponseWriter, r *http.Request) (*nova.SecurityGroup, error) {
	if groupId := path.Base(r.URL.Path); groupId != "os-security-groups" {
		id, err := strconv.Atoi(groupId)
		if err != nil {
			return nil, errBadRequestSG
		}
		group, err := n.securityGroup(id)
		if err != nil {
			return nil, errNotFoundJSONSG
		}
		return group, nil
	}
	return nil, nil
}

// handleSecurityGroups handles the os-security-groups HTTP API.
func (n *Nova) handleSecurityGroups(w http.ResponseWriter, r *http.Request) error {
	switch r.Method {
	case "GET":
		if group, err := n.processGroupId(w, r); group != nil {
			var resp struct {
				Group nova.SecurityGroup `json:"security_group"`
			}
			resp.Group = *group
			return sendJSON(http.StatusOK, resp, w, r)
		} else if err == nil {
			groups := n.allSecurityGroups()
			var resp struct {
				Groups []nova.SecurityGroup `json:"security_groups"`
			}
			resp.Groups = groups
			if len(groups) == 0 {
				resp.Groups = []nova.SecurityGroup{}
			}
			return sendJSON(http.StatusOK, resp, w, r)
		} else {
			return err
		}
	case "POST":
		if groupId := path.Base(r.URL.Path); groupId != "os-security-groups" {
			return errNotFound
		}
		body, err := ioutil.ReadAll(r.Body)
		defer r.Body.Close()
		if err != nil || len(body) == 0 {
			return errBadRequest2
		}
		var req struct {
			Group struct {
				Name        string
				Description string
			} `json:"security_group"`
		}
		if err = json.Unmarshal(body, &req); err != nil {
			return sendError(err, w, r)
		} else {
			n.nextGroupId++
			nextId := n.nextGroupId
			err = n.addSecurityGroup(nova.SecurityGroup{
				Id:          nextId,
				Name:        req.Group.Name,
				Description: req.Group.Description,
			})
			if err != nil {
				return sendError(err, w, r)
			}
			group, err := n.securityGroup(nextId)
			if err != nil {
				return sendError(err, w, r)
			}
			var resp struct {
				Group nova.SecurityGroup `json:"security_group"`
			}
			resp.Group = *group
			return sendJSON(http.StatusOK, resp, w, r)
		}
	case "PUT":
		if groupId := path.Base(r.URL.Path); groupId != "os-security-groups" {
			return errNotFoundJSON
		}
		return errNotFound
	case "DELETE":
		if group, err := n.processGroupId(w, r); group != nil {
			if err := n.removeSecurityGroup(group.Id); err != nil {
				return sendError(err, w, r)
			}
			if n.nextGroupId > 0 {
				n.nextGroupId--
			}
			return errNoContent
		} else if err == nil {
			return errNotFound
		} else {
			return err
		}
	default:
		panic("unknown request method: " + r.Method)
	}
	return nil
}

// handleSecurityGroupRules handles the os-security-group-rules HTTP API.
func (n *Nova) handleSecurityGroupRules(w http.ResponseWriter, r *http.Request) error {
	switch r.Method {
	case "GET":
		return errNotFoundJSON
	case "POST":
		if ruleId := path.Base(r.URL.Path); ruleId != "os-security-group-rules" {
			return errNotFound
		}
		body, err := ioutil.ReadAll(r.Body)
		defer r.Body.Close()
		if err != nil || len(body) == 0 {
			return errBadRequest2
		}
		var req struct {
			Rule nova.RuleInfo `json:"security_group_rule"`
		}
		if err = json.Unmarshal(body, &req); err != nil {
			return sendError(err, w, r)
		} else {
			n.nextRuleId++
			nextId := n.nextRuleId
			err = n.addSecurityGroupRule(nextId, req.Rule)
			if err != nil {
				return sendError(err, w, r)
			}
			rule, err := n.securityGroupRule(nextId)
			if err != nil {
				return sendError(err, w, r)
			}
			var resp struct {
				Rule nova.SecurityGroupRule `json:"security_group_rule"`
			}
			resp.Rule = *rule
			return sendJSON(http.StatusOK, resp, w, r)
		}
	case "PUT":
		if ruleId := path.Base(r.URL.Path); ruleId != "os-security-group-rules" {
			return errNotFoundJSON
		}
		return errNotFound
	case "DELETE":
		if ruleId := path.Base(r.URL.Path); ruleId != "os-security-group-rules" {
			id, err := strconv.Atoi(ruleId)
			if err != nil {
				// weird, but this is how nova responds
				return errBadRequestSG
			}
			if _, err = n.securityGroupRule(id); err != nil {
				return errNotFoundJSONSGR
			}
			if err = n.removeSecurityGroupRule(id); err != nil {
				return sendError(err, w, r)
			}
			if n.nextRuleId > 0 {
				n.nextRuleId--
			}
			return errNoContent
		}
		return errNotFound
	default:
		panic("unknown request method: " + r.Method)
	}
	return nil
}

// handleFloatingIPs handles the os-floating-ips HTTP API.
func (n *Nova) handleFloatingIPs(w http.ResponseWriter, r *http.Request) error {
	switch r.Method {
	case "GET":
		if ipId := path.Base(r.URL.Path); ipId != "os-floating-ips" {
			nId, err := strconv.Atoi(ipId)
			if err != nil {
				return errNotFoundJSON
			}
			fip, err := n.floatingIP(nId)
			if err != nil {
				return errNotFoundJSON
			}
			var resp struct {
				IP nova.FloatingIP `json:"floating_ip"`
			}
			resp.IP = *fip
			return sendJSON(http.StatusOK, resp, w, r)
		}
		fips := n.allFloatingIPs()
		var resp struct {
			IPs []nova.FloatingIP `json:"floating_ips"`
		}
		resp.IPs = fips
		if len(fips) == 0 {
			resp.IPs = []nova.FloatingIP{}
		}
		return sendJSON(http.StatusOK, resp, w, r)
	case "POST":
		if ipId := path.Base(r.URL.Path); ipId != "os-floating-ips" {
			return errNotFound
		}
		n.nextIPId++
		nextId := n.nextIPId
		addr := fmt.Sprintf("10.0.0.%d", nextId)
		fip := nova.FloatingIP{Id: nextId, IP: addr, Pool: "nova"}
		err := n.addFloatingIP(fip)
		if err != nil {
			return sendError(err, w, r)
		}
		var resp struct {
			IP nova.FloatingIP `json:"floating_ip"`
		}
		resp.IP = fip
		return sendJSON(http.StatusOK, resp, w, r)
	case "PUT":
		if ipId := path.Base(r.URL.Path); ipId != "os-floating-ips" {
			return errNotFoundJSON
		}
		return errNotFound
	case "DELETE":
		if ipId := path.Base(r.URL.Path); ipId != "os-floating-ips" {
			if nId, err := strconv.Atoi(ipId); err == nil {
				if err := n.removeFloatingIP(nId); err == nil {
					if n.nextIPId > 0 {
						n.nextIPId--
					}
					return errAccepted
				}
			}
			return errNotFoundJSON
		}
		return errNotFound
	default:
		panic("unknown request method: " + r.Method)
	}
	return nil
}

// setupHTTP attaches all the needed handlers to provide the HTTP API.
func (n *Nova) setupHTTP(mux *http.ServeMux) {
	handlers := map[string]http.Handler{
		"/":                              n.handler((*Nova).handleRoot),
		"/$v/":                           errBadRequest,
		"/$v/$t/":                        errNotFound,
		"/$v/$t/flavors":                 n.handler((*Nova).handleFlavors),
		"/$v/$t/flavors/detail":          n.handler((*Nova).handleFlavorsDetail),
		"/$v/$t/servers":                 n.handler((*Nova).handleServers),
		"/$v/$t/servers/detail":          n.handler((*Nova).handleServersDetail),
		"/$v/$t/os-security-groups":      n.handler((*Nova).handleSecurityGroups),
		"/$v/$t/os-security-group-rules": n.handler((*Nova).handleSecurityGroupRules),
		"/$v/$t/os-floating-ips":         n.handler((*Nova).handleFloatingIPs),
	}
	for path, h := range handlers {
		path = strings.Replace(path, "$v", n.versionPath, 1)
		path = strings.Replace(path, "$t", n.tenantId, 1)
		if !strings.HasSuffix(path, "/") {
			mux.Handle(path+"/", h)
		}
		mux.Handle(path, h)
	}
}
