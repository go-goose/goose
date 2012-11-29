package swift

import (
	"fmt"
	"launchpad.net/goose/client"
	goosehttp "launchpad.net/goose/http"
	"net/http"
)

type Client struct {
	client client.Client
}

func New(client client.Client) *Client {
	return &Client{client}
}

func (c *Client) CreateContainer(containerName string) (error) {
	// Juju expects there to be a (semi) public url for some objects. This
	// could probably be more restrictive or placed in a seperate container
	// with some refactoring, but for now just make everything public.
	headers := make(http.Header)
	headers.Add("X-Container-Read", ".r:*")
	url := fmt.Sprintf("/%s", containerName)
	requestData := goosehttp.RequestData{ReqHeaders: headers, ExpectedStatus: []int{http.StatusAccepted, http.StatusCreated}}
	err := c.client.SendRequest(client.PUT, "object-store", url, &requestData,
		"failed to create container %s.", containerName)
	return err
}

func (c *Client) DeleteContainer(containerName string) (error) {
	url := fmt.Sprintf("/%s", containerName)
	requestData := goosehttp.RequestData{ExpectedStatus: []int{http.StatusNoContent}}
	err := c.client.SendRequest(client.DELETE, "object-store", url, &requestData,
		"failed to delete container %s.", containerName)
	return err
}

func (c *Client) touchObject(requestData *goosehttp.RequestData, op, containerName, objectName string) (error) {
	path := fmt.Sprintf("/%s/%s", containerName, objectName)
	err := c.client.SendRequest(op, "object-store", path, requestData,
		"failed to %s object %s from container %s", op, objectName, containerName)
	return err
}

func (c *Client) HeadObject(containerName, objectName string) (headers http.Header, error) {
	requestData := goosehttp.RequestData{ReqHeaders: headers}
	err := c.touchObject(&requestData, client.HEAD, containerName, objectName)
	return headers, err
}

func (c *Client) GetObject(containerName, objectName string) (obj []byte, error) {
	requestData := goosehttp.RequestData{RespData: &obj}
	err := c.touchObject(&requestData, client.GET, containerName, objectName)
	return obj, err
}

func (c *Client) DeleteObject(containerName, objectName string) (error) {
	requestData := goosehttp.RequestData{ExpectedStatus: []int{http.StatusNoContent}}
	err := c.touchObject(&requestData, client.DELETE, containerName, objectName)
	return err
}

func (c *Client) PutObject(containerName, objectName string, data []byte) (error) {
	requestData := goosehttp.RequestData{ReqData: data, ExpectedStatus: []int{http.StatusCreated}}
	err := c.touchObject(&requestData, client.PUT, containerName, objectName)
	return err
}
