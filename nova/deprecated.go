package nova

import (
	"fmt"
	"net/http"

	"gopkg.in/goose.v2/client"
	"gopkg.in/goose.v2/errors"
	goosehttp "gopkg.in/goose.v2/http"
)

// The following API requests found in this file are officially deprecated by
// the upstream openstack project.
// The API requests will be left as is, but marked as deprecated and will be
// removed in the v3 release of goose. Migrating API calls to the new API
// requests is recommended.

const (
	// Deprecated.
	// https://docs.openstack.org/api-ref/compute/?expanded=list-security-groups-detail#list-security-groups
	apiSecurityGroups = "os-security-groups"

	// Deprecated.
	// https://docs.openstack.org/api-ref/compute/?expanded=list-security-groups-detail#create-security-group-rule
	apiSecurityGroupRules = "os-security-group-rules"

	// Deprecated.
	// https://docs.openstack.org/api-ref/compute/?expanded=list-security-groups-detail#show-fixed-ip-details
	apiFloatingIPs = "os-floating-ips"
)

// SecurityGroupRef refers to an existing named security group
type SecurityGroupRef struct {
	TenantId string `json:"tenant_id"`
	Name     string `json:"name"`
}

// SecurityGroupRule describes a rule of a security group. There are 2
// basic rule types: ingress and group rules (see RuleInfo struct).
type SecurityGroupRule struct {
	FromPort      *int              `json:"from_port"`   // Can be nil
	IPProtocol    *string           `json:"ip_protocol"` // Can be nil
	ToPort        *int              `json:"to_port"`     // Can be nil
	ParentGroupId string            `json:"-"`
	IPRange       map[string]string `json:"ip_range"` // Can be empty
	Id            string            `json:"-"`
	Group         SecurityGroupRef
}

// SecurityGroup describes a single security group in OpenStack.
type SecurityGroup struct {
	Rules       []SecurityGroupRule
	TenantId    string `json:"tenant_id"`
	Id          string `json:"-"`
	Name        string
	Description string
}

// ListSecurityGroups lists IDs, names, and other details for all security groups.
func (c *Client) ListSecurityGroups() ([]SecurityGroup, error) {
	var resp struct {
		Groups []SecurityGroup `json:"security_groups"`
	}
	requestData := goosehttp.RequestData{RespValue: &resp}
	err := c.client.SendRequest(client.GET, "compute", "v2", apiSecurityGroups, &requestData)
	if err != nil {
		return nil, errors.Newf(err, "failed to list security groups")
	}
	return resp.Groups, nil
}

// SecurityGroupByName returns the named security group.
// Note: due to lack of filtering support when querying security groups, this is not an efficient implementation
// but it's all we can do for now.
func (c *Client) SecurityGroupByName(name string) (*SecurityGroup, error) {
	// OpenStack does not support group filtering, so we need to load them all and manually search by name.
	groups, err := c.ListSecurityGroups()
	if err != nil {
		return nil, err
	}
	for _, group := range groups {
		if group.Name == name {
			return &group, nil
		}
	}
	return nil, errors.NewNotFoundf(nil, "", "Security group %s not found.", name)
}

// GetServerSecurityGroups list security groups for a specific server.
func (c *Client) GetServerSecurityGroups(serverId string) ([]SecurityGroup, error) {

	var resp struct {
		Groups []SecurityGroup `json:"security_groups"`
	}
	url := fmt.Sprintf("%s/%s/%s", apiServers, serverId, apiSecurityGroups)
	requestData := goosehttp.RequestData{RespValue: &resp}
	err := c.client.SendRequest(client.GET, "compute", "v2", url, &requestData)
	if err != nil {
		// Sadly HP Cloud lacks the necessary API and also doesn't provide full SecurityGroup lookup.
		// The best we can do for now is to use just the Name from the group entities.
		if errors.IsNotFound(err) {
			serverDetails, err := c.GetServer(serverId)
			if err == nil && serverDetails.Groups != nil {
				result := make([]SecurityGroup, len(*serverDetails.Groups))
				for i, e := range *serverDetails.Groups {
					result[i] = SecurityGroup{Name: e.Name}
				}
				return result, nil
			}
		}
		return nil, errors.Newf(err, "failed to list server (%s) security groups", serverId)
	}
	return resp.Groups, nil
}

// CreateSecurityGroup creates a new security group.
func (c *Client) CreateSecurityGroup(name, description string) (*SecurityGroup, error) {
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
	err := c.client.SendRequest(client.POST, "compute", "v2", apiSecurityGroups, &requestData)
	if err != nil {
		return nil, errors.Newf(err, "failed to create a security group with name: %s", name)
	}
	return &resp.SecurityGroup, nil
}

// DeleteSecurityGroup deletes the specified security group.
func (c *Client) DeleteSecurityGroup(groupId string) error {
	url := fmt.Sprintf("%s/%s", apiSecurityGroups, groupId)
	requestData := goosehttp.RequestData{ExpectedStatus: []int{http.StatusAccepted}}
	err := c.client.SendRequest(client.DELETE, "compute", "v2", url, &requestData)
	if err != nil {
		err = errors.Newf(err, "failed to delete security group with id: %s", groupId)
	}
	return err
}

// UpdateSecurityGroup updates the name and description of the given group.
func (c *Client) UpdateSecurityGroup(groupId, name, description string) (*SecurityGroup, error) {
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
	url := fmt.Sprintf("%s/%s", apiSecurityGroups, groupId)
	requestData := goosehttp.RequestData{ReqValue: req, RespValue: &resp, ExpectedStatus: []int{http.StatusOK}}
	err := c.client.SendRequest(client.PUT, "compute", "v2", url, &requestData)
	if err != nil {
		return nil, errors.Newf(err, "failed to update security group with Id %s to name: %s", groupId, name)
	}
	return &resp.SecurityGroup, nil
}

// RuleInfo allows the callers of CreateSecurityGroupRule() to
// create 2 types of security group rules: ingress rules and group
// rules. The difference stems from how the "source" is defined.
// It can be either:
// 1. Ingress rules - specified directly with any valid subnet mask
//    in CIDR format (e.g. "192.168.0.0/16");
// 2. Group rules - specified indirectly by giving a source group,
// which can be any user's group (different tenant ID).
//
// Every rule works as an iptables ACCEPT rule, thus a group/ with no
// rules does not allow ingress at all. Rules can be added and removed
// while the server(s) are running. The set of security groups that
// apply to a server is changed only when the server is
// started. Adding or removing a security group on a running server
// will not take effect until that server is restarted. However,
// changing rules of existing groups will take effect immediately.
//
// For more information:
// http://docs.openstack.org/developer/nova/nova.concepts.html#concept-security-groups
// Nova source: https://github.com/openstack/nova.git
type RuleInfo struct {
	/// IPProtocol is optional, and if specified must be "tcp", "udp" or
	//  "icmp" (in this case, both FromPort and ToPort can be -1).
	IPProtocol string `json:"ip_protocol"`

	// FromPort and ToPort are both optional, and if specifed must be
	// integers between 1 and 65535 (valid TCP port numbers). -1 is a
	// special value, meaning "use default" (e.g. for ICMP).
	FromPort int `json:"from_port"`
	ToPort   int `json:"to_port"`

	// Cidr cannot be specified with GroupId. Ingress rules need a valid
	// subnet mast in CIDR format here, while if GroupID is specifed, it
	// means you're adding a group rule, specifying source group ID, which
	// must exist already and can be equal to ParentGroupId).
	// need Cidr, while
	Cidr    string  `json:"cidr"`
	GroupId *string `json:"-"`

	// ParentGroupId is always required and specifies the group to which
	// the rule is added.
	ParentGroupId string `json:"-"`
}

// CreateSecurityGroupRule creates a security group rule.
// It can either be an ingress rule or group rule (see the
// description of RuleInfo).
func (c *Client) CreateSecurityGroupRule(ruleInfo RuleInfo) (*SecurityGroupRule, error) {
	var req struct {
		SecurityGroupRule RuleInfo `json:"security_group_rule"`
	}
	req.SecurityGroupRule = ruleInfo

	var resp struct {
		SecurityGroupRule SecurityGroupRule `json:"security_group_rule"`
	}

	requestData := goosehttp.RequestData{ReqValue: req, RespValue: &resp}
	err := c.client.SendRequest(client.POST, "compute", "v2", apiSecurityGroupRules, &requestData)
	if err != nil {
		return nil, errors.Newf(err, "failed to create a rule for the security group with id: %v", ruleInfo.GroupId)
	}
	return &resp.SecurityGroupRule, nil
}

// DeleteSecurityGroupRule deletes the specified security group rule.
func (c *Client) DeleteSecurityGroupRule(ruleId string) error {
	url := fmt.Sprintf("%s/%s", apiSecurityGroupRules, ruleId)
	requestData := goosehttp.RequestData{ExpectedStatus: []int{http.StatusAccepted}}
	err := c.client.SendRequest(client.DELETE, "compute", "v2", url, &requestData)
	if err != nil {
		err = errors.Newf(err, "failed to delete security group rule with id: %s", ruleId)
	}
	return err
}

// FloatingIP describes a floating (public) IP address, which can be
// assigned to a server, thus allowing connections from outside.
type FloatingIP struct {
	// FixedIP holds the private IP address of the machine (when assigned)
	FixedIP *string `json:"fixed_ip"`
	Id      string  `json:"-"`
	// InstanceId holds the instance id of the machine, if this FIP is assigned to one
	InstanceId *string `json:"-"`
	IP         string  `json:"ip"`
	Pool       string  `json:"pool"`
}

// ListFloatingIPs lists floating IP addresses associated with the tenant or account.
func (c *Client) ListFloatingIPs() ([]FloatingIP, error) {
	var resp struct {
		FloatingIPs []FloatingIP `json:"floating_ips"`
	}

	requestData := goosehttp.RequestData{RespValue: &resp}
	err := c.client.SendRequest(client.GET, "compute", "v2", apiFloatingIPs, &requestData)
	if err != nil {
		return nil, errors.Newf(err, "failed to list floating ips")
	}
	return resp.FloatingIPs, nil
}

// GetFloatingIP lists details of the floating IP address associated with specified id.
func (c *Client) GetFloatingIP(ipId string) (*FloatingIP, error) {
	var resp struct {
		FloatingIP FloatingIP `json:"floating_ip"`
	}

	url := fmt.Sprintf("%s/%s", apiFloatingIPs, ipId)
	requestData := goosehttp.RequestData{RespValue: &resp}
	err := c.client.SendRequest(client.GET, "compute", "v2", url, &requestData)
	if err != nil {
		return nil, errors.Newf(err, "failed to get floating ip %s details", ipId)
	}
	return &resp.FloatingIP, nil
}

// AllocateFloatingIP allocates a new floating IP address to a tenant or account.
func (c *Client) AllocateFloatingIP() (*FloatingIP, error) {
	var resp struct {
		FloatingIP FloatingIP `json:"floating_ip"`
	}

	requestData := goosehttp.RequestData{RespValue: &resp}
	err := c.client.SendRequest(client.POST, "compute", "v2", apiFloatingIPs, &requestData)
	if err != nil {
		return nil, errors.Newf(err, "failed to allocate a floating ip")
	}
	return &resp.FloatingIP, nil
}

// DeleteFloatingIP deallocates the floating IP address associated with the specified id.
func (c *Client) DeleteFloatingIP(ipId string) error {
	url := fmt.Sprintf("%s/%s", apiFloatingIPs, ipId)
	requestData := goosehttp.RequestData{ExpectedStatus: []int{http.StatusAccepted}}
	err := c.client.SendRequest(client.DELETE, "compute", "v2", url, &requestData)
	if err != nil {
		err = errors.Newf(err, "failed to delete floating ip %s details", ipId)
	}
	return err
}
