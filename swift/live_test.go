package swift_test

import (
	"bytes"
	"io/ioutil"
	. "launchpad.net/gocheck"
	"launchpad.net/goose/client"
	"launchpad.net/goose/errors"
	"launchpad.net/goose/identity"
	"launchpad.net/goose/swift"
)

func registerOpenStackTests(cred *identity.Credentials) {
	Suite(&LiveTests{
		cred: cred,
	})
}

type LiveTests struct {
	cred          *identity.Credentials
	client        *client.OpenStackClient
	swift         *swift.Client
	containerName string
}

func (s *LiveTests) SetUpSuite(c *C) {
	s.client = client.NewClient(s.cred, identity.AuthUserPass, nil)
	s.swift = swift.New(s.client)
}

func (s *LiveTests) TearDownSuite(c *C) {
	// noop, called by local test suite.
}

func (s *LiveTests) SetUpTest(c *C) {
	s.containerName = "test_container"
	s.assertCreateContainer(c, s.containerName)
}

func (s *LiveTests) TearDownTest(c *C) {
	err := s.swift.DeleteContainer(s.containerName)
	c.Check(err, IsNil)
}

func (s *LiveTests) assertCreateContainer(c *C, container string) {
	// The test container may exist already, so try and delete it.
	// If the result is a NotFound error, we don't care.
	err := s.swift.DeleteContainer(container)
	if err != nil {
		c.Check(errors.IsNotFound(err), Equals, true)
	}
	err = s.swift.CreateContainer(container)
	c.Assert(err, IsNil)
}

func (s *LiveTests) TestObject(c *C) {
	object := "test_obj1"
	data := "...some data..."
	err := s.swift.PutObject(s.containerName, object, []byte(data))
	c.Check(err, IsNil)
	objdata, err := s.swift.GetObject(s.containerName, object)
	c.Check(err, IsNil)
	c.Check(string(objdata), Equals, data)
	err = s.swift.DeleteObject(s.containerName, object)
	c.Assert(err, IsNil)
}

func (s *LiveTests) TestObjectReader(c *C) {
	object := "test_obj2"
	data := "...some data..."
	err := s.swift.PutReader(s.containerName, object, bytes.NewReader([]byte(data)))
	c.Check(err, IsNil)
	r, err := s.swift.GetReader(s.containerName, object)
	c.Check(err, IsNil)
	readData, err := ioutil.ReadAll(r)
	c.Check(err, IsNil)
	r.Close()
	c.Check(string(readData), Equals, data)
	err = s.swift.DeleteObject(s.containerName, object)
	c.Assert(err, IsNil)
}
