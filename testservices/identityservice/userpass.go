package identityservice

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

// Implement the v2 User Pass form of identity (Keystone)

type UserPassRequest struct {
	Auth struct {
		PasswordCredentials struct {
			Username string `json:"username"`
			Password string `json:"password"`
		} `json:"passwordCredentials"`
		TenantName string `json:"tenantName"`
	} `json:"auth"`
}

type Endpoint struct {
	AdminURL    string `json:"adminURL"`
	InternalURL string `json:"internalURL"`
	PublicURL   string `json:"publicURL"`
	Region      string `json:"region"`
}

type Service struct {
	Name      string `json:"name"`
	Type      string `json:"type"`
	Endpoints []Endpoint
}

type TokenResponse struct {
	Expires string `json:"expires"` // should this be a date object?
	Id      string `json:"id"`      // Actual token string
	Tenant  struct {
		Id   string `json:"id"`
		Name string `json:"name"`
	} `json:"tenant"`
}

type RoleResponse struct {
	Id       string `json:"id"`
	Name     string `json:"name"`
	TenantId string `json:"tenantId"`
}

type UserResponse struct {
	Id    string         `json:"id"`
	Name  string         `json:"name"`
	Roles []RoleResponse `json:"roles"`
}

type AccessResponse struct {
	Access struct {
		ServiceCatalog []Service     `json:"serviceCatalog"`
		Token          TokenResponse `json:"token"`
		User           UserResponse  `json:"user"`
	} `json:"access"`
}

// Taken from: http://docs.openstack.org/api/quick-start/content/index.html#Getting-Credentials-a00665
var exampleResponse = `{
    "access": {
        "serviceCatalog": [
            {
                "endpoints": [
                    {
                        "adminURL": "https://nova-api.trystack.org:9774/v1.1/1", 
                        "internalURL": "https://nova-api.trystack.org:9774/v1.1/1", 
                        "publicURL": "https://nova-api.trystack.org:9774/v1.1/1", 
                        "region": "RegionOne"
                    }
                ], 
                "name": "nova", 
                "type": "compute"
            }, 
            {
                "endpoints": [
                    {
                        "adminURL": "https://GLANCE_API_IS_NOT_DISCLOSED/v1.1/1", 
                        "internalURL": "https://GLANCE_API_IS_NOT_DISCLOSED/v1.1/1", 
                        "publicURL": "https://GLANCE_API_IS_NOT_DISCLOSED/v1.1/1", 
                        "region": "RegionOne"
                    }
                ], 
                "name": "glance", 
                "type": "image"
            }, 
            {
                "endpoints": [
                    {
                        "adminURL": "https://nova-api.trystack.org:5443/v2.0", 
                        "internalURL": "https://keystone.trystack.org:5000/v2.0", 
                        "publicURL": "https://keystone.trystack.org:5000/v2.0", 
                        "region": "RegionOne"
                    }
                ], 
                "name": "keystone", 
                "type": "identity"
            }
        ], 
        "token": {
            "expires": "2012-02-15T19:32:21", 
            "id": "5df9d45d-d198-4222-9b4c-7a280aa35666", 
            "tenant": {
                "id": "1", 
                "name": "admin"
            }
        }, 
        "user": {
            "id": "14", 
            "name": "annegentle", 
            "roles": [
                {
                    "id": "2", 
                    "name": "Member", 
                    "tenantId": "1"
                }
            ]
        }
    }
}`

type UserPass struct {
	users map[string]UserInfo
}

func NewUserPass() *UserPass {
	userpass := &UserPass{users: make(map[string]UserInfo)}
	return userpass
}

func (u *UserPass) AddUser(user, secret, token string) {
	u.users[user] = UserInfo{secret: secret, token: token}
}

func (u *UserPass) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var req UserPassRequest
	if r.Header.Get("Content-Type") != "application/json" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if content, err := ioutil.ReadAll(r.Body); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	} else {
		if err := json.Unmarshal(content, &req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
	}
	userInfo, ok := u.users[req.Auth.PasswordCredentials.Username]
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	if userInfo.secret != req.Auth.PasswordCredentials.Password {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	res := AccessResponse{}
	// We pre-populate the response with genuine entries so that it looks sane.
	// XXX: We should really build up valid state for this instead, at the
	//	very least, we should manage the URLs better.
	if err := json.Unmarshal([]byte(exampleResponse), &res); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	res.Access.Token.Id = userInfo.token
	if content, err := json.Marshal(res); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprint(err)))
		return
	} else {
		w.WriteHeader(http.StatusOK)
		w.Write(content)
		return
	}
	w.WriteHeader(http.StatusInternalServerError)
	return
}
