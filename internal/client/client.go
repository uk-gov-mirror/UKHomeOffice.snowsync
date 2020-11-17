package client

import (
	"bytes"
	"fmt"
	"net/http"
	"net/url"
	"os"
)

// Client is a HTTP client
type Client struct {
	BaseURL    *url.URL
	HTTPClient *http.Client
}

// Envelope is the JSON expected by SNOW
type Envelope struct {
	MsgID   string `json:"messageid,omitempty"`
	ExtID   string `json:"external_identifier,omitempty"`
	IntID   string `json:"internal_identifier,omitempty"`
	Payload string `json:"payload,omitempty"`
}

// NewRequest creates a HTTP request
func (c *Client) NewRequest(path string, body []byte) (*http.Request, error) {

	p, err := url.Parse(path)
	if err != nil {
		return nil, err
	}
	u := c.BaseURL.ResolveReference(p)

	req, err := http.NewRequest("POST", u.String(), bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")

	cerr := fmt.Errorf("missing credentials")
	admin, ok := os.LookupEnv("SNOW_USER")
	if !ok {
		return nil, cerr
	}
	password, ok := os.LookupEnv("SNOW_PASS")
	if !ok {
		return nil, cerr
	}
	req.SetBasicAuth(admin, password)

	return req, nil
}

// Do makes a HTTP request
func (c *Client) Do(req *http.Request) (*http.Response, error) {

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}

	return resp, err
}
