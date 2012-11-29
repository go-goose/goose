package swift_test

import (
	. "launchpad.net/gocheck"
	"launchpad.net/goose/client"
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
	s.client = client.NewClient(s.cred, identity.AuthUserPass)
	s.swift = swift.New(s.client)
}

func (s *LiveTests) TearDownSuite(c *C) {
	// noop, called by local test suite.
}

func (s *LiveTests) SetUpTest(c *C) {
	if !s.client.IsAuthenticated() {
		err := s.client.Authenticate()
		c.Assert(err, IsNil)
	}
}

func (s *LiveTests) TearDownTest(c *C) {
	// noop, called by local test suite.
}

func (s *LiveTests) TestCreateAndDeleteContainer(c *C) {
	container := "test_container"
	err := s.swift.CreateContainer(container)
	c.Check(err, IsNil)
	err = s.swift.DeleteContainer(container)
	c.Check(err, IsNil)
}

func (s *LiveTests) TestObjects(c *C) {

	container := "test_container"
	object := "test_obj"
	data := "...some data..."
	err := s.swift.CreateContainer(container)
	c.Check(err, IsNil)
	err = s.swift.PutObject(container, object, []byte(data))
	c.Check(err, IsNil)
	objdata, err := s.swift.GetObject(container, object)
	c.Check(err, IsNil)
	c.Check(string(objdata), Equals, data)
	err = s.swift.DeleteObject(container, object)
	c.Check(err, IsNil)
	err = s.swift.DeleteContainer(container)
	c.Check(err, IsNil)
}
