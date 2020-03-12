// goose/nova - Go package to interact with OpenStack Compute (Nova) API.
// See http://docs.openstack.org/api/openstack-compute/2/content/.

package nova

import (
	"fmt"
	"net/http"
	"net/url"
	"reflect"

	"gopkg.in/goose.v2/client"
	"gopkg.in/goose.v2/errors"
	goosehttp "gopkg.in/goose.v2/http"
)

// API URL parts.
const (
	apiFlavors           = "flavors"
	apiFlavorsDetail     = "flavors/detail"
	apiServers           = "servers"
	apiServersDetail     = "servers/detail"
	apiAvailabilityZone  = "os-availability-zone"
	apiVolumeAttachments = "os-volume_attachments"
	apiOSInterface       = "os-interface"
)

// Server status values.
const (
	StatusActive        = "ACTIVE"          // The server is active.
	StatusBuild         = "BUILD"           // The server has not finished the original build process.
	StatusBuildSpawning = "BUILD(spawning)" // The server has not finished the original build process but networking works (HP Cloud specific)
	StatusDeleted       = "DELETED"         // The server is deleted.
	StatusError         = "ERROR"           // The server is in error.
	StatusHardReboot    = "HARD_REBOOT"     // The server is hard rebooting.
	StatusPassword      = "PASSWORD"        // The password is being reset on the server.
	StatusReboot        = "REBOOT"          // The server is in a soft reboot state.
	StatusRebuild       = "REBUILD"         // The server is currently being rebuilt from an image.
	StatusRescue        = "RESCUE"          // The server is in rescue mode.
	StatusResize        = "RESIZE"          // Server is performing the differential copy of data that changed during its initial copy.
	StatusShutoff       = "SHUTOFF"         // The virtual machine (VM) was powered down by the user, but not through the OpenStack Compute API.
	StatusSuspended     = "SUSPENDED"       // The server is suspended, either by request or necessity.
	StatusUnknown       = "UNKNOWN"         // The state of the server is unknown. Contact your cloud provider.
	StatusVerifyResize  = "VERIFY_RESIZE"   // System is awaiting confirmation that the server is operational after a move or resize.
)

// Filter keys.
const (
	FilterStatus       = "status"        // The server status. See Server Status Values.
	FilterImage        = "image"         // The image reference specified as an ID or full URL.
	FilterFlavor       = "flavor"        // The flavor reference specified as an ID or full URL.
	FilterServer       = "name"          // The server name.
	FilterMarker       = "marker"        // The ID of the last item in the previous list.
	FilterLimit        = "limit"         // The page size.
	FilterChangesSince = "changes-since" // The changes-since time. The list contains servers that have been deleted since the changes-since time.
)

// Client provides a means to access the OpenStack Compute Service.
type Client struct {
	client client.Client
}

// New creates a new Client.
func New(client client.Client) *Client {
	return &Client{client}
}

// ----------------------------------------------------------------------------
// Filtering helper.
//
// Filter builds filtering parameters to be used in an OpenStack query which supports
// filtering.  For example:
//
//     filter := NewFilter()
//     filter.Set(nova.FilterServer, "server_name")
//     filter.Set(nova.FilterStatus, nova.StatusBuild)
//     resp, err := nova.ListServers(filter)
//
type Filter struct {
	v url.Values
}

// NewFilter creates a new Filter.
func NewFilter() *Filter {
	return &Filter{make(url.Values)}
}

func (f *Filter) Set(filter, value string) {
	f.v.Set(filter, value)
}

// Link describes a link to a flavor or server.
type Link struct {
	Href string
	Rel  string
	Type string
}

// Entity describe a basic information about a flavor or server.
type Entity struct {
	Id    string `json:"-"`
	UUID  string `json:"uuid"`
	Links []Link `json:"links"`
	Name  string `json:"name"`
}

func stringValue(item interface{}, attr string) string {
	return reflect.ValueOf(item).FieldByName(attr).String()
}

// Allow Entity slices to be sorted by named attribute.
type EntitySortBy struct {
	Attr     string
	Entities []Entity
}

func (e EntitySortBy) Len() int {
	return len(e.Entities)
}

func (e EntitySortBy) Less(i, j int) bool {
	return stringValue(e.Entities[i], e.Attr) < stringValue(e.Entities[j], e.Attr)
}

func (e EntitySortBy) Swap(i, j int) {
	e.Entities[i], e.Entities[j] = e.Entities[j], e.Entities[i]
}

// ListFlavours lists IDs, names, and links for available flavors.
func (c *Client) ListFlavors() ([]Entity, error) {
	var resp struct {
		Flavors []Entity
	}
	requestData := goosehttp.RequestData{RespValue: &resp}
	err := c.client.SendRequest(client.GET, "compute", "v2", apiFlavors, &requestData)
	if err != nil {
		return nil, errors.Newf(err, "failed to get list of flavours")
	}
	return resp.Flavors, nil
}

// FlavorDetail describes detailed information about a flavor.
type FlavorDetail struct {
	Name  string
	RAM   int    // Available RAM, in MB
	VCPUs int    // Number of virtual CPU (cores)
	Disk  int    // Available root partition space, in GB
	Id    string `json:"-"`
	Links []Link
}

// Allow FlavorDetail slices to be sorted by named attribute.
type FlavorDetailSortBy struct {
	Attr          string
	FlavorDetails []FlavorDetail
}

func (e FlavorDetailSortBy) Len() int {
	return len(e.FlavorDetails)
}

func (e FlavorDetailSortBy) Less(i, j int) bool {
	return stringValue(e.FlavorDetails[i], e.Attr) < stringValue(e.FlavorDetails[j], e.Attr)
}

func (e FlavorDetailSortBy) Swap(i, j int) {
	e.FlavorDetails[i], e.FlavorDetails[j] = e.FlavorDetails[j], e.FlavorDetails[i]
}

// ListFlavorsDetail lists all details for available flavors.
func (c *Client) ListFlavorsDetail() ([]FlavorDetail, error) {
	var resp struct {
		Flavors []FlavorDetail
	}
	requestData := goosehttp.RequestData{RespValue: &resp}
	err := c.client.SendRequest(client.GET, "compute", "v2", apiFlavorsDetail, &requestData)
	if err != nil {
		return nil, errors.Newf(err, "failed to get list of flavour details")
	}
	return resp.Flavors, nil
}

// ListServers lists IDs, names, and links for all servers.
func (c *Client) ListServers(filter *Filter) ([]Entity, error) {
	var resp struct {
		Servers []Entity
	}
	var params *url.Values
	if filter != nil {
		params = &filter.v
	}
	requestData := goosehttp.RequestData{RespValue: &resp, Params: params, ExpectedStatus: []int{http.StatusOK}}
	err := c.client.SendRequest(client.GET, "compute", "v2", apiServers, &requestData)
	if err != nil {
		return nil, errors.Newf(err, "failed to get list of servers")
	}
	return resp.Servers, nil
}

// IPAddress describes a single IPv4/6 address of a server.
type IPAddress struct {
	Version int    `json:"version"`
	Address string `json:"addr"`
	Type    string `json:"OS-EXT-IPS:type"` // fixed or floating
}

// ServerFault describes a single server fault. Details (stack trace) are available for
// those with adminstrator privilages.
type ServerFault struct {
	Code    int    `json:"code"` // Response code
	Created string `json:"created"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// ServerDetail describes a server in more detail.
// See: http://docs.openstack.org/api/openstack-compute/2/content/Extensions-d1e1444.html#ServersCBSJ
type ServerDetail struct {
	// AddressIPv4 and AddressIPv6 hold the first public IPv4 or IPv6
	// address of the server, or "" if no floating IP is assigned.
	AddressIPv4 string
	AddressIPv6 string

	// Addresses holds the list of all IP addresses assigned to this
	// server, grouped by "network" name ("public", "private" or a
	// custom name).
	Addresses map[string][]IPAddress

	// Created holds the creation timestamp of the server
	// in RFC3339 format.
	Created string

	Flavor   Entity
	HostId   string `json:"hostId"`
	Id       string `json:"-"`
	UUID     string
	Image    Entity
	Links    []Link
	Name     string
	Metadata map[string]string

	Groups *[]SecurityGroupName `json:"security_groups"`

	// Progress holds the completion percentage of
	// the current operation
	Progress int

	// Status holds the current status of the server,
	// one of the Status* constants.
	Status string

	// Only returned if status is Error
	Fault *ServerFault `json:"fault"`

	TenantId string `json:"tenant_id"`

	// Updated holds the timestamp of the last update
	// to the server in RFC3339 format.
	Updated string

	UserId string `json:"user_id"`

	AvailabilityZone string `json:"OS-EXT-AZ:availability_zone"`
}

// ListServersDetail lists all details for available servers.
func (c *Client) ListServersDetail(filter *Filter) ([]ServerDetail, error) {
	var resp struct {
		Servers []ServerDetail
	}
	var params *url.Values
	if filter != nil {
		params = &filter.v
	}
	requestData := goosehttp.RequestData{RespValue: &resp, Params: params}
	err := c.client.SendRequest(client.GET, "compute", "v2", apiServersDetail, &requestData)
	if err != nil {
		return nil, errors.Newf(err, "failed to get list of server details")
	}
	return resp.Servers, nil
}

// GetServer lists details for the specified server.
func (c *Client) GetServer(serverId string) (*ServerDetail, error) {
	var resp struct {
		Server ServerDetail
	}
	url := fmt.Sprintf("%s/%s", apiServers, serverId)
	requestData := goosehttp.RequestData{RespValue: &resp}
	err := c.client.SendRequest(client.GET, "compute", "v2", url, &requestData)
	if err != nil {
		return nil, errors.Newf(err, "failed to get details for serverId: %s", serverId)
	}
	return &resp.Server, nil
}

// DeleteServer terminates the specified server.
func (c *Client) DeleteServer(serverId string) error {
	var resp struct {
		Server ServerDetail
	}
	url := fmt.Sprintf("%s/%s", apiServers, serverId)
	requestData := goosehttp.RequestData{RespValue: &resp, ExpectedStatus: []int{http.StatusNoContent}}
	err := c.client.SendRequest(client.DELETE, "compute", "v2", url, &requestData)
	if err != nil {
		err = errors.Newf(err, "failed to delete server with serverId: %s", serverId)
	}
	return err
}

type SecurityGroupName struct {
	Name string `json:"name"`
}

// ServerNetworks sets what networks a server should be connected to on boot.
// - FixedIp may be supplied only when NetworkId is also given.
// - PortId may be supplied only if neither NetworkId or FixedIp is set.
type ServerNetworks struct {
	NetworkId string `json:"uuid,omitempty"`
	FixedIp   string `json:"fixed_ip,omitempty"`
	PortId    string `json:"port,omitempty"`
}

// RunServerOpts defines required and optional arguments for RunServer().
type RunServerOpts struct {
	Name                string               `json:"name"`                              // Required
	FlavorId            string               `json:"flavorRef"`                         // Required
	ImageId             string               `json:"imageRef,omitempty"`                // Optional
	UserData            []byte               `json:"user_data,omitempty"`               // Optional
	SecurityGroupNames  []SecurityGroupName  `json:"security_groups,omitempty"`         // Optional
	Networks            []ServerNetworks     `json:"networks"`                          // Optional
	AvailabilityZone    string               `json:"availability_zone,omitempty"`       // Optional
	Metadata            map[string]string    `json:"metadata,omitempty"`                // Optional
	ConfigDrive         bool                 `json:"config_drive,omitempty"`            // Optional
	BlockDeviceMappings []BlockDeviceMapping `json:"block_device_mapping_v2,omitempty"` // Optional
}

// BlockDeviceMapping defines block devices to be attached to the Server created by RunServer().
// See: https://developer.openstack.org/api-ref/compute/?expanded=create-server-detail
type BlockDeviceMapping struct {
	BootIndex           int    `json:"boot_index"`
	UUID                string `json:"uuid,omitempty"`
	SourceType          string `json:"source_type,omitempty"`
	DestinationType     string `json:"destination_type,omitempty"`
	VolumeSize          int    `json:"volume_size,omitempty"`
	VolumeType          string `json:"volume_type,omitempty"`
	DeleteOnTermination bool   `json:"delete_on_termination,omitempty"`
	DeviceName          string `json:"device_name,omitempty"`
	DeviceType          string `json:"device_type,omitempty"`
	DiskBus             string `json:"disk_bus,omitempty"`
	GuestFormat         string `json:"guest_format,omitempty"`
	NoDevice            bool   `json:"no_device,omitempty"`
	Tag                 string `json:"tag,omitempty"`
}

// RunServer creates a new server, based on the given RunServerOpts.
func (c *Client) RunServer(opts RunServerOpts) (*Entity, error) {
	var req struct {
		Server RunServerOpts `json:"server"`
	}
	req.Server = opts
	// opts.UserData gets serialized to base64-encoded string automatically
	var resp struct {
		Server Entity `json:"server"`
	}
	requestData := goosehttp.RequestData{ReqValue: req, RespValue: &resp, ExpectedStatus: []int{http.StatusAccepted}}
	err := c.client.SendRequest(client.POST, "compute", "v2", apiServers, &requestData)
	if err != nil {
		return nil, errors.Newf(err, "failed to run a server with %#v", opts)
	}
	return &resp.Server, nil
}

type serverUpdateNameOpts struct {
	Name string `json:"name"`
}

// UpdateServerName updates the name of the given server.
func (c *Client) UpdateServerName(serverID, name string) (*Entity, error) {
	var req struct {
		Server serverUpdateNameOpts `json:"server"`
	}
	var resp struct {
		Server Entity `json:"server"`
	}
	req.Server = serverUpdateNameOpts{Name: name}
	requestData := goosehttp.RequestData{ReqValue: req, RespValue: &resp, ExpectedStatus: []int{http.StatusOK}}
	url := fmt.Sprintf("%s/%s", apiServers, serverID)
	err := c.client.SendRequest(client.PUT, "compute", "v2", url, &requestData)
	if err != nil {
		return nil, errors.Newf(err, "failed to update server name to %q", name)
	}
	return &resp.Server, nil
}

// AddServerSecurityGroup adds a security group to the specified server.
func (c *Client) AddServerSecurityGroup(serverId, groupName string) error {
	var req struct {
		AddSecurityGroup struct {
			Name string `json:"name"`
		} `json:"addSecurityGroup"`
	}
	req.AddSecurityGroup.Name = groupName

	url := fmt.Sprintf("%s/%s/action", apiServers, serverId)
	requestData := goosehttp.RequestData{ReqValue: req, ExpectedStatus: []int{http.StatusAccepted}}
	err := c.client.SendRequest(client.POST, "compute", "v2", url, &requestData)
	if err != nil {
		err = errors.Newf(err, "failed to add security group '%s' to server with id: %s", groupName, serverId)
	}
	return err
}

// RemoveServerSecurityGroup removes a security group from the specified server.
func (c *Client) RemoveServerSecurityGroup(serverId, groupName string) error {
	var req struct {
		RemoveSecurityGroup struct {
			Name string `json:"name"`
		} `json:"removeSecurityGroup"`
	}
	req.RemoveSecurityGroup.Name = groupName

	url := fmt.Sprintf("%s/%s/action", apiServers, serverId)
	requestData := goosehttp.RequestData{ReqValue: req, ExpectedStatus: []int{http.StatusAccepted}}
	err := c.client.SendRequest(client.POST, "compute", "v2", url, &requestData)
	if err != nil {
		err = errors.Newf(err, "failed to remove security group '%s' from server with id: %s", groupName, serverId)
	}
	return err
}

// AddServerFloatingIP assigns a floating IP address to the specified server.
func (c *Client) AddServerFloatingIP(serverId, address string) error {
	var req struct {
		AddFloatingIP struct {
			Address string `json:"address"`
		} `json:"addFloatingIp"`
	}
	req.AddFloatingIP.Address = address

	url := fmt.Sprintf("%s/%s/action", apiServers, serverId)
	requestData := goosehttp.RequestData{ReqValue: req, ExpectedStatus: []int{http.StatusAccepted}}
	err := c.client.SendRequest(client.POST, "compute", "v2", url, &requestData)
	if err != nil {
		err = errors.Newf(err, "failed to add floating ip %s to server with id: %s", address, serverId)
	}
	return err
}

// RemoveServerFloatingIP removes a floating IP address from the specified server.
func (c *Client) RemoveServerFloatingIP(serverId, address string) error {
	var req struct {
		RemoveFloatingIP struct {
			Address string `json:"address"`
		} `json:"removeFloatingIp"`
	}
	req.RemoveFloatingIP.Address = address

	url := fmt.Sprintf("%s/%s/action", apiServers, serverId)
	requestData := goosehttp.RequestData{ReqValue: req, ExpectedStatus: []int{http.StatusAccepted}}
	err := c.client.SendRequest(client.POST, "compute", "v2", url, &requestData)
	if err != nil {
		err = errors.Newf(err, "failed to remove floating ip %s from server with id: %s", address, serverId)
	}
	return err
}

// AvailabilityZone identifies an availability zone, and describes its state.
type AvailabilityZone struct {
	Name  string                `json:"zoneName"`
	State AvailabilityZoneState `json:"zoneState"`
}

// AvailabilityZoneState describes an availability zone's state.
type AvailabilityZoneState struct {
	Available bool
}

// ListAvailabilityZones lists all availability zones.
//
// Availability zones are an OpenStack extension; if the server does not
// support them, then an error satisfying errors.IsNotImplemented will be
// returned.
func (c *Client) ListAvailabilityZones() ([]AvailabilityZone, error) {
	var resp struct {
		AvailabilityZoneInfo []AvailabilityZone
	}
	requestData := goosehttp.RequestData{RespValue: &resp}
	err := c.client.SendRequest(client.GET, "compute", "v2", apiAvailabilityZone, &requestData)
	if errors.IsNotFound(err) {
		// Availability zones are an extension, so don't
		// return an error if the API does not exist.
		return nil, errors.NewNotImplementedf(
			err, nil, "the server does not support availability zones",
		)
	}
	if err != nil {
		return nil, errors.Newf(err, "failed to get list of availability zones")
	}
	return resp.AvailabilityZoneInfo, nil
}

// VolumeAttachment represents both the request and response for
// attaching volumes.
type VolumeAttachment struct {
	Device   *string `json:"device,omitempty"`
	Id       string  `json:"id,omitempty"`
	ServerId string  `json:"serverId,omitempty"`
	VolumeId string  `json:"volumeId"`
}

// AttachVolume attaches the given volumeId to the given serverId at
// mount point specified in device. Note that the server must support
// the os-volume_attachments attachment; if it does not, an error will
// be returned stating such.
func (c *Client) AttachVolume(serverId, volumeId, device string) (*VolumeAttachment, error) {

	type volumeAttachment struct {
		VolumeAttachment VolumeAttachment `json:"volumeAttachment"`
	}

	var devicePtr *string
	if device != "" {
		devicePtr = &device
	}

	var resp volumeAttachment
	requestData := goosehttp.RequestData{
		ReqValue: &volumeAttachment{VolumeAttachment{
			VolumeId: volumeId,
			Device:   devicePtr,
		}},
		RespValue: &resp,
	}
	url := fmt.Sprintf("%s/%s/%s", apiServers, serverId, apiVolumeAttachments)
	err := c.client.SendRequest(client.POST, "compute", "v2", url, &requestData)
	if err != nil {
		return nil, errors.Newf(err, "failed to attach volume")
	}
	return &resp.VolumeAttachment, nil
}

// DetachVolume detaches the volume with the given attachmentId from
// the server with the given serverId.
func (c *Client) DetachVolume(serverId, attachmentId string) error {
	requestData := goosehttp.RequestData{
		ExpectedStatus: []int{http.StatusAccepted},
	}
	url := fmt.Sprintf("%s/%s/%s/%s", apiServers, serverId, apiVolumeAttachments, attachmentId)
	err := c.client.SendRequest(client.DELETE, "compute", "v2", url, &requestData)
	if err != nil {
		return errors.Newf(err, "failed to delete volume attachment")
	}
	return nil
}

// ListVolumeAttachments lists the volumes currently attached to the
// server with the given serverId.
func (c *Client) ListVolumeAttachments(serverId string) ([]VolumeAttachment, error) {
	var resp struct {
		VolumeAttachments []VolumeAttachment `json:"volumeAttachments"`
	}
	requestData := goosehttp.RequestData{
		RespValue: &resp,
	}
	url := fmt.Sprintf("%s/%s/%s", apiServers, serverId, apiVolumeAttachments)
	err := c.client.SendRequest(client.GET, "compute", "v2", url, &requestData)
	if err != nil {
		return nil, errors.Newf(err, "failed to list volume attachments")
	}
	return resp.VolumeAttachments, nil
}

// SetServerMetadata sets metadata on a server. Replaces metadata
// items that match keys - doesn't modify items that aren't in the
// request.
// See https://developer.openstack.org/api-ref/compute/?expanded=update-metadata-items-detail#update-metadata-items
func (c *Client) SetServerMetadata(serverId string, metadata map[string]string) error {
	req := struct {
		Metadata map[string]string `json:"metadata"`
	}{metadata}

	url := fmt.Sprintf("%s/%s/metadata", apiServers, serverId)
	requestData := goosehttp.RequestData{
		ReqValue: req, ExpectedStatus: []int{http.StatusOK},
	}
	err := c.client.SendRequest(client.POST, "compute", "v2", url, &requestData)
	if err != nil {
		err = errors.Newf(err, "failed to set metadata %v on server with id: %s", metadata, serverId)
	}
	return err
}

// PortFixedIP represents a FixedIP with ip addresses and an associated
// subnet id.
type PortFixedIP struct {
	IPAddress string `json:"ip_address"`
	SubnetID  string `json:"subnet_id"`
}

// OSInterface represents an interface attachment to a server.
type OSInterface struct {
	FixedIPs   []PortFixedIP `json:"fixed_ips,omitempty"`
	IPAddress  string        `json:"ip_address"`
	MacAddress string        `json:"mac_addr,omitempty"`
	NetID      string        `json:"net_id,omitempty"`
	PortID     string        `json:"port_id,omitempty"`
	PortState  string        `json:"port_state,omitempty"`
}

// ListOSInterfaces lists all the os-interfaces (port interfaces) associated
// with a given server.
// https://docs.openstack.org/api-ref/compute/?expanded=list-port-interfaces-detail
func (c *Client) ListOSInterfaces(serverId string) ([]OSInterface, error) {
	var resp struct {
		InterfaceAttachments []OSInterface `json:"interfaceAttachments"`
	}
	requestData := goosehttp.RequestData{
		RespValue: &resp,
	}
	url := fmt.Sprintf("%s/%s/%s", apiServers, serverId, apiOSInterface)
	err := c.client.SendRequest(client.GET, "compute", "v2", url, &requestData)
	if err != nil {
		return nil, errors.Newf(err, "failed to list os interfaces")
	}
	return resp.InterfaceAttachments, nil
}
