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

func (s *OpenStackSwiftProvider) touchObject(requestData *goosehttp.RequestData, op, containerName, objectName string) (err error) {
	path := fmt.Sprintf("/%s/%s", containerName, objectName)
	err = s.client.SendRequest(op, "object-store", path, requestData,
		"failed to %s object %s from container %s", op, objectName, containerName)
	return
}

func (s *OpenStackSwiftProvider) HeadObject(containerName, objectName string) (headers http.Header, err error) {
	requestData := goosehttp.RequestData{ReqHeaders: headers}
	err = s.touchObject(&requestData, client.HEAD, containerName, objectName)
	return headers, err
}

func (s *OpenStackSwiftProvider) GetObject(containerName, objectName string) (obj []byte, err error) {
	requestData := goosehttp.RequestData{RespData: &obj}
	err = s.touchObject(&requestData, client.GET, containerName, objectName)
	return obj, err
}

func (s *OpenStackSwiftProvider) DeleteObject(containerName, objectName string) (err error) {
	requestData := goosehttp.RequestData{ExpectedStatus: []int{http.StatusNoContent}}
	err = s.touchObject(&requestData, client.DELETE, containerName, objectName)
	return
}

func (s *OpenStackSwiftProvider) PutObject(containerName, objectName string, data []byte) (err error) {
	requestData := goosehttp.RequestData{ReqData: data, ExpectedStatus: []int{http.StatusCreated}}
	err = s.touchObject(&requestData, client.PUT, containerName, objectName)
	return
}
