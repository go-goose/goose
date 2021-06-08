package swift_test

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	gc "gopkg.in/check.v1"

	"gopkg.in/goose.v3/client"
	"gopkg.in/goose.v3/errors"
	"gopkg.in/goose.v3/identity"
	"gopkg.in/goose.v3/swift"
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
	s.client.SetRequiredServiceTypes([]string{"object-store"})
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
		c.Check(errors.IsNotFound(err), gc.Equals, true, gc.Commentf("cannot delete container: %v", err))
	}
	err = s.CreateContainer(container, acl)
	c.Assert(err, gc.IsNil)
}

func (s *LiveTests) checkDeleteObject(c *gc.C, object string) {
	err := s.swift.DeleteObject(s.containerName, object)
	c.Check(err, gc.IsNil)
}

func (s *LiveTests) TestObject(c *gc.C) {
	object := "test_obj1"
	data := "...some data..."
	err := s.swift.PutObject(s.containerName, object, []byte(data))
	c.Assert(err, gc.IsNil)
	defer s.checkDeleteObject(c, object)
	objdata, err := s.swift.GetObject(s.containerName, object)
	c.Check(err, gc.IsNil)
	c.Check(string(objdata), gc.Equals, data)
}

func (s *LiveTests) TestObjectNotFound(c *gc.C) {
	_, err := s.swift.GetObject(s.containerName, "not-there")
	c.Check(err, gc.ErrorMatches, `object "not-there" in container ".*" not found`)
	c.Check(errors.IsNotFound(err), gc.Equals, true)
}

func (s *LiveTests) TestObjectReader(c *gc.C) {
	object := "test_obj2"
	data := "...some data..."
	err := s.swift.PutReader(s.containerName, object, bytes.NewReader([]byte(data)), int64(len(data)))
	c.Assert(err, gc.IsNil)
	defer s.checkDeleteObject(c, object)
	r, headers, err := s.swift.GetReader(s.containerName, object)
	c.Assert(err, gc.IsNil)
	readData, err := ioutil.ReadAll(r)
	c.Check(err, gc.IsNil)
	r.Close()
	c.Check(string(readData), gc.Equals, data)
	c.Check(headers.Get("Date"), gc.Not(gc.Equals), "")
}

func (s *LiveTests) TestList(c *gc.C) {
	data := "...some data..."
	files := make([]string, 2)
	fileNames := make(map[string]bool)
	for i := range files {
		file := fmt.Sprintf("test_obj%d", i)
		files[i] = file
		fileNames[file] = true
		err := s.swift.PutObject(s.containerName, file, []byte(data))
		c.Assert(err, gc.IsNil)
		defer s.checkDeleteObject(c, file)
	}
	items, err := s.swift.List(s.containerName, "", "", "", 0)
	c.Assert(err, gc.IsNil)
	c.Check(len(items), gc.Equals, len(files))
	for _, item := range items {
		c.Check(fileNames[item.Name], gc.Equals, true)
	}
}

func (s *LiveTests) TestURL(c *gc.C) {
	object := "test_obj1"
	data := "...some data..."
	err := s.swift.PutObject(s.containerName, object, []byte(data))
	c.Assert(err, gc.IsNil)
	defer s.checkDeleteObject(c, object)
	url, err := s.swift.URL(s.containerName, object)
	c.Assert(err, gc.IsNil)
	httpClient := http.DefaultClient
	req, err := http.NewRequest("GET", url, nil)
	c.Assert(err, gc.IsNil)
	req.Header.Add("X-Auth-Token", s.client.Token())
	resp, err := httpClient.Do(req)
	defer resp.Body.Close()
	c.Assert(err, gc.IsNil)
	c.Check(resp.StatusCode, gc.Equals, http.StatusOK)
	objdata, err := ioutil.ReadAll(resp.Body)
	c.Check(err, gc.IsNil)
	c.Check(string(objdata), gc.Equals, data)
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
	s.client.SetRequiredServiceTypes([]string{"object-store"})
	s.swift = swift.New(s.client)
}

func (s *LiveTestsPublicContainer) TearDownSuite(c *gc.C) {
	// noop, called by local test suite.
}

func (s *LiveTestsPublicContainer) checkDeleteObject(c *gc.C, object string) {
	err := s.swift.DeleteObject(s.containerName, object)
	c.Check(err, gc.IsNil)
}

func (s *LiveTestsPublicContainer) SetUpTest(c *gc.C) {
	err := s.client.Authenticate()
	c.Assert(err, gc.IsNil)
	baseURL, err := s.client.MakeServiceURL("object-store", "", nil)
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
	c.Assert(err, gc.IsNil)
	defer s.checkDeleteObject(c, object)
	r, headers, err := s.publicSwift.GetReader(s.containerName, object)
	c.Assert(err, gc.IsNil)
	readData, err := ioutil.ReadAll(r)
	c.Check(err, gc.IsNil)
	r.Close()
	c.Check(string(readData), gc.Equals, data)
	c.Check(headers.Get("Date"), gc.Not(gc.Equals), "")
}

func (s *LiveTestsPublicContainer) TestPublicList(c *gc.C) {
	data := "...some data..."
	files := make([]string, 2)
	fileNames := make(map[string]bool)
	for i := range files {
		file := fmt.Sprintf("test_obj%d", i)
		files[i] = file
		fileNames[file] = true
		err := s.swift.PutObject(s.containerName, file, []byte(data))
		c.Assert(err, gc.IsNil)
		defer s.checkDeleteObject(c, file)
	}
	items, err := s.publicSwift.List(s.containerName, "", "", "", 0)
	c.Assert(err, gc.IsNil)
	c.Check(len(items), gc.Equals, len(files))
	for _, item := range items {
		c.Check(fileNames[item.Name], gc.Equals, true)
	}
}

func (s *LiveTestsPublicContainer) TestPublicURL(c *gc.C) {
	object := "test_obj1"
	data := "...some data..."
	err := s.swift.PutObject(s.containerName, object, []byte(data))
	c.Assert(err, gc.IsNil)
	defer s.checkDeleteObject(c, object)
	url, err := s.swift.URL(s.containerName, object)
	c.Assert(err, gc.IsNil)
	httpClient := http.DefaultClient
	req, err := http.NewRequest("GET", url, nil)
	c.Assert(err, gc.IsNil)
	resp, err := httpClient.Do(req)
	defer resp.Body.Close()
	c.Assert(err, gc.IsNil)
	c.Check(resp.StatusCode, gc.Equals, http.StatusOK)
	objdata, err := ioutil.ReadAll(resp.Body)
	c.Check(err, gc.IsNil)
	c.Check(string(objdata), gc.Equals, data)
}

func (s *LiveTests) TestHeadObject(c *gc.C) {
	object := "test_obj2"
	data := "...some data..."
	err := s.swift.PutReader(s.containerName, object, bytes.NewReader([]byte(data)), int64(len(data)))
	c.Assert(err, gc.IsNil)
	defer s.checkDeleteObject(c, object)
	headers, err := s.swift.HeadObject(s.containerName, object)
	c.Assert(err, gc.IsNil)
	c.Check(headers.Get("Date"), gc.Not(gc.Equals), "")
}

func (s *LiveTests) TestOpenObject(c *gc.C) {
	object := "test_obj2"
	data := "...some data..."
	err := s.swift.PutReader(s.containerName, object, bytes.NewReader([]byte(data)), int64(len(data)))
	c.Assert(err, gc.IsNil)
	defer s.checkDeleteObject(c, object)
	r, headers, err := s.swift.OpenObject(s.containerName, object, 0)
	c.Assert(err, gc.IsNil)
	defer r.Close()
	readData, err := ioutil.ReadAll(r)
	c.Assert(err, gc.IsNil)
	c.Check(string(readData), gc.Equals, data)
	c.Check(headers.Get("Date"), gc.Not(gc.Equals), "")
}

func (s *LiveTests) TestOpenObjectSeek(c *gc.C) {
	object := "test_obj2"
	data := "...some data..."
	err := s.swift.PutReader(s.containerName, object, bytes.NewReader([]byte(data)), int64(len(data)))
	c.Assert(err, gc.IsNil)
	defer s.checkDeleteObject(c, object)
	r, headers, err := s.swift.OpenObject(s.containerName, object, 0)
	c.Assert(err, gc.IsNil)
	defer r.Close()
	n, err := r.Seek(3, io.SeekStart)
	c.Check(err, gc.IsNil)
	c.Check(n, gc.Equals, int64(3))
	readData, err := ioutil.ReadAll(r)
	c.Check(err, gc.IsNil)
	c.Check(string(readData), gc.Equals, data[3:])
	c.Check(headers.Get("Date"), gc.Not(gc.Equals), "")
}

func (s *LiveTests) TestOpenObjectReadLimit(c *gc.C) {
	object := "test_obj2"
	data := "...some data..."
	err := s.swift.PutReader(s.containerName, object, bytes.NewReader([]byte(data)), int64(len(data)))
	c.Assert(err, gc.IsNil)
	defer s.checkDeleteObject(c, object)
	r, headers, err := s.swift.OpenObject(s.containerName, object, 0)
	c.Assert(err, gc.IsNil)
	defer r.Close()
	n, err := r.Seek(3, io.SeekStart)
	c.Check(err, gc.IsNil)
	c.Check(n, gc.Equals, int64(3))
	buf := make([]byte, 9)
	n1, err := r.Read(buf)
	c.Check(n1, gc.Equals, 9)
	c.Check(err, gc.IsNil)
	c.Check(string(buf), gc.Equals, data[3:12])
	c.Check(headers.Get("Date"), gc.Not(gc.Equals), "")
}

func (s *LiveTests) TestOpenObjectSeekContract(c *gc.C) {
	object := "test_obj2"
	data := "...some data..."
	err := s.swift.PutReader(s.containerName, object, bytes.NewReader([]byte(data)), int64(len(data)))
	c.Assert(err, gc.IsNil)
	defer s.checkDeleteObject(c, object)
	r, _, err := s.swift.OpenObject(s.containerName, object, 0)
	c.Assert(err, gc.IsNil)
	defer r.Close()
	_, err = r.Seek(-1, io.SeekStart)
	c.Check(err, gc.NotNil)
	_, err = r.Seek(-20, io.SeekEnd)
	c.Check(err, gc.NotNil)
}

func (s *LiveTests) TestOpenObjectFileChangedUnderfoot(c *gc.C) {
	object := "test_obj2"
	err := s.swift.PutObject(s.containerName, object, []byte("...some data..."))
	c.Assert(err, gc.IsNil)
	defer s.checkDeleteObject(c, object)

	// Get a Reader handle on the object but don't read it yet.
	r, _, err := s.swift.OpenObject(s.containerName, object, 0)
	c.Assert(err, gc.Equals, nil)
	defer r.Close()

	// Overwrite the object.
	err = s.swift.PutObject(s.containerName, object, []byte("changed!"))
	c.Assert(err, gc.Equals, nil)

	n, err := r.Read(make([]byte, 20))
	c.Check(n, gc.Equals, 0)
	c.Assert(err, gc.ErrorMatches, `file has changed since it was opened`)
}
