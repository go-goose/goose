// Copyright 2015 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package cinder

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
)

type RequestHandlerFn func(*http.Request) (*http.Response, error)

type UpdateVolumeTypeParams struct {

	// VolumeType is required.
	//
	// A volume type offers a way to categorize or group volumes.
	VolumeType string `json:"volume_type"`

	//
	// The unique identifier of the tenant or account.
	TenantId string `json:"-"`

	// VolumeTypeId is required.
	//
	// The unique identifier for an existing volume type.
	VolumeTypeId string `json:"-"`
}

type UpdateVolumeTypeResults struct {
	VolumeType struct {
		ExtraSpecs struct {
			Capabilities string `json:"capabilities"`
		} `json:"extra_specs"`
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"volume_type"`
}

//
// Updates a volume type.
func updateVolumeType(request RequestHandlerFn, args UpdateVolumeTypeParams) (*UpdateVolumeTypeResults, error) {

	argsAsJson, err := json.Marshal(args)
	if err != nil {
		return nil, err
	}

	url := "https://volume.example.com/v2/%7Btenant_id%7D/types/%7Bvolume_type_id%7D"

	url = strings.Replace(url, "%7Btenant_id%7D", args.TenantId, -1)
	url = strings.Replace(url, "%7Bvolume_type_id%7D", args.VolumeTypeId, -1)

	var req *http.Request
	if string(argsAsJson) != "{}" {
		req, err = http.NewRequest("PUT", url, bytes.NewBuffer(argsAsJson))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
	} else {
		req, err = http.NewRequest("PUT", url, nil)
		if err != nil {
			return nil, err
		}
	}

	resp, err := request(req)
	if err != nil {
		return nil, err
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	switch resp.StatusCode {
	default:
		return nil, fmt.Errorf("invalid status (%d): %s", resp.StatusCode, body)
	case 200:
		break
	}

	var results UpdateVolumeTypeResults
	json.Unmarshal(body, &results)

	return &results, nil
}

type UpdateVolumeTypeExtraSpecsParams struct {

	// VolumeType is required.
	//
	// A volume type offers a way to categorize or group volumes.
	VolumeType string `json:"volume_type"`

	// ExtraSpecs is required.
	//
	// A key:value pair that offers a way to show additional specifications associated with the volume type. Examples include capabilities, capacity, compression, and so on, depending on the storage driver in use.
	ExtraSpecs string `json:"extra_specs"`

	//
	// The unique identifier of the tenant or account.
	TenantId string `json:"-"`

	// VolumeTypeId is required.
	//
	// The unique identifier for an existing volume type.
	VolumeTypeId string `json:"-"`
}

type UpdateVolumeTypeExtraSpecsResults struct {
	VolumeType struct {
		ExtraSpecs struct {
			Capabilities string `json:"capabilities"`
		} `json:"extra_specs"`
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"volume_type"`
}

//
// Updates the extra specifications assigned to a volume type.
func updateVolumeTypeExtraSpecs(request RequestHandlerFn, args UpdateVolumeTypeExtraSpecsParams) (*UpdateVolumeTypeExtraSpecsResults, error) {

	argsAsJson, err := json.Marshal(args)
	if err != nil {
		return nil, err
	}

	url := "https://volume.example.com/v2/%7Btenant_id%7D/types/%7Bvolume_type_id%7D"

	url = strings.Replace(url, "%7Btenant_id%7D", args.TenantId, -1)
	url = strings.Replace(url, "%7Bvolume_type_id%7D", args.VolumeTypeId, -1)

	var req *http.Request
	if string(argsAsJson) != "{}" {
		req, err = http.NewRequest("PUT", url, bytes.NewBuffer(argsAsJson))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
	} else {
		req, err = http.NewRequest("PUT", url, nil)
		if err != nil {
			return nil, err
		}
	}

	resp, err := request(req)
	if err != nil {
		return nil, err
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	switch resp.StatusCode {
	default:
		return nil, fmt.Errorf("invalid status (%d): %s", resp.StatusCode, body)
	case 200:
		break
	}

	var results UpdateVolumeTypeExtraSpecsResults
	json.Unmarshal(body, &results)

	return &results, nil
}

type GetSnapshotsSimpleParams struct {

	//
	// The unique identifier of the tenant or account.
	TenantId string `json:"-"`
}

type GetSnapshotsSimpleResults struct {
	Snapshots []struct {
		CreatedAt   string   `json:"created_at"`
		Description string   `json:"description"`
		ID          string   `json:"id"`
		Metadata    struct{} `json:"metadata"`
		Name        string   `json:"name"`
		Size        int      `json:"size"`
		Status      string   `json:"status"`
		VolumeID    string   `json:"volume_id"`
	} `json:"snapshots"`
}

//
// Lists summary information for all Block Storage snapshots that the tenant who submits the request can access.
func getSnapshotsSimple(request RequestHandlerFn, args GetSnapshotsSimpleParams) (*GetSnapshotsSimpleResults, error) {

	argsAsJson, err := json.Marshal(args)
	if err != nil {
		return nil, err
	}

	url := "https://volume.example.com/v2/%7Btenant_id%7D/snapshots"

	url = strings.Replace(url, "%7Btenant_id%7D", args.TenantId, -1)

	var req *http.Request
	if string(argsAsJson) != "{}" {
		req, err = http.NewRequest("GET", url, bytes.NewBuffer(argsAsJson))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
	} else {
		req, err = http.NewRequest("GET", url, nil)
		if err != nil {
			return nil, err
		}
	}

	resp, err := request(req)
	if err != nil {
		return nil, err
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	switch resp.StatusCode {
	default:
		return nil, fmt.Errorf("invalid status (%d): %s", resp.StatusCode, body)
	case 200:
		break
	}

	var results GetSnapshotsSimpleResults
	json.Unmarshal(body, &results)

	return &results, nil
}

type UpdateSnapshotSnapshotParams struct {

	//  Describes the snapshot.
	Description string `json:"description,omitempty"`

	//  The name of the snapshot.
	Name string `json:"name,omitempty"`
}

type UpdateSnapshotParams struct {

	// Snapshot is required.

	Snapshot UpdateSnapshotSnapshotParams `json:"snapshot"`

	//
	// The unique identifier of the tenant or account.
	TenantId string `json:"-"`

	// SnapshotId is required.
	//
	// The unique identifier of an existing snapshot.
	SnapshotId string `json:"-"`
}

type UpdateSnapshotResults struct {
	Snapshot struct {
		CreatedAt   string `json:"created_at"`
		Description string `json:"description"`
		ID          string `json:"id"`
		Name        string `json:"name"`
		Size        int    `json:"size"`
		Status      string `json:"status"`
		VolumeID    string `json:"volume_id"`
	} `json:"snapshot"`
}

//
// Updates a specified snapshot.
func updateSnapshot(request RequestHandlerFn, args UpdateSnapshotParams) (*UpdateSnapshotResults, error) {

	argsAsJson, err := json.Marshal(args)
	if err != nil {
		return nil, err
	}

	url := "https://volume.example.com/v2/%7Btenant_id%7D/snapshots/%7Bsnapshot_id%7D"

	url = strings.Replace(url, "%7Btenant_id%7D", args.TenantId, -1)
	url = strings.Replace(url, "%7Bsnapshot_id%7D", args.SnapshotId, -1)

	var req *http.Request
	if string(argsAsJson) != "{}" {
		req, err = http.NewRequest("PUT", url, bytes.NewBuffer(argsAsJson))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
	} else {
		req, err = http.NewRequest("PUT", url, nil)
		if err != nil {
			return nil, err
		}
	}

	resp, err := request(req)
	if err != nil {
		return nil, err
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	switch resp.StatusCode {
	default:
		return nil, fmt.Errorf("invalid status (%d): %s", resp.StatusCode, body)
	case 200:
		break
	}

	var results UpdateSnapshotResults
	json.Unmarshal(body, &results)

	return &results, nil
}

type ShowSnapshotMetadataParams struct {

	//
	// The unique identifier of the tenant or account.
	TenantId string `json:"-"`

	// SnapshotId is required.
	//
	// The unique identifier of an existing snapshot.
	SnapshotId string `json:"-"`
}

type ShowSnapshotMetadataResults struct {
	Snapshot struct {
		CreatedAt   string      `json:"created_at"`
		Description interface{} `json:"description"`
		ID          string      `json:"id"`
		Metadata    struct {
			Key string `json:"key"`
		} `json:"metadata"`
		Name                                      string `json:"name"`
		Os_Extended_Snapshot_Attributes_Progress  string `json:"os-extended-snapshot-attributes:progress"`
		Os_Extended_Snapshot_Attributes_ProjectID string `json:"os-extended-snapshot-attributes:project_id"`
		Size                                      int    `json:"size"`
		Status                                    string `json:"status"`
		VolumeID                                  string `json:"volume_id"`
	} `json:"snapshot"`
}

//
// Shows the metadata for a specified snapshot.
func showSnapshotMetadata(request RequestHandlerFn, args ShowSnapshotMetadataParams) (*ShowSnapshotMetadataResults, error) {

	argsAsJson, err := json.Marshal(args)
	if err != nil {
		return nil, err
	}

	url := "https://volume.example.com/v2/%7Btenant_id%7D/snapshots/%7Bsnapshot_id%7D/metadata"

	url = strings.Replace(url, "%7Btenant_id%7D", args.TenantId, -1)
	url = strings.Replace(url, "%7Bsnapshot_id%7D", args.SnapshotId, -1)

	var req *http.Request
	if string(argsAsJson) != "{}" {
		req, err = http.NewRequest("GET", url, bytes.NewBuffer(argsAsJson))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
	} else {
		req, err = http.NewRequest("GET", url, nil)
		if err != nil {
			return nil, err
		}
	}

	resp, err := request(req)
	if err != nil {
		return nil, err
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	switch resp.StatusCode {
	default:
		return nil, fmt.Errorf("invalid status (%d): %s", resp.StatusCode, body)
	case 200:
		break
	}

	var results ShowSnapshotMetadataResults
	json.Unmarshal(body, &results)

	return &results, nil
}

type GetVolumesDetailParams struct {

	//
	// The unique identifier of the tenant or account.
	TenantId string `json:"-"`
}

type GetVolumesDetailResults struct {
	Volumes []struct {
		Attachments      []interface{} `json:"attachments"`
		AvailabilityZone string        `json:"availability_zone"`
		CreatedAt        string        `json:"created_at"`
		Description      string        `json:"description"`
		ID               string        `json:"id"`
		Links            []struct {
			Href string `json:"href"`
			Rel  string `json:"rel"`
		} `json:"links"`
		Metadata struct {
			Contents string `json:"contents"`
		} `json:"metadata"`
		Name                        string      `json:"name"`
		Os_Vol_Host_Attr_Host       string      `json:"os-vol-host-attr:host"`
		Os_Vol_Tenant_Attr_TenantID string      `json:"os-vol-tenant-attr:tenant_id"`
		Size                        int         `json:"size"`
		SnapshotID                  interface{} `json:"snapshot_id"`
		SourceVolid                 interface{} `json:"source_volid"`
		Status                      string      `json:"status"`
		VolumeType                  string      `json:"volume_type"`
	} `json:"volumes"`
}

//
// Lists detailed information for all Block Storage volumes that the tenant who submits the request can access.
func getVolumesDetail(request RequestHandlerFn, args GetVolumesDetailParams) (*GetVolumesDetailResults, error) {

	argsAsJson, err := json.Marshal(args)
	if err != nil {
		return nil, err
	}

	url := "https://volume.example.com/v2/%7Btenant_id%7D/volumes/detail"

	url = strings.Replace(url, "%7Btenant_id%7D", args.TenantId, -1)

	var req *http.Request
	if string(argsAsJson) != "{}" {
		req, err = http.NewRequest("GET", url, bytes.NewBuffer(argsAsJson))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
	} else {
		req, err = http.NewRequest("GET", url, nil)
		if err != nil {
			return nil, err
		}
	}

	resp, err := request(req)
	if err != nil {
		return nil, err
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	switch resp.StatusCode {
	default:
		return nil, fmt.Errorf("invalid status (%d): %s", resp.StatusCode, body)
	case 200:
		break
	}

	var results GetVolumesDetailResults
	json.Unmarshal(body, &results)

	return &results, nil
}

type GetVolumeParams struct {

	//
	// The unique identifier of the tenant or account.
	TenantId string `json:"-"`

	// VolumeId is required.
	//
	// The unique identifier of an existing volume.
	VolumeId string `json:"-"`
}

type GetVolumeResults struct {
	Volume struct {
		Attachments      []interface{} `json:"attachments"`
		AvailabilityZone string        `json:"availability_zone"`
		Bootable         string        `json:"bootable"`
		CreatedAt        string        `json:"created_at"`
		Description      string        `json:"description"`
		ID               string        `json:"id"`
		Links            []struct {
			Href string `json:"href"`
			Rel  string `json:"rel"`
		} `json:"links"`
		Metadata struct {
			Contents string `json:"contents"`
		} `json:"metadata"`
		Name                        string      `json:"name"`
		Os_Vol_Host_Attr_Host       string      `json:"os-vol-host-attr:host"`
		Os_Vol_Tenant_Attr_TenantID string      `json:"os-vol-tenant-attr:tenant_id"`
		Size                        int         `json:"size"`
		SnapshotID                  interface{} `json:"snapshot_id"`
		SourceVolid                 interface{} `json:"source_volid"`
		Status                      string      `json:"status"`
		VolumeType                  string      `json:"volume_type"`
	} `json:"volume"`
}

//
// Shows information about a specified volume.
// Preconditions
//
// The specified volume must exist. :
func getVolume(request RequestHandlerFn, args GetVolumeParams) (*GetVolumeResults, error) {

	argsAsJson, err := json.Marshal(args)
	if err != nil {
		return nil, err
	}

	url := "https://volume.example.com/v2/%7Btenant_id%7D/volumes/%7Bvolume_id%7D"

	url = strings.Replace(url, "%7Btenant_id%7D", args.TenantId, -1)
	url = strings.Replace(url, "%7Bvolume_id%7D", args.VolumeId, -1)

	var req *http.Request
	if string(argsAsJson) != "{}" {
		req, err = http.NewRequest("GET", url, bytes.NewBuffer(argsAsJson))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
	} else {
		req, err = http.NewRequest("GET", url, nil)
		if err != nil {
			return nil, err
		}
	}

	resp, err := request(req)
	if err != nil {
		return nil, err
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	switch resp.StatusCode {
	default:
		return nil, fmt.Errorf("invalid status (%d): %s", resp.StatusCode, body)
	case 200:
		break
	}

	var results GetVolumeResults
	json.Unmarshal(body, &results)

	return &results, nil
}

type UpdateVolumeVolumeParams struct {
	Name string `json:"name,omitempty"`

	Description string `json:"description,omitempty"`
}

type UpdateVolumeParams struct {

	// Volume is required.

	Volume UpdateVolumeVolumeParams `json:"volume"`

	//
	// The unique identifier of the tenant or account.
	TenantId string `json:"-"`

	// VolumeId is required.
	//
	// The unique identifier of an existing volume.
	VolumeId string `json:"-"`
}

type UpdateVolumeResults struct {
	Volume struct {
		Attachments      []interface{} `json:"attachments"`
		AvailabilityZone string        `json:"availability_zone"`
		CreatedAt        string        `json:"created_at"`
		Description      string        `json:"description"`
		ID               string        `json:"id"`
		Links            []struct {
			Href string `json:"href"`
			Rel  string `json:"rel"`
		} `json:"links"`
		Metadata struct {
			Contents string `json:"contents"`
		} `json:"metadata"`
		Name        string      `json:"name"`
		Size        int         `json:"size"`
		SnapshotID  interface{} `json:"snapshot_id"`
		SourceVolid interface{} `json:"source_volid"`
		Status      string      `json:"status"`
		VolumeType  string      `json:"volume_type"`
	} `json:"volume"`
}

//
// Updates a volume.
func updateVolume(request RequestHandlerFn, args UpdateVolumeParams) (*UpdateVolumeResults, error) {

	argsAsJson, err := json.Marshal(args)
	if err != nil {
		return nil, err
	}

	url := "https://volume.example.com/v2/%7Btenant_id%7D/volumes/%7Bvolume_id%7D"

	url = strings.Replace(url, "%7Btenant_id%7D", args.TenantId, -1)
	url = strings.Replace(url, "%7Bvolume_id%7D", args.VolumeId, -1)

	var req *http.Request
	if string(argsAsJson) != "{}" {
		req, err = http.NewRequest("PUT", url, bytes.NewBuffer(argsAsJson))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
	} else {
		req, err = http.NewRequest("PUT", url, nil)
		if err != nil {
			return nil, err
		}
	}

	resp, err := request(req)
	if err != nil {
		return nil, err
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	switch resp.StatusCode {
	default:
		return nil, fmt.Errorf("invalid status (%d): %s", resp.StatusCode, body)
	case 200:
		break
	}

	var results UpdateVolumeResults
	json.Unmarshal(body, &results)

	return &results, nil
}

type DeleteVolumeParams struct {

	//
	// The unique identifier of the tenant or account.
	TenantId string `json:"-"`

	// VolumeId is required.
	//
	// The unique identifier of an existing volume.
	VolumeId string `json:"-"`
}

type DeleteVolumeResults struct {
}

//
// Deletes a specified volume.
// Preconditions
//
// Volume status must be available, in-use, error, or error_restoring.
// You cannot already have a snapshot related to the specified volume.
// You cannot delete a volume that is in a migration. :
// Asynchronous Postconditions
//
// The volume is deleted in volume index.
// The volume managed by OpenStack Block Storage is deleted in storage node. :
// Troubleshooting
//
// If volume status remains in deleting or becomes error_deleting the request failed. Ensure you meet the preconditions then investigate the storage backend.
// The volume managed by OpenStack Block Storage is not deleted from the storage system. :
func deleteVolume(request RequestHandlerFn, args DeleteVolumeParams) (*DeleteVolumeResults, error) {

	argsAsJson, err := json.Marshal(args)
	if err != nil {
		return nil, err
	}

	url := "https://volume.example.com/v2/%7Btenant_id%7D/volumes/%7Bvolume_id%7D"

	url = strings.Replace(url, "%7Btenant_id%7D", args.TenantId, -1)
	url = strings.Replace(url, "%7Bvolume_id%7D", args.VolumeId, -1)

	var req *http.Request
	if string(argsAsJson) != "{}" {
		req, err = http.NewRequest("DELETE", url, bytes.NewBuffer(argsAsJson))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
	} else {
		req, err = http.NewRequest("DELETE", url, nil)
		if err != nil {
			return nil, err
		}
	}

	resp, err := request(req)
	if err != nil {
		return nil, err
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	switch resp.StatusCode {
	default:
		return nil, fmt.Errorf("invalid status (%d): %s", resp.StatusCode, body)
	case 202:
		break
	}

	var results DeleteVolumeResults
	json.Unmarshal(body, &results)

	return &results, nil
}

type CreateVolumeTypeVolumeTypeExtraSpecsParams struct {
	Capabilities string `json:"capabilities,omitempty"`
}

type CreateVolumeTypeVolumeTypeParams struct {

	//  The name of the volume type.
	Name string `json:"name,omitempty"`

	ExtraSpecs CreateVolumeTypeVolumeTypeExtraSpecsParams `json:"extra_specs,omitempty"`
}

type CreateVolumeTypeParams struct {

	// VolumeType is required.
	//  A partial representation of a volume type used in the creation process.
	VolumeType CreateVolumeTypeVolumeTypeParams `json:"volume_type"`

	//
	// The unique identifier of the tenant or account.
	TenantId string `json:"-"`
}

type CreateVolumeTypeResults struct {
	VolumeType struct {
		ExtraSpecs struct {
			Capabilities string `json:"capabilities"`
		} `json:"extra_specs"`
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"volume_type"`
}

//
// Creates a volume type.
func createVolumeType(request RequestHandlerFn, args CreateVolumeTypeParams) (*CreateVolumeTypeResults, error) {

	argsAsJson, err := json.Marshal(args)
	if err != nil {
		return nil, err
	}

	url := "https://volume.example.com/v2/%7Btenant_id%7D/types"

	url = strings.Replace(url, "%7Btenant_id%7D", args.TenantId, -1)

	var req *http.Request
	if string(argsAsJson) != "{}" {
		req, err = http.NewRequest("POST", url, bytes.NewBuffer(argsAsJson))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
	} else {
		req, err = http.NewRequest("POST", url, nil)
		if err != nil {
			return nil, err
		}
	}

	resp, err := request(req)
	if err != nil {
		return nil, err
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	switch resp.StatusCode {
	default:
		return nil, fmt.Errorf("invalid status (%d): %s", resp.StatusCode, body)
	case 200:
		break
	}

	var results CreateVolumeTypeResults
	json.Unmarshal(body, &results)

	return &results, nil
}

type CreateSnapshotSnapshotParams struct {

	//  [True/False] Indicate whether to snapshot, even if the volume is attached. Default==False.
	Force bool `json:"force,omitempty"`

	//  Name of the snapshot. The default is None.
	Name string `json:"name,omitempty"`

	//  Description of the snapshot. The default is None.
	Description string `json:"description,omitempty"`

	// VolumeId is required.
	//  To create a snapshot from an existing volume, specify the ID of the existing volume.
	VolumeId string `json:"volume_id"`
}

type CreateSnapshotParams struct {

	// Snapshot is required.
	//  A partial representation of a snapshot used in the creation process.
	Snapshot CreateSnapshotSnapshotParams `json:"snapshot"`

	//
	// The unique identifier of the tenant or account.
	TenantId string `json:"-"`
}

type CreateSnapshotResults struct {
	Snapshot struct {
		CreatedAt   string   `json:"created_at"`
		Description string   `json:"description"`
		ID          string   `json:"id"`
		Metadata    struct{} `json:"metadata"`
		Name        string   `json:"name"`
		Size        int      `json:"size"`
		Status      string   `json:"status"`
		VolumeID    string   `json:"volume_id"`
	} `json:"snapshot"`
}

//
// Creates a snapshot, which is a point-in-time complete copy of a volume. You can create a volume from the snapshot.
func createSnapshot(request RequestHandlerFn, args CreateSnapshotParams) (*CreateSnapshotResults, error) {

	argsAsJson, err := json.Marshal(args)
	if err != nil {
		return nil, err
	}

	url := "https://volume.example.com/v2/%7Btenant_id%7D/snapshots"

	url = strings.Replace(url, "%7Btenant_id%7D", args.TenantId, -1)

	var req *http.Request
	if string(argsAsJson) != "{}" {
		req, err = http.NewRequest("POST", url, bytes.NewBuffer(argsAsJson))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
	} else {
		req, err = http.NewRequest("POST", url, nil)
		if err != nil {
			return nil, err
		}
	}

	resp, err := request(req)
	if err != nil {
		return nil, err
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	switch resp.StatusCode {
	default:
		return nil, fmt.Errorf("invalid status (%d): %s", resp.StatusCode, body)
	case 202:
		break
	}

	var results CreateSnapshotResults
	json.Unmarshal(body, &results)

	return &results, nil
}

type GetSnapshotsDetailParams struct {

	//
	// The unique identifier of the tenant or account.
	TenantId string `json:"-"`
}

type GetSnapshotsDetailResults struct {
	Snapshots []struct {
		CreatedAt                                 string   `json:"created_at"`
		Description                               string   `json:"description"`
		ID                                        string   `json:"id"`
		Metadata                                  struct{} `json:"metadata"`
		Name                                      string   `json:"name"`
		Os_Extended_Snapshot_Attributes_Progress  string   `json:"os-extended-snapshot-attributes:progress"`
		Os_Extended_Snapshot_Attributes_ProjectID string   `json:"os-extended-snapshot-attributes:project_id"`
		Size                                      int      `json:"size"`
		Status                                    string   `json:"status"`
		VolumeID                                  string   `json:"volume_id"`
	} `json:"snapshots"`
}

//
// Lists detailed information for all Block Storage snapshots that the tenant who submits the request can access.
func getSnapshotsDetail(request RequestHandlerFn, args GetSnapshotsDetailParams) (*GetSnapshotsDetailResults, error) {

	argsAsJson, err := json.Marshal(args)
	if err != nil {
		return nil, err
	}

	url := "https://volume.example.com/v2/%7Btenant_id%7D/snapshots/detail"

	url = strings.Replace(url, "%7Btenant_id%7D", args.TenantId, -1)

	var req *http.Request
	if string(argsAsJson) != "{}" {
		req, err = http.NewRequest("GET", url, bytes.NewBuffer(argsAsJson))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
	} else {
		req, err = http.NewRequest("GET", url, nil)
		if err != nil {
			return nil, err
		}
	}

	resp, err := request(req)
	if err != nil {
		return nil, err
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	switch resp.StatusCode {
	default:
		return nil, fmt.Errorf("invalid status (%d): %s", resp.StatusCode, body)
	case 200:
		break
	}

	var results GetSnapshotsDetailResults
	json.Unmarshal(body, &results)

	return &results, nil
}

type GetSnapshotParams struct {

	//
	// The unique identifier of the tenant or account.
	TenantId string `json:"-"`

	// SnapshotId is required.
	//
	// The unique identifier of an existing snapshot.
	SnapshotId string `json:"-"`
}

type GetSnapshotResults struct {
	Snapshot struct {
		CreatedAt                                 string   `json:"created_at"`
		Description                               string   `json:"description"`
		ID                                        string   `json:"id"`
		Metadata                                  struct{} `json:"metadata"`
		Name                                      string   `json:"name"`
		Os_Extended_Snapshot_Attributes_Progress  string   `json:"os-extended-snapshot-attributes:progress"`
		Os_Extended_Snapshot_Attributes_ProjectID string   `json:"os-extended-snapshot-attributes:project_id"`
		Size                                      int      `json:"size"`
		Status                                    string   `json:"status"`
		VolumeID                                  string   `json:"volume_id"`
	} `json:"snapshot"`
}

//
// Shows information for a specified snapshot.
func getSnapshot(request RequestHandlerFn, args GetSnapshotParams) (*GetSnapshotResults, error) {

	argsAsJson, err := json.Marshal(args)
	if err != nil {
		return nil, err
	}

	url := "https://volume.example.com/v2/%7Btenant_id%7D/snapshots/%7Bsnapshot_id%7D"

	url = strings.Replace(url, "%7Btenant_id%7D", args.TenantId, -1)
	url = strings.Replace(url, "%7Bsnapshot_id%7D", args.SnapshotId, -1)

	var req *http.Request
	if string(argsAsJson) != "{}" {
		req, err = http.NewRequest("GET", url, bytes.NewBuffer(argsAsJson))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
	} else {
		req, err = http.NewRequest("GET", url, nil)
		if err != nil {
			return nil, err
		}
	}

	resp, err := request(req)
	if err != nil {
		return nil, err
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	switch resp.StatusCode {
	default:
		return nil, fmt.Errorf("invalid status (%d): %s", resp.StatusCode, body)
	case 200:
		break
	}

	var results GetSnapshotResults
	json.Unmarshal(body, &results)

	return &results, nil
}

type ListVersionsParams struct {
}

type ListVersionsResults struct {
	Versions []struct {
		ID    string `json:"id"`
		Links []struct {
			Href string `json:"href"`
			Rel  string `json:"rel"`
		} `json:"links"`
		Status  string `json:"status"`
		Updated string `json:"updated"`
	} `json:"versions"`
}

//
// Lists information about all Block Storage API versions.
func listVersions(request RequestHandlerFn, args ListVersionsParams) (*ListVersionsResults, error) {

	argsAsJson, err := json.Marshal(args)
	if err != nil {
		return nil, err
	}

	url := "https://volume.example.com/"

	var req *http.Request
	if string(argsAsJson) != "{}" {
		req, err = http.NewRequest("GET", url, bytes.NewBuffer(argsAsJson))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
	} else {
		req, err = http.NewRequest("GET", url, nil)
		if err != nil {
			return nil, err
		}
	}

	resp, err := request(req)
	if err != nil {
		return nil, err
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	switch resp.StatusCode {
	default:
		return nil, fmt.Errorf("invalid status (%d): %s", resp.StatusCode, body)
	case 200, 300:
		break
	}

	var results ListVersionsResults
	json.Unmarshal(body, &results)

	return &results, nil
}

type VersionDetailsParams struct {
}

type VersionDetailsResults struct {
	Version struct {
		ID    string `json:"id"`
		Links []struct {
			Href string `json:"href"`
			Rel  string `json:"rel"`
		} `json:"links"`
		Media_Types []struct {
			Base string `json:"base"`
			Type string `json:"type"`
		} `json:"media-types"`
		Status  string `json:"status"`
		Updated string `json:"updated"`
	} `json:"version"`
}

//
// Shows details for Block Storage API v2.
func versionDetails(request RequestHandlerFn, args VersionDetailsParams) (*VersionDetailsResults, error) {

	argsAsJson, err := json.Marshal(args)
	if err != nil {
		return nil, err
	}

	url := ""

	var req *http.Request
	if string(argsAsJson) != "{}" {
		req, err = http.NewRequest("GET", url, bytes.NewBuffer(argsAsJson))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
	} else {
		req, err = http.NewRequest("GET", url, nil)
		if err != nil {
			return nil, err
		}
	}

	resp, err := request(req)
	if err != nil {
		return nil, err
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	switch resp.StatusCode {
	default:
		return nil, fmt.Errorf("invalid status (%d): %s", resp.StatusCode, body)
	case 200, 203:
		break
	}

	var results VersionDetailsResults
	json.Unmarshal(body, &results)

	return &results, nil
}

type CreateVolumeVolumeParams struct {

	//  To create a volume from an existing volume, specify the ID of the existing volume. If specified, the volume is created with same size of the source volume.
	SourceVolid string `json:"source_volid,omitempty"`

	//  To create a volume from an existing snapshot, specify the ID of the existing volume snapshot. If specified, the volume is created in same availability zone and with same size of the snapshot.
	SnapshotId string `json:"snapshot_id,omitempty"`

	//  The ID of the image from which you want to create the volume. Required to create a bootable volume.
	ImageRef string `json:"imageRef,omitempty"`

	//  The associated volume type.
	VolumeType string `json:"volume_type,omitempty"`

	//  Enables or disables the bootable attribute. You can boot an instance from a bootable volume.
	Bootable bool `json:"bootable,omitempty"`

	//  One or more metadata key and value pairs to associate with the volume.
	Metadata interface{} `json:"metadata,omitempty"`

	//  The availability zone.
	AvailabilityZone string `json:"availability_zone,omitempty"`

	//  The volume description.
	Description string `json:"description,omitempty"`

	// Size is required.
	//  The size of the volume, in GBs.
	Size int `json:"size"`

	//  The volume name.
	Name string `json:"name,omitempty"`
}

type CreateVolumeParams struct {

	// Volume is required.

	Volume CreateVolumeVolumeParams `json:"volume"`

	//
	// The unique identifier of the tenant or account.
	TenantId string `json:"-"`
}

type CreateVolumeResults struct {
	Volume struct {
		Attachments      []interface{} `json:"attachments"`
		AvailabilityZone string        `json:"availability_zone"`
		Bootable         string        `json:"bootable"`
		CreatedAt        string        `json:"created_at"`
		Description      interface{}   `json:"description"`
		ID               string        `json:"id"`
		Metadata         struct{}      `json:"metadata"`
		Name             string        `json:"name"`
		Size             int           `json:"size"`
		SnapshotID       interface{}   `json:"snapshot_id"`
		SourceVolid      interface{}   `json:"source_volid"`
		Status           string        `json:"status"`
		VolumeType       string        `json:"volume_type"`
	} `json:"volume"`
}

//
// Creates a volume.
// To create a bootable volume, include the image ID and set the bootable flag to true in the request body.
// Preconditions
//
// The user must have enough volume storage quota remaining to create a volume of size requested. :
// Asynchronous Postconditions
//
// With correct permissions, you can see the volume status as available through API calls.
// With correct access, you can see the created volume in the storage system that OpenStack Block Storage manages. :
// Troubleshooting
//
// If volume status remains creating or shows another error status, the request failed. Ensure you meet the preconditions then investigate the storage backend.
// Volume is not created in the storage system which OpenStack Block Storage manages.
// The storage node needs enough free storage space to match the specified size of the volume creation request. :
func createVolume(request RequestHandlerFn, args CreateVolumeParams) (*CreateVolumeResults, error) {

	argsAsJson, err := json.Marshal(args)
	if err != nil {
		return nil, err
	}

	url := "https://volume.example.com/v2/%7Btenant_id%7D/volumes"

	url = strings.Replace(url, "%7Btenant_id%7D", args.TenantId, -1)

	var req *http.Request
	if string(argsAsJson) != "{}" {
		req, err = http.NewRequest("POST", url, bytes.NewBuffer(argsAsJson))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
	} else {
		req, err = http.NewRequest("POST", url, nil)
		if err != nil {
			return nil, err
		}
	}

	resp, err := request(req)
	if err != nil {
		return nil, err
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	switch resp.StatusCode {
	default:
		return nil, fmt.Errorf("invalid status (%d): %s", resp.StatusCode, body)
	case 202:
		break
	}

	var results CreateVolumeResults
	json.Unmarshal(body, &results)

	return &results, nil
}

type GetVolumesSimpleParams struct {

	//
	// The unique identifier of the tenant or account.
	TenantId string `json:"-"`
}

type GetVolumesSimpleResults struct {
	Volumes []struct {
		ID    string `json:"id"`
		Links []struct {
			Href string `json:"href"`
			Rel  string `json:"rel"`
		} `json:"links"`
		Name string `json:"name"`
	} `json:"volumes"`
}

//
// Lists summary information for all Block Storage volumes that the tenant who submits the request can access.
func getVolumesSimple(request RequestHandlerFn, args GetVolumesSimpleParams) (*GetVolumesSimpleResults, error) {

	argsAsJson, err := json.Marshal(args)
	if err != nil {
		return nil, err
	}

	url := "https://volume.example.com/v2/%7Btenant_id%7D/volumes"

	url = strings.Replace(url, "%7Btenant_id%7D", args.TenantId, -1)

	var req *http.Request
	if string(argsAsJson) != "{}" {
		req, err = http.NewRequest("GET", url, bytes.NewBuffer(argsAsJson))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
	} else {
		req, err = http.NewRequest("GET", url, nil)
		if err != nil {
			return nil, err
		}
	}

	resp, err := request(req)
	if err != nil {
		return nil, err
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	switch resp.StatusCode {
	default:
		return nil, fmt.Errorf("invalid status (%d): %s", resp.StatusCode, body)
	case 200:
		break
	}

	var results GetVolumesSimpleResults
	json.Unmarshal(body, &results)

	return &results, nil
}

type GetVolumeTypeParams struct {

	//
	// The unique identifier of the tenant or account.
	TenantId string `json:"-"`

	// VolumeTypeId is required.
	//
	// The unique identifier for an existing volume type.
	VolumeTypeId string `json:"-"`
}

type GetVolumeTypeResults struct {
	VolumeType struct {
		ExtraSpecs struct {
			Capabilities string `json:"capabilities"`
		} `json:"extra_specs"`
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"volume_type"`
}

//
// Shows information about a specified volume type.
func getVolumeType(request RequestHandlerFn, args GetVolumeTypeParams) (*GetVolumeTypeResults, error) {

	argsAsJson, err := json.Marshal(args)
	if err != nil {
		return nil, err
	}

	url := "https://volume.example.com/v2/%7Btenant_id%7D/types/%7Bvolume_type_id%7D"

	url = strings.Replace(url, "%7Btenant_id%7D", args.TenantId, -1)
	url = strings.Replace(url, "%7Bvolume_type_id%7D", args.VolumeTypeId, -1)

	var req *http.Request
	if string(argsAsJson) != "{}" {
		req, err = http.NewRequest("GET", url, bytes.NewBuffer(argsAsJson))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
	} else {
		req, err = http.NewRequest("GET", url, nil)
		if err != nil {
			return nil, err
		}
	}

	resp, err := request(req)
	if err != nil {
		return nil, err
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	switch resp.StatusCode {
	default:
		return nil, fmt.Errorf("invalid status (%d): %s", resp.StatusCode, body)
	case 200:
		break
	}

	var results GetVolumeTypeResults
	json.Unmarshal(body, &results)

	return &results, nil
}

type DeleteSnapshotParams struct {

	//
	// The unique identifier of the tenant or account.
	TenantId string `json:"-"`

	// SnapshotId is required.
	//
	// The unique identifier of an existing snapshot.
	SnapshotId string `json:"-"`
}

type DeleteSnapshotResults struct {
}

//
// Deletes a specified snapshot.
func deleteSnapshot(request RequestHandlerFn, args DeleteSnapshotParams) (*DeleteSnapshotResults, error) {

	argsAsJson, err := json.Marshal(args)
	if err != nil {
		return nil, err
	}

	url := "https://volume.example.com/v2/%7Btenant_id%7D/snapshots/%7Bsnapshot_id%7D"

	url = strings.Replace(url, "%7Btenant_id%7D", args.TenantId, -1)
	url = strings.Replace(url, "%7Bsnapshot_id%7D", args.SnapshotId, -1)

	var req *http.Request
	if string(argsAsJson) != "{}" {
		req, err = http.NewRequest("DELETE", url, bytes.NewBuffer(argsAsJson))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
	} else {
		req, err = http.NewRequest("DELETE", url, nil)
		if err != nil {
			return nil, err
		}
	}

	resp, err := request(req)
	if err != nil {
		return nil, err
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	switch resp.StatusCode {
	default:
		return nil, fmt.Errorf("invalid status (%d): %s", resp.StatusCode, body)
	case 202:
		break
	}

	var results DeleteSnapshotResults
	json.Unmarshal(body, &results)

	return &results, nil
}

type ListExtensionsCinderV2Params struct {
}

type ListExtensionsCinderV2Results struct {
	Extensions []struct {
		Alias       string        `json:"alias"`
		Description string        `json:"description"`
		Links       []interface{} `json:"links"`
		Name        string        `json:"name"`
		Namespace   string        `json:"namespace"`
		Updated     string        `json:"updated"`
	} `json:"extensions"`
}

//
// Lists Block Storage API extensions.
func listExtensionsCinderV2(request RequestHandlerFn, args ListExtensionsCinderV2Params) (*ListExtensionsCinderV2Results, error) {

	argsAsJson, err := json.Marshal(args)
	if err != nil {
		return nil, err
	}

	url := ""

	var req *http.Request
	if string(argsAsJson) != "{}" {
		req, err = http.NewRequest("GET", url, bytes.NewBuffer(argsAsJson))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
	} else {
		req, err = http.NewRequest("GET", url, nil)
		if err != nil {
			return nil, err
		}
	}

	resp, err := request(req)
	if err != nil {
		return nil, err
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	switch resp.StatusCode {
	default:
		return nil, fmt.Errorf("invalid status (%d): %s", resp.StatusCode, body)
	case 200, 300:
		break
	}

	var results ListExtensionsCinderV2Results
	json.Unmarshal(body, &results)

	return &results, nil
}

type GetVolumeTypesParams struct {

	//
	// The unique identifier of the tenant or account.
	TenantId string `json:"-"`
}

type GetVolumeTypesResults struct {
	VolumeTypes []struct {
		ExtraSpecs struct {
			Capabilities string `json:"capabilities"`
		} `json:"extra_specs"`
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"volume_types"`
}

//
// Lists volume types.
func getVolumeTypes(request RequestHandlerFn, args GetVolumeTypesParams) (*GetVolumeTypesResults, error) {

	argsAsJson, err := json.Marshal(args)
	if err != nil {
		return nil, err
	}

	url := "https://volume.example.com/v2/%7Btenant_id%7D/types"

	url = strings.Replace(url, "%7Btenant_id%7D", args.TenantId, -1)

	var req *http.Request
	if string(argsAsJson) != "{}" {
		req, err = http.NewRequest("GET", url, bytes.NewBuffer(argsAsJson))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
	} else {
		req, err = http.NewRequest("GET", url, nil)
		if err != nil {
			return nil, err
		}
	}

	resp, err := request(req)
	if err != nil {
		return nil, err
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	switch resp.StatusCode {
	default:
		return nil, fmt.Errorf("invalid status (%d): %s", resp.StatusCode, body)
	case 200:
		break
	}

	var results GetVolumeTypesResults
	json.Unmarshal(body, &results)

	return &results, nil
}

type DeleteVolumeTypeParams struct {

	//
	// The unique identifier of the tenant or account.
	TenantId string `json:"-"`

	// VolumeTypeId is required.
	//
	// The unique identifier for an existing volume type.
	VolumeTypeId string `json:"-"`
}

type DeleteVolumeTypeResults struct {
}

//
// Deletes a specified volume type.
func deleteVolumeType(request RequestHandlerFn, args DeleteVolumeTypeParams) (*DeleteVolumeTypeResults, error) {

	argsAsJson, err := json.Marshal(args)
	if err != nil {
		return nil, err
	}

	url := "https://volume.example.com/v2/%7Btenant_id%7D/types/%7Bvolume_type_id%7D"

	url = strings.Replace(url, "%7Btenant_id%7D", args.TenantId, -1)
	url = strings.Replace(url, "%7Bvolume_type_id%7D", args.VolumeTypeId, -1)

	var req *http.Request
	if string(argsAsJson) != "{}" {
		req, err = http.NewRequest("DELETE", url, bytes.NewBuffer(argsAsJson))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
	} else {
		req, err = http.NewRequest("DELETE", url, nil)
		if err != nil {
			return nil, err
		}
	}

	resp, err := request(req)
	if err != nil {
		return nil, err
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	switch resp.StatusCode {
	default:
		return nil, fmt.Errorf("invalid status (%d): %s", resp.StatusCode, body)
	case 202:
		break
	}

	var results DeleteVolumeTypeResults
	json.Unmarshal(body, &results)

	return &results, nil
}

type UpdateSnapshotMetadataMetadataParams struct {

	// Key is required.

	Key string `json:"key"`
}

type UpdateSnapshotMetadataParams struct {

	// Metadata is required.
	//  One or more metadata key and value pairs to set or unset for the snapshot. To unset a metadata key value, specify only the key name. To set a metadata key value, specify the key and value pair. The Block Storage server does not respect case-sensitive key names. For example, if you specify both "key": "v1" and "KEY": "V1", the server sets and returns only the KEY key and value pair.
	Metadata UpdateSnapshotMetadataMetadataParams `json:"metadata"`

	//
	// The unique identifier of the tenant or account.
	TenantId string `json:"-"`

	// SnapshotId is required.
	//
	// The unique identifier of an existing snapshot.
	SnapshotId string `json:"-"`
}

type UpdateSnapshotMetadataResults struct {
	Metadata struct {
		Key string `json:"key"`
	} `json:"metadata"`
}

//
// Updates the metadata for a specified snapshot.
func updateSnapshotMetadata(request RequestHandlerFn, args UpdateSnapshotMetadataParams) (*UpdateSnapshotMetadataResults, error) {

	argsAsJson, err := json.Marshal(args)
	if err != nil {
		return nil, err
	}

	url := "https://volume.example.com/v2/%7Btenant_id%7D/snapshots/%7Bsnapshot_id%7D/metadata"

	url = strings.Replace(url, "%7Btenant_id%7D", args.TenantId, -1)
	url = strings.Replace(url, "%7Bsnapshot_id%7D", args.SnapshotId, -1)

	var req *http.Request
	if string(argsAsJson) != "{}" {
		req, err = http.NewRequest("PUT", url, bytes.NewBuffer(argsAsJson))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
	} else {
		req, err = http.NewRequest("PUT", url, nil)
		if err != nil {
			return nil, err
		}
	}

	resp, err := request(req)
	if err != nil {
		return nil, err
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	switch resp.StatusCode {
	default:
		return nil, fmt.Errorf("invalid status (%d): %s", resp.StatusCode, body)
	case 200:
		break
	}

	var results UpdateSnapshotMetadataResults
	json.Unmarshal(body, &results)

	return &results, nil
}
