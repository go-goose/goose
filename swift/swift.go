// goose/swift - Go package to interact with OpenStack Object-Storage (Swift) API.
// See http://docs.openstack.org/api/openstack-object-storage/1.0/content/.

package swift

import (
	"bytes"
	goerrors "errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"

	"gopkg.in/goose.v2/client"
	"gopkg.in/goose.v2/errors"
	goosehttp "gopkg.in/goose.v2/http"
)

// Client provides a means to access the OpenStack Object Storage Service.
type Client struct {
	client client.Client
}

// New creates a new Client.
func New(client client.Client) *Client {
	return &Client{client}
}

type ACL string

const (
	Private    = ACL("")
	PublicRead = ACL(".r:*,.rlistings")
)

// CreateContainer creates a container with the given name.
func (c *Client) CreateContainer(containerName string, acl ACL) error {
	// [sodre]: Due to a possible bug in ceph-radosgw, we need to split the
	// creation of the bucket and the changing its ACL.
	requestData := goosehttp.RequestData{ExpectedStatus: []int{http.StatusAccepted, http.StatusCreated}}
	err := c.client.SendRequest(client.PUT, "object-store", "", containerName, &requestData)
	if err != nil {
		err = maybeNotFound(err, "failed to create container: %s", containerName)
		return err
	}
	// Normally accessing a container or objects within requires a token
	// for the tenant. Setting an ACL using the X-Container-Read header
	// can be used to allow unauthenticated HTTP access.
	headers := make(http.Header)
	headers.Add("X-Container-Read", string(acl))
	requestData = goosehttp.RequestData{ReqHeaders: headers,
		ExpectedStatus: []int{http.StatusAccepted, http.StatusNoContent}}
	err = c.client.SendRequest(client.POST, "object-store", "", containerName, &requestData)
	if err != nil {
		err = maybeNotFound(err, "failed to update container read acl: %s", containerName)
	}
	return err
}

// DeleteContainer deletes the specified container.
func (c *Client) DeleteContainer(containerName string) error {
	requestData := goosehttp.RequestData{ExpectedStatus: []int{http.StatusNoContent}}
	err := c.client.SendRequest(client.DELETE, "object-store", "", containerName, &requestData)
	if err != nil {
		err = maybeNotFound(err, "failed to delete container: %s", containerName)
	}
	return err
}

func (c *Client) touchObject(requestData *goosehttp.RequestData, op, containerName, objectName string) error {
	path := fmt.Sprintf("%s/%s", containerName, objectName)
	err := c.client.SendRequest(op, "object-store", "", path, requestData)
	if err != nil {
		err = maybeNotFound(err, "failed to %s object %s from container %s", op, objectName, containerName)
	}
	return err
}

// HeadObject retrieves object metadata and other standard HTTP headers.
func (c *Client) HeadObject(containerName, objectName string) (http.Header, error) {
	requestData := goosehttp.RequestData{}
	err := c.touchObject(&requestData, client.HEAD, containerName, objectName)
	return requestData.RespHeaders, err
}

// GetObject retrieves the specified object's data.
func (c *Client) GetObject(containerName, objectName string) (obj []byte, err error) {
	rc, _, err := c.GetReader(containerName, objectName)
	if err != nil {
		return
	}
	defer rc.Close()
	return ioutil.ReadAll(rc)
}

// The following defines a ReadCloser implementation which reads no data.
// It is used instead of returning a nil pointer, which is the same as http.Request.Body.
var emptyReadCloser noData

type noData struct {
	io.ReadCloser
}

// GetObject retrieves the specified object's data.
func (c *Client) GetReader(containerName, objectName string) (io.ReadCloser, http.Header, error) {
	requestData := goosehttp.RequestData{RespReader: &emptyReadCloser}
	err := c.touchObject(&requestData, client.GET, containerName, objectName)
	return requestData.RespReader, requestData.RespHeaders, err
}

// GetReadSeeker creates a read seeker closer to the specified object's data.
func (c *Client) GetReadSeeker(containerName, objectName string) (io.ReadSeeker, http.Header, error) {
	requestData := goosehttp.RequestData{}
	err := c.touchObject(&requestData, client.HEAD, containerName, objectName)
	if err != nil {
		return nil, nil, err
	}
	o := &oreadseeker{c, containerName, objectName, requestData.RespLength, 0}
	return o, requestData.RespHeaders, err
}

// DeleteObject removes an object from the storage system permanently.
func (c *Client) DeleteObject(containerName, objectName string) error {
	requestData := goosehttp.RequestData{ExpectedStatus: []int{http.StatusNoContent}}
	err := c.touchObject(&requestData, client.DELETE, containerName, objectName)
	return err
}

// PutObject writes, or overwrites, an object's content and metadata.
func (c *Client) PutObject(containerName, objectName string, data []byte) error {
	r := bytes.NewReader(data)
	return c.PutReader(containerName, objectName, r, int64(len(data)))
}

// PutReader writes, or overwrites, an object's content and metadata.
func (c *Client) PutReader(containerName, objectName string, r io.Reader, length int64) error {
	requestData := goosehttp.RequestData{ReqReader: r, ReqLength: int(length), ExpectedStatus: []int{http.StatusCreated}}
	err := c.touchObject(&requestData, client.PUT, containerName, objectName)
	return err
}

// ContainerContents describes a single container and its contents.
type ContainerContents struct {
	Name         string `json:"name"`
	Hash         string `json:"hash"`
	LengthBytes  int    `json:"bytes"`
	ContentType  string `json:"content_type"`
	LastModified string `json:"last_modified"`
}

// GetObject retrieves the specified object's data.
func (c *Client) List(containerName, prefix, delim, marker string, limit int) (contents []ContainerContents, err error) {
	params := make(url.Values)
	params.Add("prefix", prefix)
	params.Add("delimiter", delim)
	params.Add("marker", marker)
	params.Add("format", "json")
	if limit > 0 {
		params.Add("limit", fmt.Sprintf("%d", limit))
	}

	requestData := goosehttp.RequestData{Params: &params, RespValue: &contents}
	err = c.client.SendRequest(client.GET, "object-store", "", containerName, &requestData)
	if err != nil {
		err = errors.Newf(err, "failed to list contents of container: %s", containerName)
	}
	return
}

// URL returns a non-signed URL that allows retrieving the object at path.
// It only works if the object is publicly readable (see SignedURL).
func (c *Client) URL(containerName, file string) (string, error) {
	return c.client.MakeServiceURL("object-store", "", []string{containerName, file})
}

// SignedURL returns a signed URL that allows anyone holding the URL
// to retrieve the object at path. The signature is valid until expires.
func (c *Client) SignedURL(containerName, file string, expires time.Time) (string, error) {
	// expiresUnix := expires.Unix()
	// TODO(wallyworld) - 2013-02-11 bug=1121677
	// retrieve the signed URL, for now just return the public URL
	rawURL, err := c.URL(containerName, file)
	if err != nil {
		return "", err
	}
	return rawURL, nil
}

func maybeNotFound(err error, format string, arg ...interface{}) error {
	if !errors.IsNotFound(err) {
		if error, ok := err.(*goosehttp.HttpError); ok {
			// The OpenStack API says that attempts to operate on non existent containers or objects return a status code
			// of 412 (StatusPreconditionFailed).
			if error.StatusCode == http.StatusPreconditionFailed {
				err = errors.NewNotFoundf(err, "", format, arg...)
			}
		}
	}
	return errors.Newf(err, format, arg...)
}

type oreadseeker struct {
	c             *Client
	containerName string
	objectName    string
	length        int64
	pos           int64
}

// Read reads up to len(b) bytes from the oreadseeker. It returns the number of
// bytes read and any error encountered. At end of stream, Read returns no EOF
// on the first EOF and EOF after that.
func (o *oreadseeker) Read(p []byte) (n int, err error) {
	if o.pos == o.length {
		return 0, io.EOF
	}
	end := int64(len(p)) + o.pos
	header := make(http.Header)
	header.Add("Range", fmt.Sprintf("bytes=%d-%d", o.pos, end))
	requestData := goosehttp.RequestData{
		RespReader:     &emptyReadCloser,
		ReqHeaders:     header,
		ExpectedStatus: []int{http.StatusOK, http.StatusPartialContent},
	}
	err = o.c.touchObject(&requestData, client.GET, o.containerName, o.objectName)
	if err != nil {
		return
	}
	defer requestData.RespReader.Close()
	readSize := 0
	for n < len(p) {
		readSize, err = requestData.RespReader.Read(p[n:])
		n += readSize
		o.pos += int64(readSize)
		if err != nil {
			// Eat the ErrUnexpectedEOF because net/http response turns any EOF into
			// ErrUnexpectedEOF if it happened when net/http didn't expect it.
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				// Some code assumes certain semantics about when EOF is returned.
				err = nil
				// Set pos to length, this should be noop but its possible something lies.
				o.pos = o.length
			}
			return
		}
	}
	return
}

// Seek sets the offset for the next Read operation on the readseeker.
func (o *oreadseeker) Seek(offset int64, whence int) (int64, error) {
	switch whence {
	default:
		return 0, goerrors.New("Seek: invalid whence")
	case io.SeekStart:
		break
	case io.SeekCurrent:
		offset += o.pos
	case io.SeekEnd:
		offset += o.length
	}
	if offset < 0 {
		return 0, goerrors.New("Seek: invalid offset")
	}
	o.pos = offset
	return o.pos, nil
}

func (o *oreadseeker) Close() error {
	return nil
}
