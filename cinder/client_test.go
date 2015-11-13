// Copyright 2015 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

package cinder

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"

	gc "gopkg.in/check.v1"
)

const (
	testId        = "test-id"
	testToken     = "test-token"
	testTime      = "test-time"
	testDescr     = "test-description"
	testName      = "test-name"
	testAttrProgr = "test-attribute-prog"
	testAttrProj  = "test-attribute-proj"
	testStatus    = "test-status"
)

var _ = gc.Suite(&CinderTestSuite{})

type CinderTestSuite struct {
	client *Client
	*http.ServeMux
}

func (s *CinderTestSuite) SetUpSuite(c *gc.C) {

	if *live {
		return
	}

	endpoint, err := url.Parse("http://volume.testing/v2/" + testId)
	c.Assert(err, gc.IsNil)

	cinderClient := NewClient(
		testId,
		endpoint,
		SetAuthHeaderFn(func() string { return testToken }, s.localDo),
	)
	s.client = cinderClient
}

func (s *CinderTestSuite) SetUpTest(c *gc.C) {

	if *live {
		c.Skip("Only running live tests.")
	}

	// We want a fresh Muxer so that any paths that aren't explicitly
	// set up by the test are treated as errors.
	s.ServeMux = http.NewServeMux()
}

func (s *CinderTestSuite) TestCreateSnapshot(c *gc.C) {

	snapReq := CreateSnapshotSnapshotParams{
		VolumeId:    testId,
		Name:        testName,
		Description: testDescr,
	}

	numCalls := 0
	s.HandleFunc("/v2/"+testId+"/snapshots", func(w http.ResponseWriter, req *http.Request) {
		numCalls++

		c.Check(req.Header["X-Auth-Token"], gc.DeepEquals, []string{testToken})

		reqBody, err := ioutil.ReadAll(req.Body)
		c.Assert(err, gc.IsNil)

		var receivedReq CreateSnapshotParams
		err = json.Unmarshal(reqBody, &receivedReq)
		c.Assert(err, gc.IsNil)

		c.Check(receivedReq, gc.DeepEquals, CreateSnapshotParams{Snapshot: snapReq})

		resp := Snapshot{
			CreatedAt:   "test-time",
			Description: receivedReq.Snapshot.Description,
			ID:          "test-id",
			Name:        receivedReq.Snapshot.Name,
			Size:        1,
			Status:      "test-status",
			VolumeID:    receivedReq.Snapshot.VolumeId,
		}
		respBody, err := json.Marshal(&CreateSnapshotResults{Snapshot: resp})
		c.Assert(err, gc.IsNil)

		w.(*responseWriter).Body = ioutil.NopCloser(bytes.NewReader(respBody))
		w.(*responseWriter).StatusCode = 202
	})

	resp, err := s.client.CreateSnapshot(snapReq)
	c.Assert(numCalls, gc.Equals, 1)
	c.Assert(err, gc.IsNil)

	c.Check(resp.Snapshot.CreatedAt, gc.Equals, testTime)
	c.Check(resp.Snapshot.Description, gc.Equals, snapReq.Description)
	c.Check(resp.Snapshot.ID, gc.Equals, testId)
	c.Check(resp.Snapshot.Name, gc.Equals, snapReq.Name)
	c.Check(resp.Snapshot.Size, gc.Equals, 1)
	c.Check(resp.Snapshot.Status, gc.Equals, testStatus)
	c.Check(resp.Snapshot.VolumeID, gc.Equals, snapReq.VolumeId)

}

func (s *CinderTestSuite) TestDeleteSnapshot(c *gc.C) {
	numCalls := 0
	s.HandleFunc("/v2/"+testId+"/snapshots/"+testId, func(w http.ResponseWriter, req *http.Request) {
		numCalls++

		w.(*responseWriter).Body = ioutil.NopCloser(bytes.NewReader([]byte{}))
		w.(*responseWriter).StatusCode = 202
	})

	err := s.client.DeleteSnapshot(testId)
	c.Assert(numCalls, gc.Equals, 1)
	c.Assert(err, gc.IsNil)
}

func (s *CinderTestSuite) TestGetSnapshot(c *gc.C) {

	numCalls := 0
	s.HandleFunc("/v2/"+testId+"/snapshots/"+testId, func(w http.ResponseWriter, req *http.Request) {
		numCalls++

		c.Check(req.Header["X-Auth-Token"], gc.DeepEquals, []string{testToken})

		resp := Snapshot{
			CreatedAt:   testTime,
			Description: testDescr,
			ID:          testId,
			Name:        testName,
			Os_Extended_Snapshot_Attributes_Progress:  testAttrProgr,
			Os_Extended_Snapshot_Attributes_ProjectID: testAttrProj,
			Size:     1,
			Status:   testStatus,
			VolumeID: testId,
		}

		respBody, err := json.Marshal(&GetSnapshotResults{Snapshot: resp})
		c.Assert(err, gc.IsNil)

		w.(*responseWriter).Response.StatusCode = 200
		w.(*responseWriter).Body = ioutil.NopCloser(bytes.NewReader(respBody))
	})

	// Test GetSnapshot
	getResp, err := s.client.GetSnapshot(testId)
	c.Assert(err, gc.IsNil)
	c.Assert(numCalls, gc.Equals, 1)

	c.Check(getResp.Snapshot.CreatedAt, gc.Not(gc.HasLen), 0)
	c.Check(getResp.Snapshot.Description, gc.Equals, testDescr)
	c.Check(getResp.Snapshot.ID, gc.Not(gc.HasLen), 0)
	c.Check(getResp.Snapshot.Name, gc.Equals, testName)
	c.Check(getResp.Snapshot.Size, gc.Equals, 1)
	c.Check(getResp.Snapshot.Status, gc.Equals, testStatus)
	c.Check(getResp.Snapshot.VolumeID, gc.Equals, testId)
}

func (s *CinderTestSuite) TestGetSnapshotDetail(c *gc.C) {

	numCalls := 0
	s.HandleFunc("/v2/"+testId+"/snapshots/detail", func(w http.ResponseWriter, req *http.Request) {
		numCalls++

		c.Check(req.Header["X-Auth-Token"], gc.DeepEquals, []string{testToken})

		resp := []Snapshot{{
			CreatedAt:   testTime,
			Description: testDescr,
			ID:          testId,
			Name:        testName,
			Os_Extended_Snapshot_Attributes_Progress:  testAttrProgr,
			Os_Extended_Snapshot_Attributes_ProjectID: testAttrProj,
			Size:     1,
			Status:   testStatus,
			VolumeID: testId,
		}}

		respBody, err := json.Marshal(&GetSnapshotsDetailResults{Snapshots: resp})
		c.Assert(err, gc.IsNil)

		w.(*responseWriter).Response.StatusCode = 200
		w.(*responseWriter).Body = ioutil.NopCloser(bytes.NewReader(respBody))
	})

	detailGetResp, err := s.client.GetSnapshotsDetail()
	c.Assert(err, gc.IsNil)
	c.Assert(detailGetResp.Snapshots, gc.HasLen, 1)
	c.Assert(numCalls, gc.Equals, 1)

	snapshot := detailGetResp.Snapshots[0]

	c.Check(snapshot.CreatedAt, gc.Equals, testTime)
	c.Check(snapshot.Description, gc.Equals, testDescr)
	c.Check(snapshot.ID, gc.Equals, testId)
	c.Check(snapshot.Name, gc.Equals, testName)
	c.Check(snapshot.Os_Extended_Snapshot_Attributes_Progress, gc.Equals, testAttrProgr)
	c.Check(snapshot.Os_Extended_Snapshot_Attributes_ProjectID, gc.Equals, testAttrProj)
	c.Check(snapshot.Size, gc.Equals, 1)
	c.Check(snapshot.Status, gc.Equals, testStatus)
	c.Check(snapshot.VolumeID, gc.Equals, testId)
}

func (s *CinderTestSuite) TestGetSnapshotSimple(c *gc.C) {

	numCalls := 0
	s.HandleFunc("/v2/"+testId+"/snapshots", func(w http.ResponseWriter, req *http.Request) {
		numCalls++

		c.Check(req.Header["X-Auth-Token"], gc.DeepEquals, []string{testToken})

		resp := []Snapshot{{
			CreatedAt:   testTime,
			Description: testDescr,
			ID:          testId,
			Name:        testName,
			Os_Extended_Snapshot_Attributes_Progress:  testAttrProgr,
			Os_Extended_Snapshot_Attributes_ProjectID: testAttrProj,
			Size:     1,
			Status:   testStatus,
			VolumeID: testId,
		}}

		respBody, err := json.Marshal(&GetSnapshotsDetailResults{Snapshots: resp})
		c.Assert(err, gc.IsNil)

		w.(*responseWriter).Response.StatusCode = 200
		w.(*responseWriter).Body = ioutil.NopCloser(bytes.NewReader(respBody))
	})

	snapshotSimpResp, err := s.client.GetSnapshotsSimple()
	c.Assert(numCalls, gc.Equals, 1)
	c.Assert(err, gc.IsNil)
	c.Assert(snapshotSimpResp.Snapshots, gc.HasLen, 1)

	snapshot := snapshotSimpResp.Snapshots[0]

	c.Check(snapshot.CreatedAt, gc.Equals, testTime)
	c.Check(snapshot.Description, gc.Equals, testDescr)
	c.Check(snapshot.ID, gc.Equals, testId)
	c.Check(snapshot.Name, gc.Equals, testName)
	c.Check(snapshot.Size, gc.Equals, 1)
	c.Check(snapshot.Status, gc.Equals, testStatus)
	c.Check(snapshot.VolumeID, gc.Equals, testId)
}

func (s *CinderTestSuite) TestShowSnapshotMetadata(c *gc.C) {

	numCalls := 0
	s.HandleFunc("/v2/"+testId+"/snapshots/"+testId+"/metadata", func(w http.ResponseWriter, req *http.Request) {
		numCalls++

		c.Check(req.Header["X-Auth-Token"], gc.DeepEquals, []string{testToken})

		resp := Snapshot{
			CreatedAt:   testTime,
			Description: testDescr,
			ID:          testId,
			Name:        testName,
			Os_Extended_Snapshot_Attributes_Progress:  testAttrProgr,
			Os_Extended_Snapshot_Attributes_ProjectID: testAttrProj,
			Size:     1,
			Status:   testStatus,
			VolumeID: testId,
		}

		respBody, err := json.Marshal(&ShowSnapshotMetadataResults{Snapshot: resp})
		c.Assert(err, gc.IsNil)

		w.(*responseWriter).Response.StatusCode = 200
		w.(*responseWriter).Body = ioutil.NopCloser(bytes.NewReader(respBody))
	})

	resp, err := s.client.ShowSnapshotMetadata(testId)
	c.Assert(err, gc.IsNil)

	c.Check(resp.Snapshot.CreatedAt, gc.Equals, testTime)
	c.Check(resp.Snapshot.Description, gc.Equals, testDescr)
	c.Check(resp.Snapshot.ID, gc.Equals, testId)
	c.Check(resp.Snapshot.Name, gc.Equals, testName)
	c.Check(resp.Snapshot.Size, gc.Equals, 1)
	c.Check(resp.Snapshot.Status, gc.Equals, testStatus)
	c.Check(resp.Snapshot.VolumeID, gc.Equals, testId)
}

func (s *CinderTestSuite) TestUpdateSnapshot(c *gc.C) {

	updateReq := UpdateSnapshotSnapshotParams{testName, testDescr}

	numCalls := 0
	s.HandleFunc("/v2/"+testId+"/snapshots/"+testId, func(w http.ResponseWriter, req *http.Request) {
		numCalls++

		c.Check(req.Header["X-Auth-Token"], gc.DeepEquals, []string{testToken})

		reqBody, err := ioutil.ReadAll(req.Body)
		c.Assert(err, gc.IsNil)

		var receivedReq UpdateSnapshotParams
		err = json.Unmarshal(reqBody, &receivedReq)
		c.Assert(err, gc.IsNil)

		c.Check(receivedReq, gc.DeepEquals, UpdateSnapshotParams{Snapshot: updateReq})

		resp := Snapshot{
			CreatedAt:   testTime,
			Description: updateReq.Description,
			ID:          testId,
			Name:        updateReq.Name,
			Size:        1,
			Status:      testStatus,
			VolumeID:    testId,
		}

		respBody, err := json.Marshal(&UpdateSnapshotResults{Snapshot: resp})
		c.Assert(err, gc.IsNil)

		w.(*responseWriter).Response.StatusCode = 200
		w.(*responseWriter).Body = ioutil.NopCloser(bytes.NewReader(respBody))
	})

	resp, err := s.client.UpdateSnapshot(testId, updateReq)
	c.Assert(numCalls, gc.Equals, 1)
	c.Assert(err, gc.IsNil)

	c.Check(resp.Snapshot.CreatedAt, gc.Equals, testTime)
	c.Check(resp.Snapshot.Description, gc.Equals, updateReq.Description)
	c.Check(resp.Snapshot.ID, gc.Equals, testId)
	c.Check(resp.Snapshot.Name, gc.Equals, updateReq.Name)
	c.Check(resp.Snapshot.Size, gc.Equals, 1)
	c.Check(resp.Snapshot.Status, gc.Equals, testStatus)
	c.Check(resp.Snapshot.VolumeID, gc.Equals, testId)
}

func (s *CinderTestSuite) TestUpdateSnapshotMetadata(c *gc.C) {

	numCalls := 0
	s.HandleFunc("/v2/"+testId+"/snapshots/"+testId+"/metadata", func(w http.ResponseWriter, req *http.Request) {
		numCalls++

		c.Check(req.Header["X-Auth-Token"], gc.DeepEquals, []string{testToken})

		reqBody, err := ioutil.ReadAll(req.Body)
		c.Assert(err, gc.IsNil)

		var receivedReq UpdateSnapshotMetadataParams
		err = json.Unmarshal(reqBody, &receivedReq)
		c.Assert(err, gc.IsNil)

		c.Check(receivedReq.Metadata.Key, gc.DeepEquals, "test-key")

		resp := struct {
			Key string `json:"key"`
		}{
			Key: receivedReq.Metadata.Key,
		}

		respBody, err := json.Marshal(&UpdateSnapshotMetadataResults{Metadata: resp})
		c.Assert(err, gc.IsNil)

		w.(*responseWriter).Response.StatusCode = 200
		w.(*responseWriter).Body = ioutil.NopCloser(bytes.NewReader(respBody))
	})

	resp, err := s.client.UpdateSnapshotMetadata(testId, "test-key")
	c.Assert(numCalls, gc.Equals, 1)
	c.Assert(err, gc.IsNil)

	c.Check(resp.Metadata.Key, gc.Equals, "test-key")
}

func (s *CinderTestSuite) TestCreateVolume(c *gc.C) {
	req := CreateVolumeVolumeParams{
		Description: testDescr,
		Size:        1,
		Name:        testName,
	}

	numCalls := 0
	s.HandleFunc("/v2/"+testId+"/volumes", func(w http.ResponseWriter, req *http.Request) {
		numCalls++

		reqBody, err := ioutil.ReadAll(req.Body)
		c.Assert(err, gc.IsNil)

		var receivedReq CreateVolumeParams
		err = json.Unmarshal(reqBody, &receivedReq)
		c.Assert(err, gc.IsNil)

		resp := Volume{
			AvailabilityZone: receivedReq.Volume.AvailabilityZone,
			Bootable:         fmt.Sprintf("%v", receivedReq.Volume.Bootable),
			CreatedAt:        "test-time",
			Description:      receivedReq.Volume.Description,
			Name:             receivedReq.Volume.Name,
			Size:             receivedReq.Volume.Size,
			SnapshotID:       testId,
			SourceVolid:      testId,
			Status:           testStatus,
			VolumeType:       "test-volume-type",
		}

		respBody, err := json.Marshal(&CreateVolumeResults{Volume: resp})
		c.Assert(err, gc.IsNil)

		w.(*responseWriter).Response.StatusCode = 202
		w.(*responseWriter).Body = ioutil.NopCloser(bytes.NewReader(respBody))
		c.Assert(w.(*responseWriter).Body, gc.NotNil)
	})

	resp, err := s.client.CreateVolume(req)
	c.Assert(numCalls, gc.Equals, 1)
	c.Assert(err, gc.IsNil)

	c.Check(resp.Volume.Name, gc.Equals, req.Name)
	c.Check(resp.Volume.AvailabilityZone, gc.Equals, req.AvailabilityZone)
	c.Check(resp.Volume.Bootable, gc.Equals, fmt.Sprintf("%v", req.Bootable))
	c.Check(resp.Volume.Description, gc.Equals, req.Description)
	c.Check(resp.Volume.Size, gc.Equals, req.Size)
	c.Check(resp.Volume.SnapshotID, gc.Equals, testId)
	c.Check(resp.Volume.SourceVolid, gc.Equals, testId)
	c.Check(resp.Volume.Status, gc.Equals, testStatus)
	c.Check(resp.Volume.VolumeType, gc.Equals, "test-volume-type")
}

func (s *CinderTestSuite) TestDeleteVolume(c *gc.C) {

	numCalls := 0
	s.HandleFunc("/v2/"+testId+"/volumes/"+testId, func(w http.ResponseWriter, req *http.Request) {
		numCalls++

		w.(*responseWriter).Response.StatusCode = 202
		w.(*responseWriter).Body = ioutil.NopCloser(bytes.NewBuffer([]byte{}))
	})

	err := s.client.DeleteVolume(testId)
	c.Assert(numCalls, gc.Equals, 1)
	c.Assert(err, gc.IsNil)
}

func (s *CinderTestSuite) TestUpdateVolume(c *gc.C) {

	updateReq := UpdateVolumeVolumeParams{testName, testDescr}

	numCalls := 0
	s.HandleFunc("/v2/"+testId+"/volumes/"+testId, func(w http.ResponseWriter, req *http.Request) {
		numCalls++

		reqBody, err := ioutil.ReadAll(req.Body)
		c.Assert(err, gc.IsNil)

		var receivedReq UpdateVolumeParams
		err = json.Unmarshal(reqBody, &receivedReq)
		c.Assert(err, gc.IsNil)

		resp := Volume{
			AvailabilityZone: "test-avail-zone",
			Bootable:         "false",
			CreatedAt:        "test-time",
			Description:      receivedReq.Volume.Description,
			Name:             receivedReq.Volume.Name,
			Size:             1,
			SnapshotID:       testId,
			SourceVolid:      testId,
			Status:           testStatus,
			VolumeType:       "test-volume-type",
		}

		respBody, err := json.Marshal(&CreateVolumeResults{Volume: resp})
		c.Assert(err, gc.IsNil)

		w.(*responseWriter).Response.StatusCode = 200
		w.(*responseWriter).Body = ioutil.NopCloser(bytes.NewReader(respBody))
		c.Assert(w.(*responseWriter).Body, gc.NotNil)
	})

	resp, err := s.client.UpdateVolume(testId, updateReq)
	c.Assert(numCalls, gc.Equals, 1)
	c.Assert(err, gc.IsNil)

	c.Check(resp.Volume.Name, gc.Equals, updateReq.Name)
	c.Check(resp.Volume.AvailabilityZone, gc.Equals, "test-avail-zone")
	c.Check(resp.Volume.Description, gc.Equals, updateReq.Description)
	c.Check(resp.Volume.Size, gc.Equals, 1)
	c.Check(resp.Volume.SnapshotID, gc.Equals, testId)
	c.Check(resp.Volume.SourceVolid, gc.Equals, testId)
	c.Check(resp.Volume.Status, gc.Equals, testStatus)
	c.Check(resp.Volume.VolumeType, gc.Equals, "test-volume-type")
}

func (s *CinderTestSuite) TestUpdateVolumeType(c *gc.C) {

	numCalls := 0
	s.HandleFunc("/v2/"+testId+"/types/"+testId, func(w http.ResponseWriter, req *http.Request) {
		numCalls++

		resp := struct {
			ExtraSpecs struct {
				Capabilities string `json:"capabilities"`
			} `json:"extra_specs"`
			ID   string `json:"id"`
			Name string `json:"name"`
		}{
			ID:   testId,
			Name: testName,
		}

		respBody, err := json.Marshal(&UpdateVolumeTypeResults{VolumeType: resp})
		c.Assert(err, gc.IsNil)

		w.(*responseWriter).Response.StatusCode = 200
		w.(*responseWriter).Body = ioutil.NopCloser(bytes.NewBuffer(respBody))
	})

	resp, err := s.client.UpdateVolumeType(testId, "test-volume-type")
	c.Assert(numCalls, gc.Equals, 1)
	c.Assert(err, gc.IsNil)

	c.Check(resp.VolumeType.Name, gc.Equals, testName)
	c.Check(resp.VolumeType.ID, gc.Equals, testId)
}

func (s *CinderTestSuite) TestUpdateVolumeTypeExtraSpecs(c *gc.C) {

	const (
		testType  = "test-type"
		testSpecs = "test-xtra-specs"
	)

	numCalls := 0
	s.HandleFunc("/v2/"+testId+"/types/"+testId, func(w http.ResponseWriter, req *http.Request) {
		numCalls++

		reqBody, err := ioutil.ReadAll(req.Body)
		c.Assert(err, gc.IsNil)

		var receivedReq UpdateVolumeTypeExtraSpecsParams
		err = json.Unmarshal(reqBody, &receivedReq)
		c.Assert(err, gc.IsNil)
		c.Check(receivedReq.ExtraSpecs, gc.Equals, testSpecs)
		c.Check(receivedReq.VolumeType, gc.Equals, testType)

		resp := struct {
			ExtraSpecs struct {
				Capabilities string `json:"capabilities"`
			} `json:"extra_specs"`
			ID   string `json:"id"`
			Name string `json:"name"`
		}{
			ID:   testId,
			Name: testName,
			ExtraSpecs: struct {
				Capabilities string `json:"capabilities"`
			}{
				Capabilities: testSpecs,
			},
		}

		respBody, err := json.Marshal(&UpdateVolumeTypeResults{VolumeType: resp})
		c.Assert(err, gc.IsNil)

		w.(*responseWriter).Response.StatusCode = 200
		w.(*responseWriter).Body = ioutil.NopCloser(bytes.NewBuffer(respBody))
	})

	resp, err := s.client.UpdateVolumeTypeExtraSpecs(testId, testType, testSpecs)
	c.Assert(numCalls, gc.Equals, 1)
	c.Assert(err, gc.IsNil)

	c.Check(resp.VolumeType.ExtraSpecs.Capabilities, gc.Equals, testSpecs)
	c.Check(resp.VolumeType.ID, gc.Equals, testId)
	c.Check(resp.VolumeType.Name, gc.Equals, testName)
}

func (s *CinderTestSuite) TestGetVolumesDetail(c *gc.C) {

	numCalls := 0
	s.HandleFunc("/v2/"+testId+"/volumes/", func(w http.ResponseWriter, req *http.Request) {
		numCalls++

		resp := []Volume{{
			AvailabilityZone: "test-availability-zone",
			CreatedAt:        testTime,
			Description:      testDescr,
			ID:               testId,
			Name:             testName,
			Os_Vol_Host_Attr_Host:       "test-host",
			Os_Vol_Tenant_Attr_TenantID: testId,
			Size:        1,
			SnapshotID:  testId,
			SourceVolid: testId,
			Status:      testStatus,
			VolumeType:  "test-volume-type",
		}}

		respBody, err := json.Marshal(&GetVolumesDetailResults{Volumes: resp})
		c.Assert(err, gc.IsNil)

		w.(*responseWriter).Response.StatusCode = 200
		w.(*responseWriter).Body = ioutil.NopCloser(bytes.NewBuffer(respBody))
	})

	resp, err := s.client.GetVolumesDetail()
	c.Assert(numCalls, gc.Equals, 1)
	c.Assert(err, gc.IsNil)
	c.Assert(resp.Volumes, gc.HasLen, 1)

	volume := resp.Volumes[0]

	c.Check(volume.AvailabilityZone, gc.Equals, "test-availability-zone")
	c.Check(volume.CreatedAt, gc.Equals, testTime)
	c.Check(volume.Description, gc.Equals, testDescr)
	c.Check(volume.ID, gc.Equals, testId)
	c.Check(volume.Name, gc.Equals, testName)
	c.Check(volume.Os_Vol_Host_Attr_Host, gc.Equals, "test-host")
	c.Check(volume.Os_Vol_Tenant_Attr_TenantID, gc.Equals, testId)
	c.Check(volume.Size, gc.Equals, 1)
	c.Check(volume.SnapshotID, gc.Equals, testId)
	c.Check(volume.SourceVolid, gc.Equals, testId)
	c.Check(volume.Status, gc.Equals, testStatus)
	c.Check(volume.VolumeType, gc.Equals, "test-volume-type")
}

func (s *CinderTestSuite) TestGetVolumesSimple(c *gc.C) {

	numCalls := 0
	s.HandleFunc("/v2/"+testId+"/volumes", func(w http.ResponseWriter, req *http.Request) {
		numCalls++

		resp := []Volume{{
			ID:   testId,
			Name: testName,
		}}

		respBody, err := json.Marshal(&GetVolumesSimpleResults{Volumes: resp})
		c.Assert(err, gc.IsNil)

		w.(*responseWriter).Response.StatusCode = 200
		w.(*responseWriter).Body = ioutil.NopCloser(bytes.NewBuffer(respBody))
	})

	resp, err := s.client.GetVolumesSimple()
	c.Assert(numCalls, gc.Equals, 1)
	c.Assert(err, gc.IsNil)
	c.Assert(resp.Volumes, gc.HasLen, 1)

	volume := resp.Volumes[0]

	c.Check(volume.ID, gc.Equals, testId)
	c.Check(volume.Name, gc.Equals, testName)
}

func (s *CinderTestSuite) TestCreateVolumeType(c *gc.C) {
	origReq := CreateVolumeTypeVolumeTypeParams{
		Name: "test-volume-type",
		ExtraSpecs: CreateVolumeTypeVolumeTypeExtraSpecsParams{
			Capabilities: "gpu",
		},
	}

	numCalls := 0
	s.HandleFunc("/v2/"+testId+"/types", func(w http.ResponseWriter, req *http.Request) {
		numCalls++

		reqBody, err := ioutil.ReadAll(req.Body)
		c.Assert(err, gc.IsNil)

		var receivedReq CreateVolumeTypeParams
		err = json.Unmarshal(reqBody, &receivedReq)
		c.Assert(err, gc.IsNil)

		c.Check(receivedReq.VolumeType, gc.DeepEquals, origReq)

		resp := struct {
			ExtraSpecs struct {
				Capabilities string `json:"capabilities"`
			} `json:"extra_specs"`
			ID   string `json:"id"`
			Name string `json:"name"`
		}{
			ID: "test-id",
			ExtraSpecs: struct {
				Capabilities string `json:"capabilities"`
			}{
				Capabilities: receivedReq.VolumeType.ExtraSpecs.Capabilities,
			},
			Name: receivedReq.VolumeType.Name,
		}

		respBody, err := json.Marshal(&CreateVolumeTypeResults{VolumeType: resp})
		c.Assert(err, gc.IsNil)

		w.(*responseWriter).StatusCode = 200
		w.(*responseWriter).Body = ioutil.NopCloser(bytes.NewBuffer(respBody))
	})

	resp, err := s.client.CreateVolumeType(origReq)
	c.Assert(err, gc.IsNil)
	c.Assert(resp.VolumeType.ID, gc.Not(gc.HasLen), 0)

	c.Check(resp.VolumeType.ExtraSpecs.Capabilities, gc.Equals, origReq.ExtraSpecs.Capabilities)
	c.Check(resp.VolumeType.Name, gc.Equals, origReq.Name)
}

func (s *CinderTestSuite) TestDeleteVolumeType(c *gc.C) {

	numCalls := 0
	s.HandleFunc("/v2/"+testId+"/types/", func(w http.ResponseWriter, req *http.Request) {
		numCalls++

		w.(*responseWriter).StatusCode = 202
		w.(*responseWriter).Body = ioutil.NopCloser(bytes.NewBuffer([]byte{}))
	})

	err := s.client.DeleteVolumeType(testId)
	c.Assert(numCalls, gc.Equals, 1)
	c.Assert(err, gc.IsNil)
}

func (s *CinderTestSuite) TestGetVolumeType(c *gc.C) {

	numCalls := 0
	s.HandleFunc("/v2/"+testId+"/types/"+testId, func(w http.ResponseWriter, req *http.Request) {
		numCalls++

		resp := struct {
			ExtraSpecs struct {
				Capabilities string `json:"capabilities"`
			} `json:"extra_specs"`
			ID   string `json:"id"`
			Name string `json:"name"`
		}{
			ID: testId,
			ExtraSpecs: struct {
				Capabilities string `json:"capabilities"`
			}{
				Capabilities: "test-capability",
			},
			Name: testName,
		}

		respBody, err := json.Marshal(&GetVolumeTypeResults{VolumeType: resp})
		c.Assert(err, gc.IsNil)

		w.(*responseWriter).StatusCode = 200
		w.(*responseWriter).Body = ioutil.NopCloser(bytes.NewBuffer(respBody))
	})

	resp, err := s.client.GetVolumeType(testId)
	c.Assert(numCalls, gc.Equals, 1)
	c.Assert(err, gc.IsNil)
	c.Assert(resp.VolumeType.ID, gc.Not(gc.HasLen), 0)

	c.Check(resp.VolumeType.ExtraSpecs.Capabilities, gc.Equals, "test-capability")
	c.Check(resp.VolumeType.Name, gc.Equals, testName)
}

func (s *CinderTestSuite) TestGetVolumeTypes(c *gc.C) {

	numCalls := 0
	s.HandleFunc("/v2/"+testId+"/types", func(w http.ResponseWriter, req *http.Request) {
		numCalls++

		resp := []VolumeType{{
			ID: testId,
			ExtraSpecs: struct {
				Capabilities string `json:"capabilities"`
			}{
				Capabilities: "test-capability",
			},
			Name: testName,
		}}

		respBody, err := json.Marshal(&GetVolumeTypesResults{VolumeTypes: resp})
		c.Assert(err, gc.IsNil)

		w.(*responseWriter).StatusCode = 200
		w.(*responseWriter).Body = ioutil.NopCloser(bytes.NewBuffer(respBody))
	})

	resp, err := s.client.GetVolumeTypes()
	c.Assert(numCalls, gc.Equals, 1)
	c.Assert(err, gc.IsNil)
	c.Assert(resp.VolumeTypes, gc.HasLen, 1)

	volumeType := resp.VolumeTypes[0]

	c.Check(volumeType.ExtraSpecs.Capabilities, gc.Equals, "test-capability")
	c.Check(volumeType.Name, gc.Equals, testName)
	c.Check(volumeType.ID, gc.Equals, testId)
}

func (s *CinderTestSuite) localDo(req *http.Request) (*http.Response, error) {
	handler, matchedPattern := s.Handler(req)
	if matchedPattern == "" {
		return nil, fmt.Errorf("no test handler registered for %s", req.URL.Path)
	}
	fmt.Println(matchedPattern)

	var response http.Response
	handler.ServeHTTP(&responseWriter{&response}, req)

	return &response, nil
}

type responseWriter struct {
	*http.Response
}

func (w *responseWriter) Header() http.Header {
	return w.Response.Header
}

func (w *responseWriter) Write(data []byte) (int, error) {
	return len(data), w.Response.Write(bytes.NewBuffer(data))
}

func (w *responseWriter) WriteHeader(statusCode int) {
	w.Response.StatusCode = statusCode
}
