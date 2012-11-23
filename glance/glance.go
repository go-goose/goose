package glance

import (
	"fmt"
	"launchpad.net/goose/client"
	goosehttp "launchpad.net/goose/http"
	"net/http"
)

const (
	OS_API_IMAGES        = "/images"
	OS_API_IMAGES_DETAIL = "/images/detail"
)

// Provide access to the OpenStack Glance service.
type GlanceProvider interface {
	ListImages() (servers []Image, err error)

	ListImagesDetail() (servers []ImageDetail, err error)

	TestGetImageDetail(imageId string) (ImageDetail, error)
}

type OpenStackGlanceProvider struct {
	client client.Client
}

func NewGlanceProvider(client client.Client) GlanceProvider {
	n := &OpenStackGlanceProvider{client}
	return n
}

type Link struct {
	Href string
	Rel  string
	Type string
}

type Image struct {
	Id    string
	Name  string
	Links []Link
}

func (n *OpenStackGlanceProvider) ListImages() (servers []Image, err error) {

	var resp struct {
		Images []Image
	}
	requestData := goosehttp.RequestData{RespValue: &resp, ExpectedStatus: []int{http.StatusOK}}
	err = n.client.SendRequest(client.GET, "compute", OS_API_IMAGES, &requestData,
		"failed to get list of images")
	return resp.Images, err
}

type ImageMetadata struct {
	Architecture string
	State        string      `json:"image_state"`
	Location     string      `json:"image_location"`
	KernelId     interface{} `json:"kernel_id"`
	ProjectId    interface{} `json:"project_id"`
	RAMDiskId    interface{} `json:"ramdisk_id"`
	OwnerId      interface{} `json:"owner_id"`
}

type ImageDetail struct {
	Id          string
	Name        string
	Created     string
	Updated     string
	Progress    int
	Status      string
	MinimumRAM  int `json:"minRam"`
	MinimumDisk int `json:"minDisk"`
	Links       []Link
	Metadata    ImageMetadata
}

func (n *OpenStackGlanceProvider) ListImagesDetail() (images []ImageDetail, err error) {

	var resp struct {
		Images []ImageDetail
	}
	requestData := goosehttp.RequestData{RespValue: &resp}
	err = n.client.SendRequest(client.GET, "compute", OS_API_IMAGES_DETAIL, &requestData,
		"failed to get list of images details")
	return resp.Images, err
}

func (n *OpenStackGlanceProvider) TestGetImageDetail(imageId string) (ImageDetail, error) {

	var resp struct {
		Image ImageDetail
	}
	url := fmt.Sprintf("%s/%s", OS_API_IMAGES, imageId)
	requestData := goosehttp.RequestData{RespValue: &resp}
	err := n.client.SendRequest(client.GET, "compute", url, &requestData,
		"failed to get details for imageId=%s", imageId)
	return resp.Image, err
}
