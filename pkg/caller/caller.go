// Package caller makes a HTTP request to SNOW to create/update a ticket and returns a SNOW identifier.
package caller

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/UKHomeOffice/snowsync/pkg/client"
)

// CallSNOW makes an outbound create request and returns a SNOW identifier
func CallSNOW(ms map[string]interface{}) (string, error) {

	base, ok := os.LookupEnv("SNOW_URL")
	if !ok {
		return "", fmt.Errorf("missing SNOW URL")
	}

	surl, err := url.Parse(base)
	if err != nil {
		return "", fmt.Errorf("could not parse SNOW URL: %v", err)
	}

	user, ok := os.LookupEnv("ADMIN_USER")
	if !ok {
		return "", fmt.Errorf("missing username")
	}

	pass, ok := os.LookupEnv("ADMIN_PASS")
	if !ok {
		return "", fmt.Errorf("missing password")
	}

	c := &client.Client{
		BaseURL:    surl,
		HTTPClient: &http.Client{Timeout: 5 * time.Second},
	}

	out, err := json.Marshal(&ms)
	if err != nil {
		return "", fmt.Errorf("failed to marshal snow payload: %v", err)
	}

	req, err := c.NewRequest("", "POST", user, pass, out)
	if err != nil {
		return "", fmt.Errorf("failed to make request: %v", err)
	}

	res, err := c.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to call SNOW: %v", err)
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read SNOW response body %v", err)
	}

	fmt.Printf("sent request, SNOW replied with: %v", string(body))

	// dynamically decode SNOW response
	var dat map[string]interface{}
	err = json.Unmarshal(body, &dat)
	if err != nil {
		return "", fmt.Errorf("failed to decode SNOW response: %v", err)
	}

	// check for SNOW identifier
	rts := dat["result"].(map[string]interface{})
	ini := rts["internal_identifier"].(string)
	if ini != "" {
		fmt.Printf("SNOW returned an identifier: %v", ini)
		return ini, nil
	}
	return "", fmt.Errorf("request failed, SNOW did not return an identifier")

}
