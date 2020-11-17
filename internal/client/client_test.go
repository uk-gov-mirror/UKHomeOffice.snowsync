package client

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"
)

func TestClient(t *testing.T) {

	tt := []struct {
		name    string
		path    string
		payload string
		err     string
	}{
		{name: "happy", path: "/", payload: `{"foo":"bar"}`},
		{name: "unhappy", path: "/", err: "missing credentials"},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {

			testSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

				ct := r.Header.Get("Content-Type")
				if ct != "application/json" {
					t.Errorf("wrong content type: %v", ct)
				}

				sa := r.Header.Get("Authorization")
				if sa != "Basic Zm9vOmJhcg==" {
					t.Errorf("wrong auth header: %v", sa)
				}

				body, err := ioutil.ReadAll(r.Body)
				if err != nil {
					t.Errorf("could not read request body: %v", sa)
				}

				if string(body) != tc.payload {
					t.Errorf("expected %v, got %v", tc.payload, string(body))
				}
			}))

			u, _ := url.Parse(testSrv.URL)
			c := &Client{
				BaseURL:    u,
				HTTPClient: &http.Client{Timeout: 5 * time.Second},
			}

			if tc.err != "" {
				os.Setenv("SNOW_USER", "foo")
				os.Setenv("SNOW_PASS", "bar")

				req, err := c.NewRequest(tc.path, []byte(tc.payload))
				if err != nil {
					t.Fatalf("could not make request: %q", err)
				}

				if req.URL.String() != (u.String() + tc.path) {
					t.Errorf("wrong target url: %v", req.URL.String())
				}

				resp, err := c.Do(req)
				if err != nil {
					t.Errorf("call failed: %v", err)
				}
				defer resp.Body.Close()
			}

			_, err := c.NewRequest(tc.path, []byte(tc.payload))
			if err != nil {
				if msg := err.Error(); !strings.Contains(msg, tc.err) {
					t.Errorf("expected error %q, got: %q", tc.err, msg)
				}
			}

		})
	}
}
