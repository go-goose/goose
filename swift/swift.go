// The swift package provides a way to access the OpenStack Object Storage APIs.
// See http://docs.openstack.org/api/openstack-object-storage/1.0/content/.
package swift

import (
	"fmt"
	"launchpad.net/goose/client"
	"launchpad.net/goose/errors"
	goosehttp "launchpad.net/goose/http"
	"net/http"
)

// Client provides a means to access the OpenStack Object Storage Service.
type Client struct {
	client client.Client
}

func New(client client.Client) *Client {
	return &Client{client}
}

// CreateContainer creates a container with the given name.
func (c *Client) CreateContainer(containerName string) error {
	// Juju expects there to be a (semi) public url for some objects. This
	// could probably be more restrictive or placed in a seperate container
	// with some refactoring, but for now just make everything public.
	headers := make(http.Header)
	headers.Add("X-Container-Read", ".r:*")
	url := fmt.Sprintf("/%s", containerName)
	requestData := goosehttp.RequestData{ReqHeaders: headers, ExpectedStatus: []int{http.StatusAccepted, http.StatusCreated}}
	err := c.client.SendRequest(client.PUT, "object-store", url, &requestData)
	if err != nil {
		err = errors.Newf(err, "failed to create container: %s", containerName)
	}
	return err
}

// DeleteContainer deletes the specified container.
func (c *Client) DeleteContainer(containerName string) error {
	url := fmt.Sprintf("/%s", containerName)
	requestData := goosehttp.RequestData{ExpectedStatus: []int{http.StatusNoContent}}
	err := c.client.SendRequest(client.DELETE, "object-store", url, &requestData)
	if err != nil {
		err = errors.Newf(err, "failed to delete container: %s", containerName)
	}
	return err
}

func (c *Client) touchObject(requestData *goosehttp.RequestData, op, containerName, objectName string) error {
	path := fmt.Sprintf("/%s/%s", containerName, objectName)
	err := c.client.SendRequest(op, "object-store", path, requestData)
	if err != nil {
		err = errors.Newf(err, "failed to %s object %s from container %s", op, objectName, containerName)
	}
	return err
}

// HeadObject retrieves object metadata and other standard HTTP headers.
func (c *Client) HeadObject(containerName, objectName string) (headers http.Header, err error) {
	requestData := goosehttp.RequestData{ReqHeaders: headers}
	err = c.touchObject(&requestData, client.HEAD, containerName, objectName)
	return headers, err
}

// GetObject retrieves the specified object's data.
func (c *Client) GetObject(containerName, objectName string) (obj []byte, err error) {
	requestData := goosehttp.RequestData{RespData: &obj}
	err = c.touchObject(&requestData, client.GET, containerName, objectName)
	return obj, err
}

// DeleteObject removes an object from the storage system permanently.
func (c *Client) DeleteObject(containerName, objectName string) error {
	requestData := goosehttp.RequestData{ExpectedStatus: []int{http.StatusNoContent}}
	err := c.touchObject(&requestData, client.DELETE, containerName, objectName)
	return err
}

// PutObject writes, or overwrites, an object's content and metadata.
func (c *Client) PutObject(containerName, objectName string, data []byte) error {
	requestData := goosehttp.RequestData{ReqData: data, ExpectedStatus: []int{http.StatusCreated}}
	err := c.touchObject(&requestData, client.PUT, containerName, objectName)
	return err
}
