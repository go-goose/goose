package swiftservice

import (
	"net/http"
)

type SwiftService interface {
	AddContainer(name string) error
	AddObject(container, name string, data []byte) error
	HasContainer(name string) bool
	GetObject(container, name string) ([]byte, error)
	RemoveContainer(name string) error
	RemoveObject(container, name string) error
	GetURL(container, object string) (string, error)
	ServeHTTP(w http.ResponseWriter, r *http.Request)
}
