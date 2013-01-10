// Nova double testing service - HTTP API implementation

package novaservice

import (
	"crypto/rand"
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
	nova        *Nova
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
		"unauthorized request",
		nil,
	}
	errForbidden = &errorResponse{
		http.StatusForbidden,
		`{"forbidden": {"message": "Policy doesn't allow compute_extension:` +
			`flavormanage to be performed.", "code": 403}}`,
		"application/json; charset=UTF-8",
		"forbidden flavors request",
		nil,
	}
	errBadRequest = &errorResponse{
		http.StatusBadRequest,
		`{"badRequest": {"message": "Malformed request url", "code": 400}}`,
		"application/json; charset=UTF-8",
		"bad request base path or URL",
		nil,
	}
	errBadRequest2 = &errorResponse{
		http.StatusBadRequest,
		`{"badRequest": {"message": "The server could not comply with the ` +
			`request since it is either malformed or otherwise incorrect.", "code": 400}}`,
		"application/json; charset=UTF-8",
		"bad request URL",
		nil,
	}
	errBadRequest3 = &errorResponse{
		http.StatusBadRequest,
		`{"badRequest": {"message": "Malformed request body", "code": 400}}`,
		"application/json; charset=UTF-8",
		"bad request body",
		nil,
	}
	errBadRequestSrvName = &errorResponse{
		http.StatusBadRequest,
		`{"badRequest": {"message": "Server name is not defined", "code": 400}}`,
		"application/json; charset=UTF-8",
		"bad request - missing server name",
		nil,
	}
	errBadRequestSrvFlavor = &errorResponse{
		http.StatusBadRequest,
		`{"badRequest": {"message": "Missing imageRef attribute", "code": 400}}`,
		"application/json; charset=UTF-8",
		"bad request - missing flavorRef",
		nil,
	}
	errBadRequestSrvImage = &errorResponse{
		http.StatusBadRequest,
		`{"badRequest": {"message": "Missing flavorRef attribute", "code": 400}}`,
		"application/json; charset=UTF-8",
		"bad request - missing imageRef",
		nil,
	}
	errBadRequestSG = &errorResponse{
		http.StatusBadRequest,
		`{"badRequest": {"message": "Security group id should be integer", "code": 400}}`,
		"application/json; charset=UTF-8",
		"bad security group id type",
		nil,
	}
	errNotFound = &errorResponse{
		http.StatusNotFound,
		`404 Not Found

The resource could not be found.


`,
		"text/plain; charset=UTF-8",
		"resource not found",
		nil,
	}
	errNotFoundJSON = &errorResponse{
		http.StatusNotFound,
		`{"itemNotFound": {"message": "The resource could not be found.", "code": 404}}`,
		"application/json; charset=UTF-8",
		"resource not found",
		nil,
	}
	errNotFoundJSONSG = &errorResponse{
		http.StatusNotFound,
		`{"itemNotFound": {"message": "Security group $ID$ not found.", "code": 404}}`,
		"application/json; charset=UTF-8",
		"",
		nil,
	}
	errNotFoundJSONSGR = &errorResponse{
		http.StatusNotFound,
		`{"itemNotFound": {"message": "Rule ($ID$) not found.", "code": 404}}`,
		"application/json; charset=UTF-8",
		"security rule not found",
		nil,
	}
	errMultipleChoices = &errorResponse{
		http.StatusMultipleChoices,
		`{"choices": [{"status": "CURRENT", "media-types": [{"base": ` +
			`"application/xml", "type": "application/vnd.openstack.compute+` +
			`xml;version=2"}, {"base": "application/json", "type": "application/` +
			`vnd.openstack.compute+json;version=2"}], "id": "v2.0", "links": ` +
			`[{"href": "$ENDPOINT$$URL$", "rel": "self"}]}]}`,
		"application/json",
		"multiple URL redirection choices",
		nil,
	}
	errNoVersion = &errorResponse{
		http.StatusOK,
		`{"versions": [{"status": "CURRENT", "updated": "2011-01-21` +
			`T11:33:21Z", "id": "v2.0", "links": [{"href": "$ENDPOINT$", "rel": "self"}]}]}`,
		"application/json",
		"no version specified in URL",
		nil,
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
		"version missing from URL",
		nil,
	}
	errNotImplemented = &errorResponse{
		http.StatusNotImplemented,
		"501 Not Implemented",
		"text/plain; charset=UTF-8",
		"not implemented",
		nil,
	}
	errNoGroupId = &errorResponse{
		errorText: "no security group id given",
	}
)

func (e *errorResponse) Error() string {
	return e.errorText
}

// requestBody returns the body for the error response, replacing
// $ENDPOINT$, $URL$, $ID$, and $ERROR$ in e.body with the values from
// the request.
func (e *errorResponse) requestBody(r *http.Request) []byte {
	url := strings.TrimLeft(r.URL.Path, "/")
	body := e.body
	if body != "" {
		if e.nova != nil {
			body = strings.Replace(body, "$ENDPOINT$", e.nova.endpoint(true, "/"), -1)
		}
		body = strings.Replace(body, "$URL$", url, -1)
		body = strings.Replace(body, "$ERROR$", e.Error(), -1)
		if slash := strings.LastIndex(url, "/"); slash != -1 {
			body = strings.Replace(body, "$ID$", url[slash+1:], -1)
		}
	}
	return []byte(body)
}

func (e *errorResponse) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if e.contentType != "" {
		w.Header().Set("Content-Type", e.contentType)
	}
	body := e.requestBody(r)
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
		resp = &errorResponse{
			http.StatusInternalServerError,
			`{"internalServerError":{"message":"$ERROR$",code:500}}`,
			"application/json",
			err.Error(),
			h.n,
		}
	}
	resp.ServeHTTP(w, r)
}

func writeResponse(w http.ResponseWriter, code int, body []byte) {
	// workaround for https://code.google.com/p/go/issues/detail?id=4454
	w.Header().Set("Content-Length", strconv.Itoa(len(body)))
	w.WriteHeader(code)
	w.Write(body)
}

// sendJSON sends the specified response serialized as JSON.
func sendJSON(code int, resp interface{}, w http.ResponseWriter, r *http.Request) error {
	data, err := json.Marshal(resp)
	if err != nil {
		return err
	}
	writeResponse(w, code, data)
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
			resp := struct {
				Flavor nova.FlavorDetail `json:"flavor"`
			}{*flavor}
			return sendJSON(http.StatusOK, resp, w, r)
		}
		entities := n.allFlavorsAsEntities()
		if len(entities) == 0 {
			entities = []nova.Entity{}
		}
		resp := struct {
			Flavors []nova.Entity `json:"flavors"`
		}{entities}
		return sendJSON(http.StatusOK, resp, w, r)
	case "POST":
		if flavorId := path.Base(r.URL.Path); flavorId != "flavors" {
			return errNotFound
		}
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			return err
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
	}
	return fmt.Errorf("unknown request method %q for %s", r.Method, r.URL.Path)
}

// handleFlavorsDetail handles the flavors/detail HTTP API.
func (n *Nova) handleFlavorsDetail(w http.ResponseWriter, r *http.Request) error {
	switch r.Method {
	case "GET":
		if flavorId := path.Base(r.URL.Path); flavorId != "detail" {
			return errNotFound
		}
		flavors := n.allFlavors()
		if len(flavors) == 0 {
			flavors = []nova.FlavorDetail{}
		}
		resp := struct {
			Flavors []nova.FlavorDetail `json:"flavors"`
		}{flavors}
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
	}
	return fmt.Errorf("unknown request method %q for %s", r.Method, r.URL.Path)
}

// handleServerActions handles the servers/<id>/action HTTP API.
func (n *Nova) handleServerActions(server *nova.ServerDetail, w http.ResponseWriter, r *http.Request) error {
	if server == nil {
		return errNotFound
	}
	body, err := ioutil.ReadAll(r.Body)
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
		return err
	}
	switch {
	case action.AddSecurityGroup != nil:
		name := action.AddSecurityGroup.Name
		group, err := n.securityGroupByName(name)
		if err != nil || n.hasServerSecurityGroup(server.Id, group.Id) {
			return errNotFound
		}
		if err := n.addServerSecurityGroup(server.Id, group.Id); err != nil {
			return err
		}
		writeResponse(w, http.StatusNoContent, nil)
		return nil
	case action.RemoveSecurityGroup != nil:
		name := action.RemoveSecurityGroup.Name
		group, err := n.securityGroupByName(name)
		if err != nil || !n.hasServerSecurityGroup(server.Id, group.Id) {
			return errNotFound
		}
		if err := n.removeServerSecurityGroup(server.Id, group.Id); err != nil {
			return err
		}
		writeResponse(w, http.StatusNoContent, nil)
		return nil
	case action.AddFloatingIP != nil:
		addr := action.AddFloatingIP.Address
		if n.hasServerFloatingIP(server.Id, addr) {
			return errNotFound
		}
		fip, err := n.floatingIPByAddr(addr)
		if err != nil {
			return errNotFound
		}
		if err := n.addServerFloatingIP(server.Id, fip.Id); err != nil {
			return err
		}
		writeResponse(w, http.StatusNoContent, nil)
		return nil
	case action.RemoveFloatingIP != nil:
		addr := action.RemoveFloatingIP.Address
		if !n.hasServerFloatingIP(server.Id, addr) {
			return errNotFound
		}
		fip, err := n.floatingIPByAddr(addr)
		if err != nil {
			return errNotFound
		}
		if err := n.removeServerFloatingIP(server.Id, fip.Id); err != nil {
			return err
		}
		writeResponse(w, http.StatusNoContent, nil)
		return nil
	}
	return fmt.Errorf("unknown server action: %q", string(body))
}

// generateUUID generates a random UUID
// (taken from: http://www.ashishbanerjee.com/home/go/go-generate-uuid/0
func generateUUID() (string, error) {
	uuid := make([]byte, 16)
	n, err := rand.Read(uuid)
	if n != len(uuid) || err != nil {
		return "", err
	}
	// see RFC 4122
	uuid[8] = 0x80 // variant bits see page 5
	uuid[4] = 0x40 // version 4 Pseudo Random, see page 7
	return fmt.Sprintf("%x-%x-%x-%x-%x", uuid[0:4], uuid[4:6], uuid[6:8], uuid[8:10], uuid[10:]), nil
}

// handleRunServer handles creating and running a server.
func (n *Nova) handleRunServer(body []byte, w http.ResponseWriter, r *http.Request) error {
	var req struct {
		Server struct {
			FlavorRef      string
			ImageRef       string
			Name           string
			Metadata       map[string]string
			SecurityGroups []map[string]string `json:"security_groups"`
		}
	}
	if err := json.Unmarshal(body, &req); err != nil {
		return errBadRequest3
	}
	if req.Server.Name == "" {
		return errBadRequestSrvName
	}
	if req.Server.ImageRef == "" {
		return errBadRequestSrvImage
	}
	if req.Server.FlavorRef == "" {
		return errBadRequestSrvFlavor
	}
	id, err := generateUUID()
	if err != nil {
		return err
	}
	// TODO(dimitern): make sure flavor/image exist (if needed)
	flavor := nova.FlavorDetail{Id: req.Server.FlavorRef}
	n.buildFlavorLinks(&flavor)
	flavorEnt := nova.Entity{Id: flavor.Id, Links: flavor.Links}
	image := nova.Entity{Id: req.Server.ImageRef}
	server := nova.ServerDetail{
		Id:       id,
		Name:     req.Server.Name,
		TenantId: n.tenantId,
		Image:    image,
		Flavor:   flavorEnt,
		Status:   nova.StatusActive,
	}
	n.buildServerLinks(&server)
	if err := n.addServer(server); err != nil {
		return err
	}
	var resp struct {
		Server struct {
			SecurityGroups []map[string]string `json:"security_groups"`
			Id             string              `json:"id"`
			Links          []nova.Link         `json:"links"`
			AdminPass      string              `json:"adminPass"`
		} `json:"server"`
	}
	if len(req.Server.SecurityGroups) > 0 {
		errNoGroup := &errorResponse{
			http.StatusBadRequest,
			`{"badRequest": {"message": "Security group $SG$ not found for project $TENANT$.", "code": 400}}`,
			"application/json; charset=UTF-8",
			"bad request URL",
			nil,
		}
		for _, group := range req.Server.SecurityGroups {
			groupName := group["name"]
			if groupName == "default" {
				// assume default security group exists
				continue
			}
			if sg, err := n.securityGroupByName(groupName); err != nil {
				tmp := errNoGroup
				tmp.body = strings.Replace(tmp.body, "$SG$", groupName, -1)
				tmp.body = strings.Replace(tmp.body, "$TENANT$", n.tenantId, -1)
				return tmp
			} else if err := n.addServerSecurityGroup(id, sg.Id); err != nil {
				return err
			}
		}
		resp.Server.SecurityGroups = req.Server.SecurityGroups
	} else {
		resp.Server.SecurityGroups = make([]map[string]string, 1)
		groups := make(map[string]string)
		groups["name"] = "default"
		resp.Server.SecurityGroups[0] = groups
	}
	// TODO(dimitern): verify the security group names & add them to the server
	resp.Server.Id = id
	resp.Server.Links = server.Links
	resp.Server.AdminPass = "secret"
	return sendJSON(http.StatusAccepted, resp, w, r)
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
				srvGroups := n.allServerSecurityGroups(serverId)
				if len(srvGroups) == 0 {
					srvGroups = []nova.SecurityGroup{}
				}
				resp := struct {
					Groups []nova.SecurityGroup `json:"security_groups"`
				}{srvGroups}
				return sendJSON(http.StatusOK, resp, w, r)
			}
			resp := struct {
				Server nova.ServerDetail `json:"server"`
			}{*server}
			return sendJSON(http.StatusOK, resp, w, r)
		}
		var filter *nova.Filter
		if err := r.ParseForm(); err == nil && len(r.Form) > 0 {
			filter = &nova.Filter{r.Form}
		}
		entities := n.allServersAsEntities(filter)
		if len(entities) == 0 {
			entities = []nova.Entity{}
		}
		resp := struct {
			Servers []nova.Entity `json:"servers"`
		}{entities}
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
		if err != nil {
			return err
		}
		if len(body) == 0 {
			return errBadRequest2
		}
		return n.handleRunServer(body, w, r)
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
				return err
			}
			writeResponse(w, http.StatusNoContent, nil)
			return nil
		}
		return errNotFound
	}
	return fmt.Errorf("unknown request method %q for %s", r.Method, r.URL.Path)
}

// handleServersDetail handles the servers/detail HTTP API.
func (n *Nova) handleServersDetail(w http.ResponseWriter, r *http.Request) error {
	switch r.Method {
	case "GET":
		if serverId := path.Base(r.URL.Path); serverId != "detail" {
			return errNotFound
		}
		var filter *nova.Filter
		if err := r.ParseForm(); err == nil && len(r.Form) > 0 {
			filter = &nova.Filter{r.Form}
		}
		servers := n.allServers(filter)
		if len(servers) == 0 {
			servers = []nova.ServerDetail{}
		}
		resp := struct {
			Servers []nova.ServerDetail `json:"servers"`
		}{servers}
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
	}
	return fmt.Errorf("unknown request method %q for %s", r.Method, r.URL.Path)
}

// processGroupId returns the group id from the given request.
// If there was no group id specified in the path, it returns errNoGroupId
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
	return nil, errNoGroupId
}

// handleSecurityGroups handles the os-security-groups HTTP API.
func (n *Nova) handleSecurityGroups(w http.ResponseWriter, r *http.Request) error {
	switch r.Method {
	case "GET":
		group, err := n.processGroupId(w, r)
		if err == errNoGroupId {
			groups := n.allSecurityGroups()
			if len(groups) == 0 {
				groups = []nova.SecurityGroup{}
			}
			resp := struct {
				Groups []nova.SecurityGroup `json:"security_groups"`
			}{groups}
			return sendJSON(http.StatusOK, resp, w, r)
		}
		if err != nil {
			return err
		}
		resp := struct {
			Group nova.SecurityGroup `json:"security_group"`
		}{*group}
		return sendJSON(http.StatusOK, resp, w, r)
	case "POST":
		if groupId := path.Base(r.URL.Path); groupId != "os-security-groups" {
			return errNotFound
		}
		body, err := ioutil.ReadAll(r.Body)
		if err != nil || len(body) == 0 {
			return errBadRequest2
		}
		var req struct {
			Group struct {
				Name        string
				Description string
			} `json:"security_group"`
		}
		if err := json.Unmarshal(body, &req); err != nil {
			return err
		} else {
			n.nextGroupId++
			nextId := n.nextGroupId
			err = n.addSecurityGroup(nova.SecurityGroup{
				Id:          nextId,
				Name:        req.Group.Name,
				Description: req.Group.Description,
			})
			if err != nil {
				return err
			}
			group, err := n.securityGroup(nextId)
			if err != nil {
				return err
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
				return err
			}
			if n.nextGroupId > 0 {
				n.nextGroupId--
			}
			writeResponse(w, http.StatusNoContent, nil)
			return nil
		} else if err == errNoGroupId {
			return errNotFound
		} else {
			return err
		}
	}
	return fmt.Errorf("unknown request method %q for %s", r.Method, r.URL.Path)
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
		if err != nil || len(body) == 0 {
			return errBadRequest2
		}
		var req struct {
			Rule nova.RuleInfo `json:"security_group_rule"`
		}
		if err = json.Unmarshal(body, &req); err != nil {
			return err
		} else {
			n.nextRuleId++
			nextId := n.nextRuleId
			err = n.addSecurityGroupRule(nextId, req.Rule)
			if err != nil {
				return err
			}
			rule, err := n.securityGroupRule(nextId)
			if err != nil {
				return err
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
				return err
			}
			if n.nextRuleId > 0 {
				n.nextRuleId--
			}
			writeResponse(w, http.StatusNoContent, nil)
			return nil
		}
		return errNotFound
	}
	return fmt.Errorf("unknown request method %q for %s", r.Method, r.URL.Path)
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
			resp := struct {
				IP nova.FloatingIP `json:"floating_ip"`
			}{*fip}
			return sendJSON(http.StatusOK, resp, w, r)
		}
		fips := n.allFloatingIPs()
		if len(fips) == 0 {
			fips = []nova.FloatingIP{}
		}
		resp := struct {
			IPs []nova.FloatingIP `json:"floating_ips"`
		}{fips}
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
			return err
		}
		resp := struct {
			IP nova.FloatingIP `json:"floating_ip"`
		}{fip}
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
					writeResponse(w, http.StatusAccepted, nil)
					return nil
				}
			}
			return errNotFoundJSON
		}
		return errNotFound
	}
	return fmt.Errorf("unknown request method %q for %s", r.Method, r.URL.Path)
}

// setupHTTP attaches all the needed handlers to provide the HTTP API.
func (n *Nova) SetupHTTP(mux *http.ServeMux) {
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
