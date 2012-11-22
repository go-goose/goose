package client

import (
	"encoding/base64"
	"errors"
	"fmt"
	gooseerrors "launchpad.net/goose/errors"
	goosehttp "launchpad.net/goose/http"
	"launchpad.net/goose/identity"
	"net/http"
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
	client *goosehttp.Client

	creds *identity.Credentials
	auth  identity.Authenticator

	//TODO - store service urls by region.
	ServiceURLs map[string]string
	Token       string
	TenantId    string
	UserId      string
}

func NewOpenStackClient(creds *identity.Credentials, auth_method identity.AuthMethod) *OpenStackClient {
	client := OpenStackClient{creds: creds}
	client.creds.URL = client.creds.URL + OS_API_TOKENS
	switch auth_method {
	default:
		panic(fmt.Errorf("Invalid identity authorisation method: %d", auth_method))
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
		err = gooseerrors.AddContext(err, "authentication failed")
		return
	}

	c.Token = authDetails.Token
	c.TenantId = authDetails.TenantId
	c.UserId = authDetails.UserId
	c.ServiceURLs = authDetails.ServiceURLs
	return nil
}

func (c *OpenStackClient) IsAuthenticated() bool {
	return c.Token != ""
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
	err = c.sendRequest(GET, "compute", OS_API_FLAVORS, &requestData, "failed to get list of flavors")
	return resp.Flavors, err
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
	err = c.sendRequest(GET, "compute", OS_API_FLAVORS_DETAIL, &requestData,
		"failed to get list of flavors details")
	return resp.Flavors, err
}

func (c *OpenStackClient) ListServers() (servers []Entity, err error) {

	var resp struct {
		Servers []Entity
	}
	requestData := goosehttp.RequestData{RespValue: &resp, ExpectedStatus: []int{http.StatusOK}}
	err = c.sendRequest(GET, "compute", OS_API_SERVERS, &requestData,
		"failed to get list of servers")
	return resp.Servers, err
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
	err = c.sendRequest(GET, "compute", OS_API_SERVERS_DETAIL, &requestData,
		"failed to get list of servers details")
	return resp.Servers, err
}

func (c *OpenStackClient) GetServer(serverId string) (ServerDetail, error) {

	var resp struct {
		Server ServerDetail
	}
	url := fmt.Sprintf("%s/%s", OS_API_SERVERS, serverId)
	requestData := goosehttp.RequestData{RespValue: &resp}
	err := c.sendRequest(GET, "compute", url, &requestData,
		"failed to get details for serverId=%s", serverId)
	return resp.Server, err
}

func (c *OpenStackClient) DeleteServer(serverId string) (err error) {

	var resp struct {
		Server ServerDetail
	}
	url := fmt.Sprintf("%s/%s", OS_API_SERVERS, serverId)
	requestData := goosehttp.RequestData{RespValue: &resp, ExpectedStatus: []int{http.StatusNoContent}}
	err = c.sendRequest(DELETE, "compute", url, &requestData,
		"failed to delete server with serverId=%s", serverId)
	return
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
	err = c.sendRequest(POST, "compute", OS_API_SERVERS, &requestData,
		"failed to run a server with %#v", opts)
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
	err = c.sendRequest(GET, "compute", OS_API_SECURITY_GROUPS, &requestData,
		"failed to list security groups")
	return resp.Groups, err
}

func (c *OpenStackClient) GetServerSecurityGroups(serverId string) (groups []SecurityGroup, err error) {

	var resp struct {
		Groups []SecurityGroup `json:"security_groups"`
	}
	url := fmt.Sprintf("%s/%s/%s", OS_API_SERVERS, serverId, OS_API_SECURITY_GROUPS)
	requestData := goosehttp.RequestData{RespValue: &resp}
	err = c.sendRequest(GET, "compute", url, &requestData,
		"failed to list server (%s) security groups", serverId)
	return resp.Groups, err
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
	err = c.sendRequest(POST, "compute", OS_API_SECURITY_GROUPS, &requestData,
		"failed to create a security group with name=%s", name)
	return resp.SecurityGroup, err
}

func (c *OpenStackClient) DeleteSecurityGroup(groupId int) (err error) {

	url := fmt.Sprintf("%s/%d", OS_API_SECURITY_GROUPS, groupId)
	requestData := goosehttp.RequestData{ExpectedStatus: []int{http.StatusAccepted}}
	err = c.sendRequest(DELETE, "compute", url, &requestData,
		"failed to delete a security group with id=%d", groupId)
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
	err = c.sendRequest(POST, "compute", OS_API_SECURITY_GROUP_RULES, &requestData,
		"failed to create a rule for the security group with id=%s", ruleInfo.GroupId)
	return resp.SecurityGroupRule, err
}

func (c *OpenStackClient) DeleteSecurityGroupRule(ruleId int) (err error) {

	url := fmt.Sprintf("%s/%d", OS_API_SECURITY_GROUP_RULES, ruleId)
	requestData := goosehttp.RequestData{ExpectedStatus: []int{http.StatusAccepted}}
	err = c.sendRequest(DELETE, "compute", url, &requestData,
		"failed to delete a security group rule with id=%d", ruleId)
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
	err = c.sendRequest(POST, "compute", url, &requestData,
		"failed to add security group '%s' from server with id=%s", groupName, serverId)
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
	err = c.sendRequest(POST, "compute", url, &requestData,
		"failed to remove security group '%s' from server with id=%s", groupName, serverId)
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
	err = c.sendRequest(GET, "compute", OS_API_FLOATING_IPS, &requestData,
		"failed to list floating ips")
	return resp.FloatingIPs, err
}

func (c *OpenStackClient) GetFloatingIP(ipId int) (ip FloatingIP, err error) {

	var resp struct {
		FloatingIP FloatingIP `json:"floating_ip"`
	}

	url := fmt.Sprintf("%s/%d", OS_API_FLOATING_IPS, ipId)
	requestData := goosehttp.RequestData{RespValue: &resp}
	err = c.sendRequest(GET, "compute", url, &requestData,
		"failed to get floating ip %d details", ipId)
	return resp.FloatingIP, err
}

func (c *OpenStackClient) AllocateFloatingIP() (ip FloatingIP, err error) {

	var resp struct {
		FloatingIP FloatingIP `json:"floating_ip"`
	}

	requestData := goosehttp.RequestData{RespValue: &resp}
	err = c.sendRequest(POST, "compute", OS_API_FLOATING_IPS, &requestData,
		"failed to allocate a floating ip")
	return resp.FloatingIP, err
}

func (c *OpenStackClient) DeleteFloatingIP(ipId int) (err error) {

	url := fmt.Sprintf("%s/%d", OS_API_FLOATING_IPS, ipId)
	requestData := goosehttp.RequestData{ExpectedStatus: []int{http.StatusAccepted}}
	err = c.sendRequest(DELETE, "compute", url, &requestData,
		"failed to delete floating ip %d details", ipId)
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
	err = c.sendRequest(POST, "compute", url, &requestData,
		"failed to add floating ip %s to server %s", address, serverId)
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
	err = c.sendRequest(POST, "compute", url, &requestData,
		"failed to remove floating ip %s to server %s", address, serverId)
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
	err = c.sendRequest(PUT, "object-store", url, &requestData,
		"failed to create container %s.", containerName)
	return
}

func (c *OpenStackClient) DeleteContainer(containerName string) (err error) {

	url := fmt.Sprintf("/%s", containerName)
	requestData := goosehttp.RequestData{ExpectedStatus: []int{http.StatusNoContent}}
	err = c.sendRequest(DELETE, "object-store", url, &requestData,
		"failed to delete container %s.", containerName)
	return
}

func (c *OpenStackClient) PublicObjectURL(containerName, objectName string) (url string, err error) {
	path := fmt.Sprintf("/%s/%s", containerName, objectName)
	return c.makeServiceURL("object-store", []string{path})
}

func (c *OpenStackClient) HeadObject(containerName, objectName string) (headers http.Header, err error) {

	url, err := c.PublicObjectURL(containerName, objectName)
	if err != nil {
		return nil, err
	}
	requestData := goosehttp.RequestData{ReqHeaders: headers, ExpectedStatus: []int{http.StatusOK}}
	err = c.sendRequest(HEAD, "object-store", url, &requestData,
		"failed to HEAD object %s from container %s", objectName, containerName)
	return headers, err
}

func (c *OpenStackClient) GetObject(containerName, objectName string) (obj []byte, err error) {

	url, err := c.PublicObjectURL(containerName, objectName)
	if err != nil {
		return nil, err
	}
	requestData := goosehttp.RequestData{RespData: &obj, ExpectedStatus: []int{http.StatusOK}}
	err = c.sendRequest(GET, "object-store", url, &requestData,
		"failed to GET object %s content from container %s", objectName, containerName)
	return obj, err
}

func (c *OpenStackClient) DeleteObject(containerName, objectName string) (err error) {

	url, err := c.PublicObjectURL(containerName, objectName)
	if err != nil {
		return err
	}
	requestData := goosehttp.RequestData{ExpectedStatus: []int{http.StatusAccepted}}
	err = c.sendRequest(DELETE, "object-store", url, &requestData,
		"failed to DELETE object %s content from container %s", objectName, containerName)
	return err
}

func (c *OpenStackClient) PutObject(containerName, objectName string, data []byte) (err error) {

	url, err := c.PublicObjectURL(containerName, objectName)
	if err != nil {
		return err
	}
	requestData := goosehttp.RequestData{ReqData: data, ExpectedStatus: []int{http.StatusAccepted}}
	err = c.sendRequest(PUT, "object-store", url, &requestData,
		"failed to PUT object %s content from container %s", objectName, containerName)
	return err
}

////////////////////////////////////////////////////////////////////////
// Private helpers

// makeServiceURL prepares a full URL to a service endpoint, with optional
// URL parts. It uses the first endpoint it can find for the given service type.
func (c *OpenStackClient) makeServiceURL(serviceType string, parts []string) (string, error) {
	url, ok := c.ServiceURLs[serviceType]
	if !ok {
		return "", errors.New("no endpoints known for service type: " + serviceType)
	}
	for _, part := range parts {
		url += part
	}
	return url, nil
}

func (c *OpenStackClient) sendRequest(method, svcType, apiCall string, requestData *goosehttp.RequestData,
	context string, contextArgs ...interface{}) (err error) {
	if !c.IsAuthenticated() {
		err = gooseerrors.AddContext(errors.New("not authenticated"), context, contextArgs...)
		return
	}

	url, err := c.makeServiceURL(svcType, []string{apiCall})
	if err != nil {
		err = gooseerrors.AddContext(err, "cannot find a '%s' node endpoint", svcType)
		return
	}

	if c.client == nil {
		c.client = &goosehttp.Client{http.Client{CheckRedirect: nil}, c.Token}
	}
	if requestData.ReqValue != nil || requestData.RespValue != nil {
		err = c.client.JsonRequest(method, url, requestData)
	} else {
		err = c.client.BinaryRequest(method, url, requestData)
	}
	if err != nil {
		err = gooseerrors.AddContext(err, context, contextArgs...)
	}
	return
}
