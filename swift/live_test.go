package swift_test

import (
	. "launchpad.net/gocheck"
	"launchpad.net/goose/client"
	gooseerrors "launchpad.net/goose/errors"
	"launchpad.net/goose/identity"
	"launchpad.net/goose/swift"
)

func registerOpenStackTests(cred *identity.Credentials) {
	Suite(&LiveTests{
		cred: cred,
	})
}

type LiveTests struct {
	cred   *identity.Credentials
	client *client.OpenStackClient
	swift  *swift.Client
}

func (s *LiveTests) SetUpSuite(c *C) {
	s.client = client.NewClient(s.cred, identity.AuthUserPass, nil)
	s.swift = swift.New(s.client)
}

func (s *LiveTests) TearDownSuite(c *C) {
	// noop, called by local test suite.
}

func (s *LiveTests) SetUpTest(c *C) {
	// noop, called by local test suite.
}

func (s *LiveTests) TearDownTest(c *C) {
	// noop, called by local test suite.
}

func (s *LiveTests) assertCreateContainer(c *C, container string) {
	// The test container may exist already, so try and delete it.
	// If the result is a NotFound error, we don't care.
	err := s.swift.DeleteContainer(container)
	if err != nil {
		c.Check(gooseerrors.IsNotFound(err), Equals, true)
	}
	err = s.swift.CreateContainer(container)
	c.Assert(err, IsNil)
}

func (s *LiveTests) TestCreateAndDeleteContainer(c *C) {
	container := "test_container"
	s.assertCreateContainer(c, container)
	err := s.swift.DeleteContainer(container)
	c.Assert(err, IsNil)
}

func (s *LiveTests) TestObjects(c *C) {
	container := "test_container"
	s.assertCreateContainer(c, container)
	object := "test_obj"
	data := "...some data..."
	err := s.swift.PutObject(container, object, []byte(data))
	c.Check(err, IsNil)
	objdata, err := s.swift.GetObject(container, object)
	c.Check(err, IsNil)
	c.Check(string(objdata), Equals, data)
	err = s.swift.DeleteObject(container, object)
	c.Check(err, IsNil)
	err = s.swift.DeleteContainer(container)
	c.Assert(err, IsNil)
}

func (s *LiveTests) TestMissingContainer(c *C) {
	object := "test_obj"
//	data := "...some data..."
	_, err := s.swift.GetObject("nonexistantcontainer", object)
	c.Check(err, IsNil)
}

