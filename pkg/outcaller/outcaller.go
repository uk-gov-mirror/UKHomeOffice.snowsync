// Package outcaller makes HTTP calls to SNow.
package outcaller

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

// CallSNow makes HTTP requests to SNow
func CallSNow(ms map[string]interface{}) (string, error) {

	// check environment
	base, ok := os.LookupEnv("SNOW_URL")
	if !ok {
		return "", fmt.Errorf("missing SNow URL")
	}

	surl, err := url.Parse(base)
	if err != nil {
		return "", fmt.Errorf("could not parse SNow URL: %v", err)
	}

	user, ok := os.LookupEnv("ADMIN_USER")
	if !ok {
		return "", fmt.Errorf("missing username")
	}

	pass, ok := os.LookupEnv("ADMIN_PASS")
	if !ok {
		return "", fmt.Errorf("missing password")
	}

	// create client and request
	c := &client.Client{
		BaseURL:    surl,
		HTTPClient: &http.Client{Timeout: 5 * time.Second},
	}

	out, err := json.Marshal(&ms)
	if err != nil {
		return "", fmt.Errorf("failed to marshal snow payload: %v", err)
	}

	//fmt.Printf("debug payload %+v", string(out))

	req, err := c.NewRequest("", "POST", user, pass, out)
	if err != nil {
		return "", fmt.Errorf("could not make request: %v", err)
	}

	// make HTTP request to SNow
	res, err := c.Do(req)
	if err != nil {
		return "", fmt.Errorf("could not call SNow: %v", err)
	}
	defer res.Body.Close()

	// read HTTP response
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return "", fmt.Errorf("could not read SNow response body %v", err)
	}

	fmt.Printf("sent request, SNow replied with: %v", string(body))

	// dynamically decode response and check for SNow assigned identifier
	var dat map[string]interface{}
	err = json.Unmarshal(body, &dat)
	if err != nil {
		return "", fmt.Errorf("could not decode SNow response: %v", err)
	}
	rts, ok := dat["result"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("could not find a result in SNow response")
	}
	ini, ok := rts["internal_identifier"].(string)
	if !ok {
		return "", fmt.Errorf("could not find an identifier in SNow response")
	}

	// return internal identifier
	if ini != "" {
		fmt.Printf("SNow returned an identifier: %v", ini)
		return ini, nil
	}
	return "", fmt.Errorf("request failed, SNow did not return an identifier")

}
