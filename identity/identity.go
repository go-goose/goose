package identity

type AuthDetails struct {
	Token       string
	ServiceURLs map[string]string
}

type Credentials struct {
	URL     string // The URL to authenticate against
	User    string // The username to authenticate as
	Secrets string // The secrets to pass
}

type Authenticator interface {
	Auth(URL string) (*AuthDetails, error)
}
