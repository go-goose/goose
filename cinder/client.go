package cinder

import (
	"fmt"
	"net/http"
	"net/url"
	"time"
)

type TokenFn func() string

func SetEndpointFn(endpoint *url.URL, wrappedHandler RequestHandlerFn) RequestHandlerFn {
	return func(req *http.Request) (*http.Response, error) {
		req.URL.Host = endpoint.Host
		req.Host = endpoint.Host
		return wrappedHandler(req)
	}
}

func SetAuthHeaderFn(token TokenFn, wrappedHandler RequestHandlerFn) RequestHandlerFn {
	return func(req *http.Request) (*http.Response, error) {
		req.Header.Set("X-Auth-Token", token())
		return wrappedHandler(req)
	}
}

func NewClient(tenantId string, handleRequest RequestHandlerFn) *Client {
	return &Client{tenantId, handleRequest}
}

type Client struct {
	tenantId      string
	handleRequest RequestHandlerFn
}

// Shows information for a specified snapshot.
func (c *Client) GetSnapshot(snapshotId string) (*GetSnapshotResults, error) {
	return getSnapshot(
		c.handleRequest,
		GetSnapshotParams{TenantId: c.tenantId, SnapshotId: snapshotId},
	)
}

// Updates a specified snapshot.
func (c *Client) UpdateSnapshot(snapshotId string, args UpdateSnapshotSnapshotParams) (*UpdateSnapshotResults, error) {
	return updateSnapshot(c.handleRequest, UpdateSnapshotParams{
		TenantId:   c.tenantId,
		SnapshotId: snapshotId,
		Snapshot:   args,
	})
}

// Deletes a specified snapshot.
func (c *Client) DeleteSnapshot(snapshotId string) error {
	_, err := deleteSnapshot(
		c.handleRequest,
		DeleteSnapshotParams{TenantId: c.tenantId, SnapshotId: snapshotId},
	)
	return err
}

// Shows details for Block Storage API v2.
func (c *Client) VersionDetails() (*VersionDetailsResults, error) {
	return versionDetails(c.handleRequest, VersionDetailsParams{})
}

// Lists Block Storage API extensions.
func (c *Client) ListExtensionsCinderV2() (*ListExtensionsCinderV2Results, error) {
	return listExtensionsCinderV2(c.handleRequest, ListExtensionsCinderV2Params{})
}

// Lists summary information for all Block Storage volumes that the
// tenant who submits the request can access.
func (c *Client) GetVolumesSimple() (*GetVolumesSimpleResults, error) {
	return getVolumesSimple(c.handleRequest, GetVolumesSimpleParams{TenantId: c.tenantId})
}

// Updates a volume type.
func (c *Client) UpdateVolumeType(volumeTypeId, volumeType string) (*UpdateVolumeTypeResults, error) {
	return updateVolumeType(c.handleRequest, UpdateVolumeTypeParams{
		TenantId:     c.tenantId,
		VolumeTypeId: volumeTypeId,
		VolumeType:   volumeType,
	})
}

// Deletes a specified volume type.
func (c *Client) DeleteVolumeType(volumeTypeId string) error {
	_, err := deleteVolumeType(
		c.handleRequest,
		DeleteVolumeTypeParams{TenantId: c.tenantId, VolumeTypeId: volumeTypeId},
	)
	return err
}

// Lists detailed information for all Block Storage volumes that the tenant who submits the request can access.
func (c *Client) GetVolumesDetail() (*GetVolumesDetailResults, error) {
	return getVolumesDetail(c.handleRequest, GetVolumesDetailParams{TenantId: c.tenantId})
}

// The specified volume must exist. :
func (c *Client) GetVolume(volumeId string) (*GetVolumeResults, error) {
	return getVolume(c.handleRequest, GetVolumeParams{TenantId: c.tenantId, VolumeId: volumeId})
}

// Creates a volume type.
func (c *Client) CreateVolumeType(args CreateVolumeTypeVolumeTypeParams) (*CreateVolumeTypeResults, error) {
	return createVolumeType(
		c.handleRequest,
		CreateVolumeTypeParams{TenantId: c.tenantId, VolumeType: args},
	)
}

// Shows information about a specified volume type.
func (c *Client) GetVolumeType(volumeTypeId string) (*GetVolumeTypeResults, error) {
	return getVolumeType(
		c.handleRequest,
		GetVolumeTypeParams{TenantId: c.tenantId, VolumeTypeId: volumeTypeId},
	)
}

// Lists information about all Block Storage API versions.
func (c *Client) ListVersions() (*ListVersionsResults, error) {
	return listVersions(c.handleRequest, ListVersionsParams{})
}

// Updates the extra specifications assigned to a volume type.
func (c *Client) UpdateVolumeTypeExtraSpecs(volumeTypeId, volumeType, extraSpecs string) (*UpdateVolumeTypeExtraSpecsResults, error) {
	return updateVolumeTypeExtraSpecs(c.handleRequest, UpdateVolumeTypeExtraSpecsParams{
		TenantId:     c.tenantId,
		VolumeTypeId: volumeTypeId,
		VolumeType:   volumeType,
		ExtraSpecs:   extraSpecs,
	})
}

// Lists summary information for all Block Storage snapshots that the
// tenant who submits the request can access.
func (c *Client) GetSnapshotsSimple() (*GetSnapshotsSimpleResults, error) {
	return getSnapshotsSimple(c.handleRequest, GetSnapshotsSimpleParams{TenantId: c.tenantId})
}

// Shows the metadata for a specified snapshot.
func (c *Client) ShowSnapshotMetadata(snapshotId string) (*ShowSnapshotMetadataResults, error) {
	return showSnapshotMetadata(
		c.handleRequest,
		ShowSnapshotMetadataParams{TenantId: c.tenantId, SnapshotId: snapshotId},
	)
}

// Creates a snapshot, which is a point-in-time complete copy of a
// volume. You can create a volume from the snapshot.
func (c *Client) CreateSnapshot(args CreateSnapshotSnapshotParams) (*CreateSnapshotResults, error) {
	return createSnapshot(c.handleRequest, CreateSnapshotParams{TenantId: c.tenantId, Snapshot: args})
}

// Lists detailed information for all Block Storage snapshots that the
// tenant who submits the request can access.
func (c *Client) GetSnapshotsDetail() (*GetSnapshotsDetailResults, error) {
	return getSnapshotsDetail(c.handleRequest, GetSnapshotsDetailParams{TenantId: c.tenantId})
}

// Updates the metadata for a specified snapshot.
func (c *Client) UpdateSnapshotMetadata(snapshotId, key string) (*UpdateSnapshotMetadataResults, error) {
	return updateSnapshotMetadata(c.handleRequest, UpdateSnapshotMetadataParams{
		TenantId:   c.tenantId,
		SnapshotId: snapshotId,
		Metadata: UpdateSnapshotMetadataMetadataParams{
			Key: key,
		},
	})
}

// Creates a volume. To create a bootable volume, include the image
// ID and set the bootable flag to true in the request body.
//
// Preconditions:
//
// - The user must have enough volume storage quota remaining to create
//   a volume of size requested.
//
// Asynchronous Postconditions:
//
// - With correct permissions, you can see the volume status as
//   available through API calls.
// - With correct access, you can see the created volume in the
//   storage system that OpenStack Block Storage manages.
//
// Troubleshooting:
//
// - If volume status remains creating or shows another error status,
//   the request failed. Ensure you meet the preconditions then
//   investigate the storage backend.
// - Volume is not created in the storage system which OpenStack Block Storage manages.
// - The storage node needs enough free storage space to match the
//   specified size of the volume creation request.
func (c *Client) CreateVolume(args CreateVolumeVolumeParams) (*CreateVolumeResults, error) {
	return createVolume(c.handleRequest, CreateVolumeParams{TenantId: c.tenantId, Volume: args})
}

// Updates a volume.
func (c *Client) UpdateVolume(volumeId string, args UpdateVolumeVolumeParams) (*UpdateVolumeResults, error) {
	return updateVolume(c.handleRequest, UpdateVolumeParams{TenantId: c.tenantId, VolumeId: volumeId, Volume: args})
}

// The volume managed by OpenStack Block Storage is not deleted from the storage system. :
func (c *Client) DeleteVolume(volumeId string) error {
	_, err := deleteVolume(
		c.handleRequest,
		DeleteVolumeParams{TenantId: c.tenantId, VolumeId: volumeId},
	)
	return err
}

// Lists volume types.
func (c *Client) GetVolumeTypes() (*GetVolumeTypesResults, error) {
	return getVolumeTypes(c.handleRequest, GetVolumeTypesParams{TenantId: c.tenantId})
}

type StatusResultFn func() (string, error)

// VolumeStatusNotifier will check a volume's status to determine
// whether it matches the given status. After a check, it waits for
// "waitDur" before attempting again. If the status does not match
// after "numAttempts", an error is returned.
func (c *Client) StatusNotifier(
	getStatus StatusResultFn,
	desiredStatus string,
	numAttempts int,
	waitDur time.Duration,
) <-chan error {
	notifierChan := make(chan error)
	go func() {
		for attemptNum := 0; attemptNum < numAttempts; attemptNum++ {
			if retrievedStatus, err := getStatus(); err != nil {
				notifierChan <- err
				return
			} else if retrievedStatus == desiredStatus {
				notifierChan <- nil
				return
			}

			time.Sleep(waitDur)
		}
		notifierChan <- fmt.Errorf("too many attempts")
	}()
	return notifierChan
}

func (c *Client) VolumeStatusNotifier(volId, status string, numAttempts int, waitDur time.Duration) <-chan error {
	getStatus := func() (string, error) {
		volInfo, err := c.GetVolume(volId)
		return volInfo.Volume.Status, err
	}
	return c.StatusNotifier(getStatus, status, numAttempts, waitDur)
}

func (c *Client) SnapshotStatusNotifier(snapId, status string, numAttempts int, waitDur time.Duration) <-chan error {
	getStatus := func() (string, error) {
		snapInfo, err := c.GetSnapshot(snapId)
		return snapInfo.Snapshot.Status, err
	}
	return c.StatusNotifier(getStatus, status, numAttempts, waitDur)
}
