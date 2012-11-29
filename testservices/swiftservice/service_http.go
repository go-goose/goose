// Swift double testing service - HTTP API implementation

package swiftservice

import (
	"io/ioutil"
	"net/http"
	"strings"
)

// verbatim real Swift responses
const (
	notFoundResponse = `404 Not Found

The resource could not be found.


`
	createdResponse = `201 Created




`
	acceptedResponse = `202 Accepted

The request is accepted for processing.


`
)

// handleContainers processes HTTP requests for container management.
func (s *Swift) handleContainers(container string, w http.ResponseWriter, r *http.Request) {
	var err error
	w.Header().Set("Content-Type", "text/html; charset=UTF-8")
	exists := s.HasContainer(container)
	if !exists && r.Method != "PUT" {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(notFoundResponse))
		return
	}
	switch r.Method {
	case "GET":
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("[]"))
	case "DELETE":
		if err = s.RemoveContainer(container); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
		} else {
			w.Header().Set("Content-Length", "0")
			w.WriteHeader(http.StatusNoContent)
		}
	case "PUT":
		if exists {
			w.WriteHeader(http.StatusAccepted)
			w.Write([]byte(acceptedResponse))
		} else {
			if err = s.AddContainer(container); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(err.Error()))
			} else {
				w.WriteHeader(http.StatusCreated)
				w.Write([]byte(createdResponse))
			}
		}
	default:
		panic("not implemented request type: " + r.Method)
	}
}

// handleObjects processes HTTP requests for object management.
func (s *Swift) handleObjects(container, object string, w http.ResponseWriter, r *http.Request) {
	var err error
	w.Header().Set("Content-Type", "text/html; charset=UTF-8")
	if exists := s.HasContainer(container); !exists {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(notFoundResponse))
		return
	}
	objdata, err := s.GetObject(container, object)
	if err != nil && r.Method != "PUT" {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(notFoundResponse))
		return
	}
	exists := err == nil
	switch r.Method {
	case "GET":
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json; charset=UF-8")
		w.Write([]byte(objdata))
	case "DELETE":
		if err = s.RemoveObject(container, object); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
		} else {
			w.Header().Set("Content-Length", "0")
			w.WriteHeader(http.StatusNoContent)
		}
	case "PUT":
		bodydata, err := ioutil.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}
		if exists {
			err = s.RemoveObject(container, object)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(err.Error()))
			}
		}
		if err = s.AddObject(container, object, bodydata); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
		} else {
			w.WriteHeader(http.StatusCreated)
			w.Write([]byte(createdResponse))
		}
	default:
		panic("not implemented request type: " + r.Method)
	}
}

// ServeHTTP is the main entry point in the HTTP request processing.
func (s *Swift) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	token := r.Header.Get("X-Auth-Token")
	if token != s.token {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	path := r.URL.Path
	if path[:len(s.baseURL)] == s.baseURL {
		path = path[len(s.baseURL):]
	}
	path = strings.TrimRight(path, "/")
	parts := strings.Split(path, "/")
	if len(parts) == 1 {
		container := parts[0]
		s.handleContainers(container, w, r)
	} else if len(parts) == 2 {
		container := parts[0]
		object := parts[1]
		s.handleObjects(container, object, w, r)
	} else {
		panic("not implemented request: " + r.URL.Path)
	}
}
