// Swift double testing service - mimics OpenStack Swift object
// storage service for testing goose against close-to-live API.

package swiftservice

import (
	"launchpad.net/goose/swift"
	"net/http"
)

// SwiftService presents an direct-API to manipulate the internal
// state, as well as an HTTP API double for OpenStack Swift.
type SwiftService interface {
	// AddContainer creates a new container with the given name.
	AddContainer(name string) error

	// AddObject creates a new named object in an existing container.
	AddObject(container, name string, data []byte) error

	// HasContainer verifies the given container exists or not.
	HasContainer(name string) bool

	// ListContainer lists the objects in the given container.
	ListContainer(name string) ([]swift.ContainerContents, error)

	// GetObject retrieves a given object's data from its container.
	GetObject(container, name string) ([]byte, error)

	// RemoveContainer deletes an existing named container.
	RemoveContainer(name string) error

	// RemoveObject deletes an existing named object, from its container.
	RemoveObject(container, name string) error

	// GetURL returns the named object's full public URL to get its data.
	GetURL(container, object string) (string, error)

	// ServeHTTP is the main entry point in the HTTP request processing.
	ServeHTTP(w http.ResponseWriter, r *http.Request)
}
