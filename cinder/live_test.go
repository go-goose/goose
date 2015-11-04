// Copyright 2015 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

package cinder

import (
	"log"
	"net/http"
	"net/url"
	"time"

	gc "gopkg.in/check.v1"
	"gopkg.in/goose.v1/client"
	"gopkg.in/goose.v1/identity"
)

var _ = gc.Suite(&liveCinderSuite{})

type liveCinderSuite struct {
	client *Client
}

func (s *liveCinderSuite) SetUpSuite(c *gc.C) {
	if *live == false {
		return
	}

	cred, err := identity.CompleteCredentialsFromEnv()
	if err != nil {
		log.Fatalf("error retrieving credentials from the environment: %v", err)
	}

	authClient := client.NewClient(cred, identity.AuthUserPass, nil)
	if err = authClient.Authenticate(); err != nil {
		log.Fatalf("error authenticating: %v", err)
	}
	endpoint := authClient.EndpointsForRegion(cred.Region)["volume"]
	endpointUrl, err := url.Parse(endpoint)
	if err != nil {
		log.Fatalf("error parsing endpoint: %v", err)
	}

	handleRequest := SetAuthHeaderFn(authClient.Token,
		func(req *http.Request) (*http.Response, error) {
			log.Printf("Method: %v", req.Method)
			log.Printf("URL: %v", req.URL)
			log.Printf("req.Headers: %v", req.Header)
			log.Printf("req.Body: %d", req.ContentLength)
			return http.DefaultClient.Do(req)
		})

	s.client = NewClient(authClient.TenantId(), endpointUrl, handleRequest)
}

func (s *liveCinderSuite) SetUpTest(c *gc.C) {

	if *live == false {
		c.Skip("Not running live tests.")
	}
}

func (s *liveCinderSuite) TestVolumesAndSnapshots(c *gc.C) {

	volInfo, err := s.client.CreateVolume(CreateVolumeVolumeParams{Size: 1})
	c.Assert(err, gc.IsNil)
	defer func() {
		err := s.client.DeleteVolume(volInfo.Volume.ID)
		c.Assert(err, gc.IsNil)
	}()

	err = <-s.client.VolumeStatusNotifier(volInfo.Volume.ID, "available", 10, 1*time.Second)
	c.Assert(err, gc.IsNil)

	snapInfo, err := s.client.CreateSnapshot(CreateSnapshotSnapshotParams{VolumeId: volInfo.Volume.ID})
	c.Assert(err, gc.IsNil)

	c.Check(snapInfo.Snapshot.VolumeID, gc.Equals, volInfo.Volume.ID)

	knownSnapInfo, err := s.client.GetSnapshot(snapInfo.Snapshot.ID)
	c.Assert(err, gc.IsNil)

	c.Check(knownSnapInfo.Snapshot.ID, gc.Equals, snapInfo.Snapshot.ID)

	// Wait for snapshot to be available (or error?) before deleting.
	err = <-s.client.SnapshotStatusNotifier(snapInfo.Snapshot.ID, "available", 10, 1*time.Second)
	c.Check(err, gc.IsNil)

	err = s.client.DeleteSnapshot(snapInfo.Snapshot.ID)
	c.Assert(err, gc.IsNil)

	// Wait for the snapshot to be deleted so that the volume can be deleted.
	<-s.client.SnapshotStatusNotifier(snapInfo.Snapshot.ID, "deleted", 10, 1*time.Second)
}

func (s *liveCinderSuite) TestVolumeTypeOperations(c *gc.C) {

	typeInfo, err := s.client.CreateVolumeType(CreateVolumeTypeVolumeTypeParams{
		Name: "number-monster",
		ExtraSpecs: CreateVolumeTypeVolumeTypeExtraSpecsParams{
			Capabilities: "gpu",
		},
	})
	c.Assert(err, gc.IsNil)

	knownTypeInfo, err := s.client.GetVolumeType(typeInfo.VolumeType.ID)
	c.Assert(err, gc.IsNil)
	c.Check(knownTypeInfo.VolumeType.ID, gc.Equals, typeInfo.VolumeType.ID)

	err = s.client.DeleteVolumeType(typeInfo.VolumeType.ID)
	c.Assert(err, gc.IsNil)
}

func (s *liveCinderSuite) TestVolumeMetadata(c *gc.C) {

	metadata := map[string]string{
		"a": "b",
		"c": "d",
	}
	volInfo, err := s.client.CreateVolume(CreateVolumeVolumeParams{
		Size:     1,
		Metadata: metadata,
	})
	c.Assert(err, gc.IsNil)
	defer func() {
		err := s.client.DeleteVolume(volInfo.Volume.ID)
		c.Assert(err, gc.IsNil)
	}()

	err = <-s.client.VolumeStatusNotifier(volInfo.Volume.ID, "available", 10, 1*time.Second)
	c.Assert(err, gc.IsNil)

	result, err := s.client.GetVolume(volInfo.Volume.ID)
	c.Assert(err, gc.IsNil)
	c.Assert(result, gc.NotNil)
	c.Assert(result.Volume.Metadata, gc.DeepEquals, metadata)
}

func (s *liveCinderSuite) TestListVersions(c *gc.C) {
	result, err := s.client.ListVersions()
	c.Assert(err, gc.IsNil)
	c.Assert(result, gc.NotNil)

	c.Logf("versions: %#v", result.Versions)
	c.Assert(len(result.Versions), gc.Not(gc.Equals), 0)
}
