package identityservice

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"gopkg.in/goose.v1/testservices/hook"
)

// V3UserPassRequest Implement the v3 User Pass form of identity (Keystone)
type V3UserPassRequest struct {
	Auth struct {
		Identity struct {
			Methods  []string `json:"methods"`
			Password struct {
				User struct {
					Name     string `json:"name"`
					Password string `json:"password"`
				} `json:"user"`
			} `json:"password"`
		} `json:"identity"`
		Scope struct {
			Project struct {
				Name string `json:"name"`
			} `json:"project"`
		} `json:"scope"`
	} `json:"auth"`
}

type V3Endpoint struct {
	Interface string `json:"interface"`
	RegionID  string `json:"region_id"`
	URL       string `json:"url"`
}

func NewV3Endpoints(adminURL, internalURL, publicURL, regionID string) []V3Endpoint {
	var eps []V3Endpoint
	if adminURL != "" {
		eps = append(eps, V3Endpoint{
			RegionID:  regionID,
			Interface: "admin",
			URL:       adminURL,
		})
	}
	if internalURL != "" {
		eps = append(eps, V3Endpoint{
			RegionID:  regionID,
			Interface: "internal",
			URL:       internalURL,
		})
	}
	if publicURL != "" {
		eps = append(eps, V3Endpoint{
			RegionID:  regionID,
			Interface: "public",
			URL:       publicURL,
		})
	}
	return eps

}

type V3Service struct {
	ID        string       `json:"id"`
	Name      string       `json:"name"`
	Type      string       `json:"type"`
	Endpoints []V3Endpoint `json:"endpoints"`
}

type V3TokenResponse struct {
	Expires time.Time   `json:"expires_at"`
	Issued  time.Time   `json:"issued_at"`
	Methods []string    `json:"methods"`
	Catalog []V3Service `json:"catalog,omitempty"`
	Project *v3Project  `json:"project,omitempty"`
	User    struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"user"`
}

type v3Project struct {
	ID string `json:"id,omitempty"`
}

type V3UserPass struct {
	hook.TestService
	Users
	services []V3Service
}

func NewV3UserPass() *V3UserPass {
	userpass := &V3UserPass{
		services: make([]V3Service, 0),
	}
	userpass.users = make(map[string]UserInfo)
	userpass.tenants = make(map[string]string)
	return userpass
}

func (u *V3UserPass) RegisterServiceProvider(name, serviceType string, serviceProvider ServiceProvider) {
	service := V3Service{
		ID:        name,
		Name:      name,
		Type:      serviceType,
		Endpoints: serviceProvider.V3Endpoints(),
	}
	u.AddService(Service{V3: service})
}

func (u *V3UserPass) AddService(service Service) {
	u.services = append(u.services, service.V3)
}

func (u *V3UserPass) ReturnFailure(w http.ResponseWriter, status int, message string) {
	e := ErrorWrapper{
		Error: ErrorResponse{
			Message: message,
			Code:    status,
			Title:   http.StatusText(status),
		},
	}
	if content, err := json.Marshal(e); err != nil {
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(internalError)))
		w.WriteHeader(http.StatusInternalServerError)
		w.Write(internalError)
	} else {
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(content)))
		w.WriteHeader(status)
		w.Write(content)
	}
}

func (u *V3UserPass) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var req V3UserPassRequest
	// Testing against Canonistack, all responses are application/json, even failures
	w.Header().Set("Content-Type", "application/json")
	if r.Header.Get("Content-Type") != "application/json" {
		u.ReturnFailure(w, http.StatusBadRequest, notJSON)
		return
	}
	if content, err := ioutil.ReadAll(r.Body); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	} else {
		if err := json.Unmarshal(content, &req); err != nil {
			u.ReturnFailure(w, http.StatusBadRequest, notJSON)
			return
		}
	}
	userInfo, errmsg := u.authenticate(
		req.Auth.Identity.Password.User.Name,
		req.Auth.Identity.Password.User.Password,
	)
	if errmsg != "" {
		u.ReturnFailure(w, http.StatusUnauthorized, errmsg)
		return
	}

	res, err := u.generateV3TokenResponse(userInfo)
	if err != nil {
		u.ReturnFailure(w, http.StatusInternalServerError, err.Error())
		return
	}
	if req.Auth.Scope.Project.Name != "" {
		res.Project = &v3Project{
			ID: u.addTenant(req.Auth.Scope.Project.Name),
		}
	}
	content, err := json.Marshal(struct {
		Token *V3TokenResponse `json:"token"`
	}{
		Token: res,
	})
	if err != nil {
		u.ReturnFailure(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.Header().Set("X-Subject-Token", userInfo.Token)
	w.WriteHeader(http.StatusCreated)
	w.Write(content)
}

func (u *V3UserPass) generateV3TokenResponse(userInfo *UserInfo) (*V3TokenResponse, error) {
	res := V3TokenResponse{}

	res.Issued = time.Now()
	res.Expires = res.Issued.Add(24 * time.Hour)
	res.Methods = []string{"password"}
	res.Catalog = u.services
	res.User.ID = userInfo.Id

	return &res, nil
}

// setupHTTP attaches all the needed handlers to provide the HTTP API.
func (u *V3UserPass) SetupHTTP(mux *http.ServeMux) {
	mux.Handle("/v3/auth/tokens", u)
}
