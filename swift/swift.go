package swift

import (
	"fmt"
	"launchpad.net/goose/client"
	goosehttp "launchpad.net/goose/http"
	"net/http"
)

// Provide access to the OpenStack Object Storage service.
type SwiftProvider interface {
	CreateContainer(containerName string) (err error)

	DeleteContainer(containerName string) (err error)

	publicObjectURL(containerName, objectName string) (url string, err error)

	HeadObject(containerName, objectName string) (headers http.Header, err error)

	GetObject(containerName, objectName string) (obj []byte, err error)

	DeleteObject(containerName, objectName string) (err error)

	PutObject(containerName, objectName string, data []byte) (err error)
}

type OpenStackSwiftProvider struct {
	client client.Client
}

func NewSwiftProvider(client client.Client) SwiftProvider {
	s := &OpenStackSwiftProvider{client}
	return s
}

func (s *OpenStackSwiftProvider) CreateContainer(containerName string) (err error) {

	// Juju expects there to be a (semi) public url for some objects. This
	// could probably be more restrictive or placed in a seperate container
	// with some refactoring, but for now just make everything public.
	headers := make(http.Header)
	headers.Add("X-Container-Read", ".r:*")
	url := fmt.Sprintf("/%s", containerName)
	requestData := goosehttp.RequestData{ReqHeaders: headers, ExpectedStatus: []int{http.StatusAccepted, http.StatusCreated}}
	err = s.client.SendRequest(client.PUT, "object-store", url, &requestData,
		"failed to create container %s.", containerName)
	return
}

func (s *OpenStackSwiftProvider) DeleteContainer(containerName string) (err error) {

	url := fmt.Sprintf("/%s", containerName)
	requestData := goosehttp.RequestData{ExpectedStatus: []int{http.StatusNoContent}}
	err = s.client.SendRequest(client.DELETE, "object-store", url, &requestData,
		"failed to delete container %s.", containerName)
	return
}

func (s *OpenStackSwiftProvider) publicObjectURL(containerName, objectName string) (url string, err error) {
	path := fmt.Sprintf("/%s/%s", containerName, objectName)
	url, err = s.client.MakeServiceURL("object-store", []string{path})
	return
}

func (s *OpenStackSwiftProvider) HeadObject(containerName, objectName string) (headers http.Header, err error) {
	path := fmt.Sprintf("/%s/%s", containerName, objectName)
	requestData := goosehttp.RequestData{ReqHeaders: headers, ExpectedStatus: []int{http.StatusOK}}
	err = s.client.SendRequest(client.HEAD, "object-store", path, &requestData,
		"failed to HEAD object %s from container %s", objectName, containerName)
	return headers, err
}

func (s *OpenStackSwiftProvider) GetObject(containerName, objectName string) (obj []byte, err error) {
	path := fmt.Sprintf("/%s/%s", containerName, objectName)
	requestData := goosehttp.RequestData{RespData: &obj, ExpectedStatus: []int{http.StatusOK}}
	err = s.client.SendRequest(client.GET, "object-store", path, &requestData,
		"failed to GET object %s content from container %s", objectName, containerName)
	return obj, err
}

func (s *OpenStackSwiftProvider) DeleteObject(containerName, objectName string) (err error) {
	path := fmt.Sprintf("/%s/%s", containerName, objectName)
	requestData := goosehttp.RequestData{ExpectedStatus: []int{http.StatusNoContent}}
	err = s.client.SendRequest(client.DELETE, "object-store", path, &requestData,
		"failed to DELETE object %s content from container %s", objectName, containerName)
	return err
}

func (s *OpenStackSwiftProvider) PutObject(containerName, objectName string, data []byte) (err error) {
	path := fmt.Sprintf("/%s/%s", containerName, objectName)
	requestData := goosehttp.RequestData{ReqData: data, ExpectedStatus: []int{http.StatusCreated}}
	err = s.client.SendRequest(client.PUT, "object-store", path, &requestData,
		"failed to PUT object %s content from container %s", objectName, containerName)
	return err
}
