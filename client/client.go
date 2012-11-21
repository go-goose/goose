package client

import (
	"encoding/base64"
	"errors"
	"fmt"
	gooseerrors "launchpad.net/goose/errors"
	goosehttp "launchpad.net/goose/http"
	"launchpad.net/goose/identity"
	"net/http"
	"net/url"
)

const (
	OS_API_TOKENS               = "/tokens"
	OS_API_FLAVORS              = "/flavors"
	OS_API_FLAVORS_DETAIL       = "/flavors/detail"
	OS_API_SERVERS              = "/servers"
	OS_API_SERVERS_DETAIL       = "/servers/detail"
	OS_API_SECURITY_GROUPS      = "/os-security-groups"
	OS_API_SECURITY_GROUP_RULES = "/os-security-group-rules"
	OS_API_FLOATING_IPS         = "/os-floating-ips"

	GET    = "GET"
	POST   = "POST"
	PUT    = "PUT"
	DELETE = "DELETE"
	HEAD   = "HEAD"
	COPY   = "COPY"
)

type OpenStackClient struct {

	client *goosehttp.GooseHTTPClient

	creds *identity.Credentials
	auth identity.Authenticator

	//TODO - store service urls by region.
	ServiceURLs map[string]string
	TokenId     string
	TenantId    string
	UserId      string
}

func NewOpenStackClient(creds *identity.Credentials, auth_method int) *OpenStackClient {
	client := OpenStackClient{creds: creds}
	client.creds.URL = client.creds.URL + OS_API_TOKENS
	switch auth_method {
	default: panic(fmt.Errorf("Invalid identity authorisation method: %d", auth_method))
	case identity.AUTH_LEGACY:
		client.auth = &identity.Legacy{}
	case identity.AUTH_USERPASS:
		client.auth = &identity.UserPass{}
	}
	return &client
}

func (c *OpenStackClient) Authenticate() (err error) {
	err = nil
	if c.auth == nil {
		return fmt.Errorf("Authentication method has not been specified")
	}
	authDetails, err := c.auth.Auth(c.creds)
	if err != nil {
		gooseerrors.AddErrorContext(&err, "authentication failed")
		return
	}

	c.TokenId = authDetails.TokenId
	c.TenantId = authDetails.TenantId
	c.UserId = authDetails.UserId
	c.ServiceURLs = authDetails.ServiceURLs
	return nil
}

func (c *OpenStackClient) IsAuthenticated() bool {
	return c.TokenId != ""
}

type Link struct {
	Href string
	Rel  string
	Type string
}

// Entity can describe a flavor, flavor detail or server.
// Contains a list of links.
type Entity struct {
	Id    string
	Links []Link
	Name  string
}

func (c *OpenStackClient) ListFlavors() (flavors []Entity, err error) {

	var resp struct {
		Flavors []Entity
	}
	requestData := goosehttp.RequestData{RespValue: &resp}
	err = c.authRequest(GET, "compute", OS_API_FLAVORS, nil, &requestData)
	if err != nil {
		gooseerrors.AddErrorContext(&err, "failed to get list of flavors")
		return
	}

	return resp.Flavors, nil
}

type FlavorDetail struct {
	Name  string
	RAM   int
	VCPUs int
	Disk  int
	Id    string
	Swap  interface{} // Can be an empty string (?!)
}

func (c *OpenStackClient) ListFlavorsDetail() (flavors []FlavorDetail, err error) {

	var resp struct {
		Flavors []FlavorDetail
	}
	requestData := goosehttp.RequestData{RespValue: &resp}
	err = c.authRequest(GET, "compute", OS_API_FLAVORS_DETAIL, nil, &requestData)
	if err != nil {
		gooseerrors.AddErrorContext(&err, "failed to get list of flavors details")
		return
	}

	return resp.Flavors, nil
}

func (c *OpenStackClient) ListServers() (servers []Entity, err error) {

	var resp struct {
		Servers []Entity
	}
	requestData := goosehttp.RequestData{RespValue: &resp, ExpectedStatus: []int{http.StatusOK}}
	err = c.authRequest(GET, "compute", OS_API_SERVERS, nil, &requestData)
	if err != nil {
		gooseerrors.AddErrorContext(&err, "failed to get list of servers")
		return
	}
	return resp.Servers, nil
}

type ServerDetail struct {
	AddressIPv4 string
	AddressIPv6 string
	Created     string
	Flavor      Entity
	HostId      string
	Id          string
	Image       Entity
	Links       []Link
	Name        string
	Progress    int
	Status      string
	TenantId    string `json:"tenant_id"`
	Updated     string
	UserId      string `json:"user_id"`
}

func (c *OpenStackClient) ListServersDetail() (servers []ServerDetail, err error) {

	var resp struct {
		Servers []ServerDetail
	}
	requestData := goosehttp.RequestData{RespValue: &resp}
	err = c.authRequest(GET, "compute", OS_API_SERVERS_DETAIL, nil, &requestData)
	if err != nil {
		gooseerrors.AddErrorContext(&err, "failed to get list of servers details")
		return
	}

	return resp.Servers, nil
}

func (c *OpenStackClient) GetServer(serverId string) (ServerDetail, error) {

	var resp struct {
		Server ServerDetail
	}
	url := fmt.Sprintf("%s/%s", OS_API_SERVERS, serverId)
	requestData := goosehttp.RequestData{RespValue: &resp}
	err := c.authRequest(GET, "compute", url, nil, &requestData)
	if err != nil {
		gooseerrors.AddErrorContext(&err, "failed to get details for serverId=%s", serverId)
		return ServerDetail{}, err
	}

	return resp.Server, nil
}

func (c *OpenStackClient) DeleteServer(serverId string) error {

	var resp struct {
		Server ServerDetail
	}
	url := fmt.Sprintf("%s/%s", OS_API_SERVERS, serverId)
	requestData := goosehttp.RequestData{RespValue: &resp, ExpectedStatus: []int{http.StatusNoContent}}
	err := c.authRequest(DELETE, "compute", url, nil, &requestData)
	if err != nil {
		gooseerrors.AddErrorContext(&err, "failed to delete server with serverId=%s", serverId)
		return err
	}

	return nil
}

type RunServerOpts struct {
	Name               string  `json:"name"`
	FlavorId           string  `json:"flavorRef"`
	ImageId            string  `json:"imageRef"`
	UserData           *string `json:"user_data"`
	SecurityGroupNames []struct {
		Name string `json:"name"`
	} `json:"security_groups"`
}

func (c *OpenStackClient) RunServer(opts RunServerOpts) (err error) {

	var req struct {
		Server RunServerOpts `json:"server"`
	}
	req.Server = opts
	if opts.UserData != nil {
		data := []byte(*opts.UserData)
		encoded := base64.StdEncoding.EncodeToString(data)
		req.Server.UserData = &encoded
	}
	requestData := goosehttp.RequestData{ReqValue: req, ExpectedStatus: []int{http.StatusAccepted}}
	err = c.authRequest(POST, "compute", OS_API_SERVERS, nil, &requestData)
	if err != nil {
		gooseerrors.AddErrorContext(&err, "failed to run a server with %#v", opts)
	}

	return
}

type SecurityGroupRule struct {
	FromPort      *int              `json:"from_port"`   // Can be nil
	IPProtocol    *string           `json:"ip_protocol"` // Can be nil
	ToPort        *int              `json:"to_port"`     // Can be nil
	ParentGroupId int               `json:"parent_group_id"`
	IPRange       map[string]string `json:"ip_range"` // Can be empty
	Id            int
	Group         map[string]string // Can be empty
}

type SecurityGroup struct {
	Rules       []SecurityGroupRule
	TenantId    string `json:"tenant_id"`
	Id          int
	Name        string
	Description string
}

func (c *OpenStackClient) ListSecurityGroups() (groups []SecurityGroup, err error) {

	var resp struct {
		Groups []SecurityGroup `json:"security_groups"`
	}
	requestData := goosehttp.RequestData{RespValue: &resp}
	err = c.authRequest(GET, "compute", OS_API_SECURITY_GROUPS, nil, &requestData)
	if err != nil {
		gooseerrors.AddErrorContext(&err, "failed to list security groups")
		return nil, err
	}

	return resp.Groups, nil
}

func (c *OpenStackClient) GetServerSecurityGroups(serverId string) (groups []SecurityGroup, err error) {

	var resp struct {
		Groups []SecurityGroup `json:"security_groups"`
	}
	url := fmt.Sprintf("%s/%s/%s", OS_API_SERVERS, serverId, OS_API_SECURITY_GROUPS)
	requestData := goosehttp.RequestData{RespValue: &resp}
	err = c.authRequest(GET, "compute", url, nil, &requestData)
	if err != nil {
		gooseerrors.AddErrorContext(&err, "failed to list server (%s) security groups", serverId)
		return nil, err
	}

	return resp.Groups, nil
}

func (c *OpenStackClient) CreateSecurityGroup(name, description string) (group SecurityGroup, err error) {

	var req struct {
		SecurityGroup struct {
			Name        string `json:"name"`
			Description string `json:"description"`
		} `json:"security_group"`
	}
	req.SecurityGroup.Name = name
	req.SecurityGroup.Description = description

	var resp struct {
		SecurityGroup SecurityGroup `json:"security_group"`
	}
	requestData := goosehttp.RequestData{ReqValue: req, RespValue: &resp, ExpectedStatus: []int{http.StatusOK}}
	err = c.authRequest(POST, "compute", OS_API_SECURITY_GROUPS, nil, &requestData)
	if err != nil {
		gooseerrors.AddErrorContext(&err, "failed to create a security group with name=%s", name)
	}
	group = resp.SecurityGroup

	return
}

func (c *OpenStackClient) DeleteSecurityGroup(groupId int) (err error) {

	url := fmt.Sprintf("%s/%d", OS_API_SECURITY_GROUPS, groupId)
	requestData := goosehttp.RequestData{ExpectedStatus: []int{http.StatusAccepted}}
	err = c.authRequest(DELETE, "compute", url, nil, &requestData)
	if err != nil {
		gooseerrors.AddErrorContext(&err, "failed to delete a security group with id=%d", groupId)
	}

	return
}

type RuleInfo struct {
	IPProtocol    string `json:"ip_protocol"`     // Required, if GroupId is nil
	FromPort      int    `json:"from_port"`       // Required, if GroupId is nil
	ToPort        int    `json:"to_port"`         // Required, if GroupId is nil
	Cidr          string `json:"cidr"`            // Required, if GroupId is nil
	GroupId       *int   `json:"group_id"`        // If nil, FromPort/ToPort/IPProtocol must be set
	ParentGroupId int    `json:"parent_group_id"` // Required always
}

func (c *OpenStackClient) CreateSecurityGroupRule(ruleInfo RuleInfo) (rule SecurityGroupRule, err error) {

	var req struct {
		SecurityGroupRule RuleInfo `json:"security_group_rule"`
	}
	req.SecurityGroupRule = ruleInfo

	var resp struct {
		SecurityGroupRule SecurityGroupRule `json:"security_group_rule"`
	}

	requestData := goosehttp.RequestData{ReqValue: req, RespValue: &resp}
	err = c.authRequest(POST, "compute", OS_API_SECURITY_GROUP_RULES, nil, &requestData)
	if err != nil {
		gooseerrors.AddErrorContext(&err, "failed to create a rule for the security group with id=%s", ruleInfo.GroupId)
	}

	return resp.SecurityGroupRule, err
}

func (c *OpenStackClient) DeleteSecurityGroupRule(ruleId int) (err error) {

	url := fmt.Sprintf("%s/%d", OS_API_SECURITY_GROUP_RULES, ruleId)
	requestData := goosehttp.RequestData{ExpectedStatus: []int{http.StatusAccepted}}
	err = c.authRequest(DELETE, "compute", url, nil, &requestData)
	if err != nil {
		gooseerrors.AddErrorContext(&err, "failed to delete a security group rule with id=%d", ruleId)
	}

	return
}

func (c *OpenStackClient) AddServerSecurityGroup(serverId, groupName string) (err error) {

	var req struct {
		AddSecurityGroup struct {
			Name string `json:"name"`
		} `json:"addSecurityGroup"`
	}
	req.AddSecurityGroup.Name = groupName

	url := fmt.Sprintf("%s/%s/action", OS_API_SERVERS, serverId)
	requestData := goosehttp.RequestData{ReqValue: req, ExpectedStatus: []int{http.StatusAccepted}}
	err = c.authRequest(POST, "compute", url, nil, &requestData)
	if err != nil {
		gooseerrors.AddErrorContext(&err, "failed to add security group '%s' from server with id=%s", groupName, serverId)
	}
	return
}

func (c *OpenStackClient) RemoveServerSecurityGroup(serverId, groupName string) (err error) {

	var req struct {
		RemoveSecurityGroup struct {
			Name string `json:"name"`
		} `json:"removeSecurityGroup"`
	}
	req.RemoveSecurityGroup.Name = groupName

	url := fmt.Sprintf("%s/%s/action", OS_API_SERVERS, serverId)
	requestData := goosehttp.RequestData{ReqValue: req, ExpectedStatus: []int{http.StatusAccepted}}
	err = c.authRequest(POST, "compute", url, nil, &requestData)
	if err != nil {
		gooseerrors.AddErrorContext(&err, "failed to remove security group '%s' from server with id=%s", groupName, serverId)
	}
	return
}

type FloatingIP struct {
	FixedIP    interface{} `json:"fixed_ip"` // Can be a string or null
	Id         int         `json:"id"`
	InstanceId interface{} `json:"instance_id"` // Can be a string or null
	IP         string      `json:"ip"`
	Pool       string      `json:"pool"`
}

func (c *OpenStackClient) ListFloatingIPs() (ips []FloatingIP, err error) {

	var resp struct {
		FloatingIPs []FloatingIP `json:"floating_ips"`
	}

	requestData := goosehttp.RequestData{RespValue: &resp}
	err = c.authRequest(GET, "compute", OS_API_FLOATING_IPS, nil, &requestData)
	if err != nil {
		gooseerrors.AddErrorContext(&err, "failed to list floating ips")
	}

	return resp.FloatingIPs, err
}

func (c *OpenStackClient) GetFloatingIP(ipId int) (ip FloatingIP, err error) {

	var resp struct {
		FloatingIP FloatingIP `json:"floating_ip"`
	}

	url := fmt.Sprintf("%s/%d", OS_API_FLOATING_IPS, ipId)
	requestData := goosehttp.RequestData{RespValue: &resp}
	err = c.authRequest(GET, "compute", url, nil, &requestData)
	if err != nil {
		gooseerrors.AddErrorContext(&err, "failed to get floating ip %d details", ipId)
	}

	return resp.FloatingIP, err
}

func (c *OpenStackClient) AllocateFloatingIP() (ip FloatingIP, err error) {

	var resp struct {
		FloatingIP FloatingIP `json:"floating_ip"`
	}

	requestData := goosehttp.RequestData{RespValue: &resp}
	err = c.authRequest(POST, "compute", OS_API_FLOATING_IPS, nil, &requestData)
	if err != nil {
		gooseerrors.AddErrorContext(&err, "failed to allocate a floating ip")
	}

	return resp.FloatingIP, err
}

func (c *OpenStackClient) DeleteFloatingIP(ipId int) (err error) {

	url := fmt.Sprintf("%s/%d", OS_API_FLOATING_IPS, ipId)
	requestData := goosehttp.RequestData{ExpectedStatus: []int{http.StatusAccepted}}
	err = c.authRequest(DELETE, "compute", url, nil, &requestData)
	if err != nil {
		gooseerrors.AddErrorContext(&err, "failed to delete floating ip %d details", ipId)
	}

	return
}

func (c *OpenStackClient) AddServerFloatingIP(serverId, address string) (err error) {

	var req struct {
		AddFloatingIP struct {
			Address string `json:"address"`
		} `json:"addFloatingIp"`
	}
	req.AddFloatingIP.Address = address

	url := fmt.Sprintf("%s/%s/action", OS_API_SERVERS, serverId)
	requestData := goosehttp.RequestData{ReqValue: req, ExpectedStatus: []int{http.StatusAccepted}}
	err = c.authRequest(POST, "compute", url, nil, &requestData)
	if err != nil {
		gooseerrors.AddErrorContext(&err, "failed to add floating ip %s to server %s", address, serverId)
	}

	return
}

func (c *OpenStackClient) RemoveServerFloatingIP(serverId, address string) (err error) {

	var req struct {
		RemoveFloatingIP struct {
			Address string `json:"address"`
		} `json:"removeFloatingIp"`
	}
	req.RemoveFloatingIP.Address = address

	url := fmt.Sprintf("%s/%s/action", OS_API_SERVERS, serverId)
	requestData := goosehttp.RequestData{ReqValue: req, ExpectedStatus: []int{http.StatusAccepted}}
	err = c.authRequest(POST, "compute", url, nil, &requestData)
	if err != nil {
		gooseerrors.AddErrorContext(&err, "failed to remove floating ip %s to server %s", address, serverId)
	}

	return
}

func (c *OpenStackClient) CreateContainer(containerName string) (err error) {

	// Juju expects there to be a (semi) public url for some objects. This
	// could probably be more restrictive or placed in a seperate container
	// with some refactoring, but for now just make everything public.
	headers := make(http.Header)
	headers.Add("X-Container-Read", ".r:*")
	url := fmt.Sprintf("/%s", containerName)
	requestData := goosehttp.RequestData{ReqHeaders: headers, ExpectedStatus: []int{http.StatusAccepted, http.StatusCreated}}
	err = c.authRequest(PUT, "object-store", url, nil, &requestData)
	if err != nil {
		gooseerrors.AddErrorContext(&err, "failed to create container %s.", containerName)
	}

	return
}

func (c *OpenStackClient) DeleteContainer(containerName string) (err error) {

	url := fmt.Sprintf("/%s", containerName)
	requestData := goosehttp.RequestData{ExpectedStatus: []int{http.StatusNoContent}}
	err = c.authRequest(DELETE, "object-store", url, nil, &requestData)
	if err != nil {
		gooseerrors.AddErrorContext(&err, "failed to delete container %s.", containerName)
	}

	return
}

func (c *OpenStackClient) PublicObjectURL(containerName, objectName string) (url string, err error) {
	path := fmt.Sprintf("/%s/%s", containerName, objectName)
	return c.makeUrl("object-store", []string{path}, nil)
}

func (c *OpenStackClient) HeadObject(containerName, objectName string) (headers http.Header, err error) {

	url, err := c.PublicObjectURL(containerName, objectName)
	if err != nil {
		return nil, err
	}
	requestData := goosehttp.RequestData{ReqHeaders: headers, ExpectedStatus: []int{http.StatusOK}}
	err = c.authRequest(HEAD, "object-store", url, nil, &requestData)
	if err != nil {
		gooseerrors.AddErrorContext(&err, "failed to HEAD object %s from container %s", objectName, containerName)
		return nil, err
	}
	return headers, nil
}

func (c *OpenStackClient) GetObject(containerName, objectName string) (obj []byte, err error) {

	url, err := c.PublicObjectURL(containerName, objectName)
	if err != nil {
		return nil, err
	}
	requestData := goosehttp.RequestData{RespData: &obj, ExpectedStatus: []int{http.StatusOK}}
	err = c.authBinaryRequest(GET, "object-store", url, nil, &requestData)
	if err != nil {
		gooseerrors.AddErrorContext(&err, "failed to GET object %s content from container %s", objectName, containerName)
		return nil, err
	}
	return obj, nil
}

func (c *OpenStackClient) DeleteObject(containerName, objectName string) (err error) {

	url, err := c.PublicObjectURL(containerName, objectName)
	if err != nil {
		return err
	}
	requestData := goosehttp.RequestData{ExpectedStatus: []int{http.StatusAccepted}}
	err = c.authRequest(DELETE, "object-store", url, nil, &requestData)
	if err != nil {
		gooseerrors.AddErrorContext(&err, "failed to DELETE object %s content from container %s", objectName, containerName)
	}
	return err
}

func (c *OpenStackClient) PutObject(containerName, objectName string, data []byte) (err error) {

	url, err := c.PublicObjectURL(containerName, objectName)
	if err != nil {
		return err
	}
	requestData := goosehttp.RequestData{ReqData: data, ExpectedStatus: []int{http.StatusAccepted}}
	err = c.authBinaryRequest(PUT, "object-store", url, nil, &requestData)
	if err != nil {
		gooseerrors.AddErrorContext(&err, "failed to PUT object %s content from container %s", objectName, containerName)
	}
	return err
}

////////////////////////////////////////////////////////////////////////
// Private helpers

// makeUrl prepares a full URL to a service endpoint, with optional
// URL parts, appended to it and optional query string params. It
// uses the first endpoint it can find for the given service type
func (c *OpenStackClient) makeUrl(serviceType string, parts []string, params url.Values) (string, error) {
	url, ok := c.ServiceURLs[serviceType]
	if !ok {
		return "", errors.New("no endpoints known for service type: " + serviceType)
	}
	for _, part := range parts {
		url += part
	}
	if params != nil {
		url += "?" + params.Encode()
	}
	return url, nil
}

func (c *OpenStackClient) setupRequest(svcType, apiCall string, params url.Values, requestData *goosehttp.RequestData) (url string, err error) {
	if !c.IsAuthenticated() {
		return "", errors.New("not authenticated")
	}

	url, err = c.makeUrl(svcType, []string{apiCall}, params)
	if err != nil {
		gooseerrors.AddErrorContext(&err, "cannot find a '%s' node endpoint", svcType)
		return
	}

	if c.client == nil {
		c.client = &goosehttp.GooseHTTPClient{http.Client{CheckRedirect: nil}}
	}

	if requestData.ReqHeaders == nil {
		requestData.ReqHeaders = make(http.Header)
	}
	requestData.ReqHeaders.Add("X-Auth-Token", c.TokenId)
	return
}

func (c *OpenStackClient) authRequest(method, svcType, apiCall string, params url.Values, requestData *goosehttp.RequestData) (err error) {
	url, err := c.setupRequest(svcType, apiCall, params, requestData)
	if err != nil {
		return
	}
	err = c.client.JsonRequest(method, url, requestData)
	return
}

func (c *OpenStackClient) authBinaryRequest(method, svcType, apiCall string, params url.Values, requestData *goosehttp.RequestData) (err error) {
	url, err := c.setupRequest(svcType, apiCall, params, requestData)
	if err != nil {
		return
	}
	err = c.client.BinaryRequest(method, url, requestData)
	return
}
