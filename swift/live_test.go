package swift_test

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	gc "gopkg.in/check.v1"

	"gopkg.in/goose.v1/client"
	"gopkg.in/goose.v1/errors"
	"gopkg.in/goose.v1/identity"
	"gopkg.in/goose.v1/swift"
)

func registerOpenStackTests(cred *identity.Credentials) {
	gc.Suite(&LiveTests{
		cred: cred,
	})
	gc.Suite(&LiveTestsPublicContainer{
		cred: cred,
	})
}

func randomName() string {
	buf := make([]byte, 8)
	_, err := io.ReadFull(rand.Reader, buf)
	if err != nil {
		panic(fmt.Sprintf("error from crypto rand: %v", err))
	}
	return fmt.Sprintf("%x", buf)
}

type LiveTests struct {
	cred          *identity.Credentials
	client        client.AuthenticatingClient
	swift         *swift.Client
	containerName string
}

func (s *LiveTests) SetUpSuite(c *gc.C) {
	s.containerName = "test_container" + randomName()
	s.client = client.NewClient(s.cred, identity.AuthUserPass, nil)
	s.swift = swift.New(s.client)
}

func (s *LiveTests) TearDownSuite(c *gc.C) {
	// noop, called by local test suite.
}

func (s *LiveTests) SetUpTest(c *gc.C) {
	assertCreateContainer(c, s.containerName, s.swift, swift.Private)
}

func (s *LiveTests) TearDownTest(c *gc.C) {
	err := s.swift.DeleteContainer(s.containerName)
	c.Check(err, gc.IsNil)
}

func assertCreateContainer(c *gc.C, container string, s *swift.Client, acl swift.ACL) {
	// The test container may exist already, so try and delete it.
	// If the result is a NotFound error, we don't care.
	err := s.DeleteContainer(container)
	if err != nil {
		c.Check(errors.IsNotFound(err), gc.Equals, true)
	}
	err = s.CreateContainer(container, acl)
	c.Assert(err, gc.IsNil)
}

func (s *LiveTests) TestObject(c *gc.C) {
	object := "test_obj1"
	data := "...some data..."
	err := s.swift.PutObject(s.containerName, object, []byte(data))
	c.Check(err, gc.IsNil)
	objdata, err := s.swift.GetObject(s.containerName, object)
	c.Check(err, gc.IsNil)
	c.Check(string(objdata), gc.Equals, data)
	err = s.swift.DeleteObject(s.containerName, object)
	c.Assert(err, gc.IsNil)
}

func (s *LiveTests) TestObjectReader(c *gc.C) {
	object := "test_obj2"
	data := "...some data..."
	err := s.swift.PutReader(s.containerName, object, bytes.NewReader([]byte(data)), int64(len(data)))
	c.Check(err, gc.IsNil)
	r, headers, err := s.swift.GetReader(s.containerName, object)
	c.Check(err, gc.IsNil)
	readData, err := ioutil.ReadAll(r)
	c.Check(err, gc.IsNil)
	r.Close()
	c.Check(string(readData), gc.Equals, data)
	err = s.swift.DeleteObject(s.containerName, object)
	c.Assert(err, gc.IsNil)
	c.Check(headers.Get("Date"), gc.Not(gc.Equals), "")
}

func (s *LiveTests) TestList(c *gc.C) {
	data := "...some data..."
	var files []string = make([]string, 2)
	var fileNames map[string]bool = make(map[string]bool)
	for i := 0; i < 2; i++ {
		files[i] = fmt.Sprintf("test_obj%d", i)
		fileNames[files[i]] = true
		err := s.swift.PutObject(s.containerName, files[i], []byte(data))
		c.Check(err, gc.IsNil)
	}
	items, err := s.swift.List(s.containerName, "", "", "", 0)
	c.Check(err, gc.IsNil)
	c.Check(len(items), gc.Equals, len(files))
	for _, item := range items {
		c.Check(fileNames[item.Name], gc.Equals, true)
	}
	for i := 0; i < len(files); i++ {
		s.swift.DeleteObject(s.containerName, files[i])
	}
}

func (s *LiveTests) TestURL(c *gc.C) {
	object := "test_obj1"
	data := "...some data..."
	err := s.swift.PutObject(s.containerName, object, []byte(data))
	c.Check(err, gc.IsNil)
	url, err := s.swift.URL(s.containerName, object)
	c.Check(err, gc.IsNil)
	httpClient := http.DefaultClient
	req, err := http.NewRequest("GET", url, nil)
	req.Header.Add("X-Auth-Token", s.client.Token())
	c.Check(err, gc.IsNil)
	resp, err := httpClient.Do(req)
	defer resp.Body.Close()
	c.Check(err, gc.IsNil)
	c.Check(resp.StatusCode, gc.Equals, http.StatusOK)
	objdata, err := ioutil.ReadAll(resp.Body)
	c.Check(err, gc.IsNil)
	c.Check(string(objdata), gc.Equals, data)
	err = s.swift.DeleteObject(s.containerName, object)
	c.Assert(err, gc.IsNil)
}

type LiveTestsPublicContainer struct {
	cred          *identity.Credentials
	client        client.AuthenticatingClient
	publicClient  client.Client
	swift         *swift.Client
	publicSwift   *swift.Client
	containerName string
}

func (s *LiveTestsPublicContainer) SetUpSuite(c *gc.C) {
	s.containerName = "test_container" + randomName()
	s.client = client.NewClient(s.cred, identity.AuthUserPass, nil)
	s.swift = swift.New(s.client)
}

func (s *LiveTestsPublicContainer) TearDownSuite(c *gc.C) {
	// noop, called by local test suite.
}

func (s *LiveTestsPublicContainer) SetUpTest(c *gc.C) {
	err := s.client.Authenticate()
	c.Assert(err, gc.IsNil)
	baseURL, err := s.client.MakeServiceURL("object-store", nil)
	c.Assert(err, gc.IsNil)
	s.publicClient = client.NewPublicClient(baseURL, nil)
	s.publicSwift = swift.New(s.publicClient)
	assertCreateContainer(c, s.containerName, s.swift, swift.PublicRead)
}

func (s *LiveTestsPublicContainer) TearDownTest(c *gc.C) {
	err := s.swift.DeleteContainer(s.containerName)
	c.Check(err, gc.IsNil)
}

func (s *LiveTestsPublicContainer) TestPublicObjectReader(c *gc.C) {
	object := "test_obj2"
	data := "...some data..."
	err := s.swift.PutReader(s.containerName, object, bytes.NewReader([]byte(data)), int64(len(data)))
	c.Check(err, gc.IsNil)
	r, headers, err := s.publicSwift.GetReader(s.containerName, object)
	c.Check(err, gc.IsNil)
	readData, err := ioutil.ReadAll(r)
	c.Check(err, gc.IsNil)
	r.Close()
	c.Check(string(readData), gc.Equals, data)
	err = s.swift.DeleteObject(s.containerName, object)
	c.Assert(err, gc.IsNil)
	c.Check(headers.Get("Date"), gc.Not(gc.Equals), "")
}

func (s *LiveTestsPublicContainer) TestPublicList(c *gc.C) {
	data := "...some data..."
	var files []string = make([]string, 2)
	var fileNames map[string]bool = make(map[string]bool)
	for i := 0; i < 2; i++ {
		files[i] = fmt.Sprintf("test_obj%d", i)
		fileNames[files[i]] = true
		err := s.swift.PutObject(s.containerName, files[i], []byte(data))
		c.Check(err, gc.IsNil)
	}
	items, err := s.publicSwift.List(s.containerName, "", "", "", 0)
	c.Check(err, gc.IsNil)
	c.Check(len(items), gc.Equals, len(files))
	for _, item := range items {
		c.Check(fileNames[item.Name], gc.Equals, true)
	}
	for i := 0; i < len(files); i++ {
		s.swift.DeleteObject(s.containerName, files[i])
	}
}

func (s *LiveTestsPublicContainer) TestPublicURL(c *gc.C) {
	object := "test_obj1"
	data := "...some data..."
	err := s.swift.PutObject(s.containerName, object, []byte(data))
	c.Check(err, gc.IsNil)
	url, err := s.swift.URL(s.containerName, object)
	c.Check(err, gc.IsNil)
	httpClient := http.DefaultClient
	req, err := http.NewRequest("GET", url, nil)
	c.Check(err, gc.IsNil)
	resp, err := httpClient.Do(req)
	defer resp.Body.Close()
	c.Check(err, gc.IsNil)
	c.Check(resp.StatusCode, gc.Equals, http.StatusOK)
	objdata, err := ioutil.ReadAll(resp.Body)
	c.Check(err, gc.IsNil)
	c.Check(string(objdata), gc.Equals, data)
	err = s.swift.DeleteObject(s.containerName, object)
	c.Assert(err, gc.IsNil)
}

func (s *LiveTests) TestHeadObject(c *gc.C) {
	object := "test_obj2"
	data := "...some data..."
	err := s.swift.PutReader(s.containerName, object, bytes.NewReader([]byte(data)), int64(len(data)))
	c.Check(err, gc.IsNil)
	headers, err := s.swift.HeadObject(s.containerName, object)
	c.Check(err, gc.IsNil)
	err = s.swift.DeleteObject(s.containerName, object)
	c.Assert(err, gc.IsNil)
	c.Check(headers.Get("Date"), gc.Not(gc.Equals), "")
}
