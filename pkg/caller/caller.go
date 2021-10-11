package caller

import (
	"bytes"
	"net/http"
	"net/url"
)

// Client is a HTTP client
type Client struct {
	BaseURL    *url.URL
	HTTPClient *http.Client
}

// NewRequest creates a HTTP request
func (c *Client) NewRequest(path, method, user, pass string, body []byte) (*http.Request, error) {

	p, err := url.Parse(path)
	if err != nil {
		return nil, err
	}
	u := c.BaseURL.ResolveReference(p)

	req, err := http.NewRequest(method, u.String(), bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")

	req.SetBasicAuth(user, pass)

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
