package neutron

import (
	"fmt"
	"net/http"

	gc "gopkg.in/check.v1"

	goosehttp "github.com/go-goose/goose/v5/http"
)

type clientHeaderSuite struct{}

var _ = gc.Suite(&clientHeaderSuite{})

func (s *clientHeaderSuite) TestNeutronHeaders(c *gc.C) {
	makeHeaders := func(m map[string]string) http.Header {
		headers := goosehttp.BasicHeaders()
		for k, v := range m {
			headers.Add(k, v)
		}
		return headers
	}

	type test struct {
		name          string
		method        string
		headers       http.Header
		contentType   string
		authToken     string
		payloadExists bool
		expected      http.Header
	}

	tests := []test{
		{
			name:   "test GET with empty args",
			method: "GET",
			expected: makeHeaders(map[string]string{
				"Accept": "",
			}),
		},
		{
			// TODO (stickupkid): This test is actually wrong, it shouldn't
			// return a Content-Type for GET, but to keep backwards
			// compatibility, we accept this.
			name:          "test GET",
			method:        "GET",
			contentType:   "application/json",
			payloadExists: true,
			expected: makeHeaders(map[string]string{
				"Content-Type": "application/json",
				"Accept":       "application/json",
			}),
		},
	}

	// Test that Content-Type and Accept are correctly applied.
	for _, method := range []string{"POST", "PUT", "PATCH"} {
		tests = append(tests, test{
			name:          fmt.Sprintf("test %s", method),
			method:        method,
			contentType:   "application/json",
			payloadExists: true,
			expected: makeHeaders(map[string]string{
				"Content-Type": "application/json",
				"Accept":       "application/json",
			}),
		}, test{
			name:        fmt.Sprintf("test %s", method),
			method:      method,
			contentType: "application/json",
			expected: makeHeaders(map[string]string{
				"Accept": "application/json",
			}),
		})
	}

	// Test that Content-Type is correctly applied, but not Accept.
	for _, method := range []string{"HEAD", "OPTIONS", "DELETE", "COPY"} {
		tests = append(tests, test{
			name:          fmt.Sprintf("test %s", method),
			method:        method,
			contentType:   "application/json",
			payloadExists: true,
			expected: makeHeaders(map[string]string{
				"Content-Type": "application/json",
			}),
		}, test{
			name:        fmt.Sprintf("test %s", method),
			method:      method,
			contentType: "application/json",
			expected:    makeHeaders(map[string]string{}),
		})
	}

	for i, test := range tests {
		c.Logf("test: %d, %s", i, test.name)

		got := NeutronHeaders(test.method, test.headers, test.contentType, test.authToken, test.payloadExists)
		c.Assert(got, gc.DeepEquals, test.expected)
	}
}
