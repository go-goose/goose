package swiftservice

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
)

type object map[string][]byte

type Swift struct {
	containers map[string]object
	hostname   string
	baseURL    string
	token      string
}

func New(hostname, baseURL, token string) *Swift {
	swift := &Swift{
		containers: make(map[string]object),
		hostname:   hostname,
		baseURL:    baseURL,
		token:      token,
	}
	return swift
}

func (s *Swift) HasContainer(name string) bool {
	_, ok := s.containers[name]
	return ok
}

func (s *Swift) GetObject(container, name string) ([]byte, error) {
	data, ok := s.containers[container][name]
	if !ok {
		return nil, fmt.Errorf("no such object %q in container %q", name, container)
	}
	return data, nil
}

func (s *Swift) AddContainer(name string) error {
	if s.HasContainer(name) {
		return fmt.Errorf("container already exists %q", name)
	}
	s.containers[name] = make(object)
	return nil
}

func (s *Swift) AddObject(container, name string, data []byte) error {
	_, err := s.GetObject(container, name)
	if err == nil {
		return fmt.Errorf(
			"object %q in container %q already exists",
			name,
			container)
	}
	if ok := s.HasContainer(container); !ok {
		err = s.AddContainer(container)
		if err != nil {
			return err
		}
	}
	s.containers[container][name] = data
	return nil
}

func (s *Swift) RemoveContainer(name string) error {
	if ok := s.HasContainer(name); !ok {
		return fmt.Errorf("no such container %q", name)
	}
	s.containers[name] = nil
	delete(s.containers, name)
	return nil
}

func (s *Swift) RemoveObject(container, name string) error {
	_, err := s.GetObject(container, name)
	if err != nil {
		return err
	}
	s.containers[container][name] = nil
	delete(s.containers[container], name)
	return nil
}

func (s *Swift) GetURL(container, object string) (string, error) {
	_, err := s.GetObject(container, object)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s%s%s/%s", s.hostname, s.baseURL, container, object), nil
}

func (s *Swift) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := strings.Replace(r.URL.Path, s.baseURL, "", -1)
	if strings.HasSuffix(path, "/") {
		path = strings.TrimRight(path, "/")
	}
	parts := strings.Split(path, "/")
	var err error
	token := r.Header.Get("X-Auth-Token")
	if token != s.token {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	if len(parts) == 1 && r.Method == "PUT" {
		err = s.AddContainer(parts[0])
		if err != nil {
			w.WriteHeader(http.StatusAccepted)
			return
		} else {
			w.WriteHeader(http.StatusCreated)
			return
		}
	} else if len(parts) == 1 && r.Method == "GET" {
		w.WriteHeader(http.StatusNotImplemented)
	} else if len(parts) == 1 && r.Method == "DELETE" {
		err = s.RemoveContainer(parts[0])
		if err == nil {
			w.WriteHeader(http.StatusOK)
			return
		}
	} else if len(parts) == 2 && r.Method == "PUT" {
		defer r.Body.Close()
		body, err := ioutil.ReadAll(r.Body)
		if err == nil {
			err = s.AddObject(parts[0], parts[1], body)
			if err == nil {
				w.WriteHeader(http.StatusOK)
				return
			}
		}
	} else if len(parts) == 2 && r.Method == "GET" {
		data, err := s.GetObject(parts[0], parts[1])
		if err == nil {
			w.WriteHeader(http.StatusOK)
			w.Write(data)
			return
		}
	}
	w.WriteHeader(http.StatusNotFound)
}
