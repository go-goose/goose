package httpsuite

import (
	"io/ioutil"
	. "launchpad.net/gocheck"
	"net/http"
	"testing"
)

type HTTPTestSuite struct {
	HTTPSuite
}

func Test(t *testing.T) {
	TestingT(t)
}

var _ = Suite(&HTTPTestSuite{})

type HelloHandler struct{}

func (h *HelloHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(200)
	w.Write([]byte("Hello World\n"))
}

func (s *HTTPTestSuite) TestHelloWorld(c *C) {
	s.Mux.Handle("/", &HelloHandler{})
	// fmt.Printf("Running HelloWorld\n")
	response, err := http.Get(s.Server.URL)
	c.Check(err, IsNil)
	content, err := ioutil.ReadAll(response.Body)
	response.Body.Close()
	c.Check(err, IsNil)
	c.Check(response.Status, Equals, "200 OK")
	c.Check(response.StatusCode, Equals, 200)
	c.Check(string(content), Equals, "Hello World\n")
}
