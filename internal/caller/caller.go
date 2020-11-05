package caller

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/UKHomeOffice/snowsync/internal/client"
)

// Call makes an outbound create request and returns a SNOW identifier
func Call(e client.Envelope) (string, error) {

	surl, err := url.Parse(os.Getenv("SNOW_URL"))
	if err != nil {
		return "", fmt.Errorf("no SNOW URL provided: %v", err)
	}

	c := &client.Client{
		BaseURL:    surl,
		HTTPClient: &http.Client{Timeout: 5 * time.Second},
	}

	out, err := json.Marshal(e)
	if err != nil {
		return "", fmt.Errorf("failed to marshal snow payload: %v", err)
	}

	req, err := c.NewRequest("", out)
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
