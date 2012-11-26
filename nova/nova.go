package nova

import (
	"encoding/base64"
	"fmt"
	"launchpad.net/goose/client"
	goosehttp "launchpad.net/goose/http"
	"net/http"
)

const (
	OS_API_FLAVORS              = "/flavors"
	OS_API_FLAVORS_DETAIL       = "/flavors/detail"
	OS_API_SERVERS              = "/servers"
	OS_API_SERVERS_DETAIL       = "/servers/detail"
	OS_API_SECURITY_GROUPS      = "/os-security-groups"
	OS_API_SECURITY_GROUP_RULES = "/os-security-group-rules"
	OS_API_FLOATING_IPS         = "/os-floating-ips"
)

// Provide access to the OpenStack Compute service.
type NovaClient interface {
	ListFlavors() (flavors []Entity, err error)

	ListFlavorsDetail() (flavors []FlavorDetail, err error)

	ListServers() (servers []Entity, err error)

	ListServersDetail() (servers []ServerDetail, err error)

	GetServer(serverId string) (ServerDetail, error)

	DeleteServer(serverId string) (err error)

	RunServer(opts RunServerOpts) (err error)

	ListSecurityGroups() (groups []SecurityGroup, err error)

	GetServerSecurityGroups(serverId string) (groups []SecurityGroup, err error)

	CreateSecurityGroup(name, description string) (group SecurityGroup, err error)

	DeleteSecurityGroup(groupId int) (err error)

	CreateSecurityGroupRule(ruleInfo RuleInfo) (rule SecurityGroupRule, err error)

	DeleteSecurityGroupRule(ruleId int) (err error)

	AddServerSecurityGroup(serverId, groupName string) (err error)

	RemoveServerSecurityGroup(serverId, groupName string) (err error)

	ListFloatingIPs() (ips []FloatingIP, err error)

	GetFloatingIP(ipId int) (ip FloatingIP, err error)

	AllocateFloatingIP() (ip FloatingIP, err error)

	DeleteFloatingIP(ipId int) (err error)

	AddServerFloatingIP(serverId, address string) (err error)

	RemoveServerFloatingIP(serverId, address string) (err error)
}

type OpenStackNovaClient struct {
	client client.Client
}

func NewNovaClient(client client.Client) NovaClient {
	n := &OpenStackNovaClient{client}
	return n
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

func (n *OpenStackNovaClient) ListFlavors() (flavors []Entity, err error) {

	var resp struct {
		Flavors []Entity
	}
	requestData := goosehttp.RequestData{RespValue: &resp}
	err = n.client.SendRequest(client.GET, "compute", OS_API_FLAVORS, &requestData, "failed to get list of flavors")
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

func (n *OpenStackNovaClient) ListFlavorsDetail() (flavors []FlavorDetail, err error) {

	var resp struct {
		Flavors []FlavorDetail
	}
	requestData := goosehttp.RequestData{RespValue: &resp}
	err = n.client.SendRequest(client.GET, "compute", OS_API_FLAVORS_DETAIL, &requestData,
		"failed to get list of flavors details")
	return resp.Flavors, err
}

func (n *OpenStackNovaClient) ListServers() (servers []Entity, err error) {

	var resp struct {
		Servers []Entity
	}
	requestData := goosehttp.RequestData{RespValue: &resp, ExpectedStatus: []int{http.StatusOK}}
	err = n.client.SendRequest(client.GET, "compute", OS_API_SERVERS, &requestData,
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

func (n *OpenStackNovaClient) ListServersDetail() (servers []ServerDetail, err error) {

	var resp struct {
		Servers []ServerDetail
	}
	requestData := goosehttp.RequestData{RespValue: &resp}
	err = n.client.SendRequest(client.GET, "compute", OS_API_SERVERS_DETAIL, &requestData,
		"failed to get list of servers details")
	return resp.Servers, err
}

func (n *OpenStackNovaClient) GetServer(serverId string) (ServerDetail, error) {

	var resp struct {
		Server ServerDetail
	}
	url := fmt.Sprintf("%s/%s", OS_API_SERVERS, serverId)
	requestData := goosehttp.RequestData{RespValue: &resp}
	err := n.client.SendRequest(client.GET, "compute", url, &requestData,
		"failed to get details for serverId=%s", serverId)
	return resp.Server, err
}

func (n *OpenStackNovaClient) DeleteServer(serverId string) (err error) {

	var resp struct {
		Server ServerDetail
	}
	url := fmt.Sprintf("%s/%s", OS_API_SERVERS, serverId)
	requestData := goosehttp.RequestData{RespValue: &resp, ExpectedStatus: []int{http.StatusNoContent}}
	err = n.client.SendRequest(client.DELETE, "compute", url, &requestData,
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

func (n *OpenStackNovaClient) RunServer(opts RunServerOpts) (err error) {

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
	err = n.client.SendRequest(client.POST, "compute", OS_API_SERVERS, &requestData,
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

func (n *OpenStackNovaClient) ListSecurityGroups() (groups []SecurityGroup, err error) {

	var resp struct {
		Groups []SecurityGroup `json:"security_groups"`
	}
	requestData := goosehttp.RequestData{RespValue: &resp}
	err = n.client.SendRequest(client.GET, "compute", OS_API_SECURITY_GROUPS, &requestData,
		"failed to list security groups")
	return resp.Groups, err
}

func (n *OpenStackNovaClient) GetServerSecurityGroups(serverId string) (groups []SecurityGroup, err error) {

	var resp struct {
		Groups []SecurityGroup `json:"security_groups"`
	}
	url := fmt.Sprintf("%s/%s/%s", OS_API_SERVERS, serverId, OS_API_SECURITY_GROUPS)
	requestData := goosehttp.RequestData{RespValue: &resp}
	err = n.client.SendRequest(client.GET, "compute", url, &requestData,
		"failed to list server (%s) security groups", serverId)
	return resp.Groups, err
}

func (n *OpenStackNovaClient) CreateSecurityGroup(name, description string) (group SecurityGroup, err error) {

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
	err = n.client.SendRequest(client.POST, "compute", OS_API_SECURITY_GROUPS, &requestData,
		"failed to create a security group with name=%s", name)
	return resp.SecurityGroup, err
}

func (n *OpenStackNovaClient) DeleteSecurityGroup(groupId int) (err error) {

	url := fmt.Sprintf("%s/%d", OS_API_SECURITY_GROUPS, groupId)
	requestData := goosehttp.RequestData{ExpectedStatus: []int{http.StatusAccepted}}
	err = n.client.SendRequest(client.DELETE, "compute", url, &requestData,
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

func (n *OpenStackNovaClient) CreateSecurityGroupRule(ruleInfo RuleInfo) (rule SecurityGroupRule, err error) {

	var req struct {
		SecurityGroupRule RuleInfo `json:"security_group_rule"`
	}
	req.SecurityGroupRule = ruleInfo

	var resp struct {
		SecurityGroupRule SecurityGroupRule `json:"security_group_rule"`
	}

	requestData := goosehttp.RequestData{ReqValue: req, RespValue: &resp}
	err = n.client.SendRequest(client.POST, "compute", OS_API_SECURITY_GROUP_RULES, &requestData,
		"failed to create a rule for the security group with id=%s", ruleInfo.GroupId)
	return resp.SecurityGroupRule, err
}

func (n *OpenStackNovaClient) DeleteSecurityGroupRule(ruleId int) (err error) {

	url := fmt.Sprintf("%s/%d", OS_API_SECURITY_GROUP_RULES, ruleId)
	requestData := goosehttp.RequestData{ExpectedStatus: []int{http.StatusAccepted}}
	err = n.client.SendRequest(client.DELETE, "compute", url, &requestData,
		"failed to delete a security group rule with id=%d", ruleId)
	return
}

func (n *OpenStackNovaClient) AddServerSecurityGroup(serverId, groupName string) (err error) {

	var req struct {
		AddSecurityGroup struct {
			Name string `json:"name"`
		} `json:"addSecurityGroup"`
	}
	req.AddSecurityGroup.Name = groupName

	url := fmt.Sprintf("%s/%s/action", OS_API_SERVERS, serverId)
	requestData := goosehttp.RequestData{ReqValue: req, ExpectedStatus: []int{http.StatusAccepted}}
	err = n.client.SendRequest(client.POST, "compute", url, &requestData,
		"failed to add security group '%s' from server with id=%s", groupName, serverId)
	return
}

func (n *OpenStackNovaClient) RemoveServerSecurityGroup(serverId, groupName string) (err error) {

	var req struct {
		RemoveSecurityGroup struct {
			Name string `json:"name"`
		} `json:"removeSecurityGroup"`
	}
	req.RemoveSecurityGroup.Name = groupName

	url := fmt.Sprintf("%s/%s/action", OS_API_SERVERS, serverId)
	requestData := goosehttp.RequestData{ReqValue: req, ExpectedStatus: []int{http.StatusAccepted}}
	err = n.client.SendRequest(client.POST, "compute", url, &requestData,
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

func (n *OpenStackNovaClient) ListFloatingIPs() (ips []FloatingIP, err error) {

	var resp struct {
		FloatingIPs []FloatingIP `json:"floating_ips"`
	}

	requestData := goosehttp.RequestData{RespValue: &resp}
	err = n.client.SendRequest(client.GET, "compute", OS_API_FLOATING_IPS, &requestData,
		"failed to list floating ips")
	return resp.FloatingIPs, err
}

func (n *OpenStackNovaClient) GetFloatingIP(ipId int) (ip FloatingIP, err error) {

	var resp struct {
		FloatingIP FloatingIP `json:"floating_ip"`
	}

	url := fmt.Sprintf("%s/%d", OS_API_FLOATING_IPS, ipId)
	requestData := goosehttp.RequestData{RespValue: &resp}
	err = n.client.SendRequest(client.GET, "compute", url, &requestData,
		"failed to get floating ip %d details", ipId)
	return resp.FloatingIP, err
}

func (n *OpenStackNovaClient) AllocateFloatingIP() (ip FloatingIP, err error) {

	var resp struct {
		FloatingIP FloatingIP `json:"floating_ip"`
	}

	requestData := goosehttp.RequestData{RespValue: &resp}
	err = n.client.SendRequest(client.POST, "compute", OS_API_FLOATING_IPS, &requestData,
		"failed to allocate a floating ip")
	return resp.FloatingIP, err
}

func (n *OpenStackNovaClient) DeleteFloatingIP(ipId int) (err error) {

	url := fmt.Sprintf("%s/%d", OS_API_FLOATING_IPS, ipId)
	requestData := goosehttp.RequestData{ExpectedStatus: []int{http.StatusAccepted}}
	err = n.client.SendRequest(client.DELETE, "compute", url, &requestData,
		"failed to delete floating ip %d details", ipId)
	return
}

func (n *OpenStackNovaClient) AddServerFloatingIP(serverId, address string) (err error) {

	var req struct {
		AddFloatingIP struct {
			Address string `json:"address"`
		} `json:"addFloatingIp"`
	}
	req.AddFloatingIP.Address = address

	url := fmt.Sprintf("%s/%s/action", OS_API_SERVERS, serverId)
	requestData := goosehttp.RequestData{ReqValue: req, ExpectedStatus: []int{http.StatusAccepted}}
	err = n.client.SendRequest(client.POST, "compute", url, &requestData,
		"failed to add floating ip %s to server %s", address, serverId)
	return
}

func (n *OpenStackNovaClient) RemoveServerFloatingIP(serverId, address string) (err error) {

	var req struct {
		RemoveFloatingIP struct {
			Address string `json:"address"`
		} `json:"removeFloatingIp"`
	}
	req.RemoveFloatingIP.Address = address

	url := fmt.Sprintf("%s/%s/action", OS_API_SERVERS, serverId)
	requestData := goosehttp.RequestData{ReqValue: req, ExpectedStatus: []int{http.StatusAccepted}}
	err = n.client.SendRequest(client.POST, "compute", url, &requestData,
		"failed to remove floating ip %s to server %s", address, serverId)
	return
}
