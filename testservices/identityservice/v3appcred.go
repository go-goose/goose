package identityservice

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/go-goose/goose/v5/testservices/hook"
)

// V3AppCredRequest implements the v3 application credential form of
// identity (Keystone) authentication.
type V3AppCredRequest struct {
	Auth struct {
		Identity struct {
			Methods               []string `json:"methods"`
			ApplicationCredential struct {
				ID     string `json:"id"`
				Secret string `json:"secret"`
			} `json:"application_credential"`
		} `json:"identity"`
		// A real Keystone rejects any request that combines an
		// application credential with an explicit scope,  since the
		// credential is already scoped to a single project at
		// creation time. Captured as raw JSON only so ServeHTTP can
		// detect - and reject - a caller that mistakenly sends one.
		Scope json.RawMessage `json:"scope,omitempty"`
	} `json:"auth"`
}

// AppCredInfo records a minted application credential and the
// user/project it authenticates as.
type AppCredInfo struct {
	Secret      string
	Token       string
	UserId      string
	ProjectId   string
	ProjectName string
}

// V3AppCred represents a fake Keystone v3 identity service that only
// accepts application-credential authentication.
type V3AppCred struct {
	hook.TestService
	creds    map[string]AppCredInfo
	services []V3Service
}

// NewV3AppCred returns a new V3AppCred.
func NewV3AppCred() *V3AppCred {
	return &V3AppCred{
		creds:    make(map[string]AppCredInfo),
		services: make([]V3Service, 0),
	}
}

// AddCredential registers an application credential that can
// authenticate against this fake service, returning the AppCredInfo
// so tests can assert on the token/user/project it will yield.
func (a *V3AppCred) AddCredential(
	id, secret, token, userId, projectId, projectName string,
) AppCredInfo {
	info := AppCredInfo{
		Secret:      secret,
		Token:       token,
		UserId:      userId,
		ProjectId:   projectId,
		ProjectName: projectName,
	}
	a.creds[id] = info
	return info
}

// AddService adds a service to the current V3AppCred.
func (a *V3AppCred) AddService(service Service) {
	a.services = append(a.services, service.V3)
}

// returnFailure wraps and returns an error through the http
// connection.
func (a *V3AppCred) returnFailure(
	w http.ResponseWriter, status int, message string,
) {
	e := ErrorWrapper{
		Error: ErrorResponse{
			Message: message,
			Code:    status,
			Title:   http.StatusText(status),
		},
	}
	content, err := json.Marshal(e)
	if err != nil {
		w.Header().Set(
			"Content-Length", fmt.Sprintf("%d", len(internalError)),
		)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write(internalError)
		return
	}
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(content)))
	w.WriteHeader(status)
	w.Write(content)
}

// ServeHTTP serves V3AppCred for testing purposes.
func (a *V3AppCred) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var req V3AppCredRequest
	w.Header().Set("Content-Type", "application/json")
	if r.Header.Get("Content-Type") != "application/json" {
		a.returnFailure(w, http.StatusBadRequest, notJSON)
		return
	}
	content, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if err := json.Unmarshal(content, &req); err != nil {
		a.returnFailure(w, http.StatusBadRequest, notJSON)
		return
	}
	if err := a.ProcessControlHook("preauthentication", a, req); err != nil {
		a.returnFailure(w, http.StatusInternalServerError, err.Error())
		return
	}
	if len(req.Auth.Scope) > 0 {
		a.returnFailure(
			w, http.StatusBadRequest,
			"an application credential cannot be scoped to a project",
		)
		return
	}
	id := req.Auth.Identity.ApplicationCredential.ID
	secret := req.Auth.Identity.ApplicationCredential.Secret
	info, ok := a.creds[id]
	if !ok || secret == "" || info.Secret != secret {
		a.returnFailure(
			w, http.StatusUnauthorized,
			"The request you have made requires authentication.",
		)
		return
	}

	res := V3TokenResponse{
		Issued:  time.Now(),
		Expires: time.Now().Add(time.Hour),
		Methods: []string{"application_credential"},
		Catalog: a.services,
		Project: &V3Project{
			ID:   info.ProjectId,
			Name: info.ProjectName,
		},
	}
	res.User.ID = info.UserId

	responseContent, err := json.Marshal(struct {
		Token *V3TokenResponse `json:"token"`
	}{
		Token: &res,
	})
	if err != nil {
		a.returnFailure(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.Header().Set("X-Subject-Token", info.Token)
	w.WriteHeader(http.StatusCreated)
	w.Write(responseContent)
}

// SetupHTTP attaches all the needed handlers to provide the HTTP API.
func (a *V3AppCred) SetupHTTP(mux *http.ServeMux) {
	mux.Handle("/v3/auth/tokens", a)
}

func (a *V3AppCred) Stop() {
	// noop
}
