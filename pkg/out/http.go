package out

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/UKHomeOffice/snowsync/pkg/caller"
)

func create(inc *Incident) (string, error) {

	// construct payload with SNOW required headers
	dat := make(map[string]interface{})
	// TODO: Check this is correct when ServiceNow creates the ServiceNow app/service.
	dat["messageid"] = "HO_SIAM_IN_REST_INC_POST_JSON_CORECLOUD_Incident_Create"
	dat["external_identifier"] = inc.Identifier
	dat["payload"] = inc

	new, err := json.Marshal(dat)
	if err != nil {
		return "", fmt.Errorf("could not marshal creator payload: %v", err)
	}

	iid, err := callSNOW(new)
	if err != nil {
		return "", fmt.Errorf("could not invoke a create call: %v", err)
	}

	// check for and return internal identifier
	if iid != "" {
		return iid, nil
	}
	return "", fmt.Errorf("no identifier in SNOW response")
}

func update(inc *Incident) error {

	// construct payload with SNOW required headers
	dat := make(map[string]interface{})

	switch inc.Service {
	case "CSOC":
		// TODO: Check this is correct when ServiceNow creates the ServiceNow app/service.
		dat["messageid"] = "HO_SIAM_IN_REST_SIT_UPDATE_JSON_CORECLOUD_SIRT_Update"
	default:
		// TODO: Check this is correct when ServiceNow creates the ServiceNow app/service.
		dat["messageid"] = "HO_SIAM_IN_REST_INC_UPDATE_JSON_CORECLOUD_Incident_Update"
	}

	dat["internal_identifier"] = inc.IntID
	// avoid repeating internal identifier in payload
	inc.IntID = ""
	dat["payload"] = inc

	update, err := json.Marshal(dat)
	if err != nil {
		return fmt.Errorf("could not marshal updater payload: %v", err)
	}

	_, err = callSNOW(update)
	if err != nil {
		return fmt.Errorf("could not invoke caller: %v", err)
	}
	return nil
}

func progress(inc *Incident) error {

	// construct payload with SNOW required headers
	dat := make(map[string]interface{})

	switch inc.Service {
	case "CSOC":
		// TODO: Check this is correct when ServiceNow creates the ServiceNow app/service.
		dat["messageid"] = "HO_SIAM_IN_REST_SIT_UPDATE_JSON_CORECLOUD_SIRT_Update"
	default:
		// TODO: Check this is correct when ServiceNow creates the ServiceNow app/service.
		dat["messageid"] = "HO_SIAM_IN_REST_INC_UPDATE_JSON_CORECLOUD_Incident_Update"
	}

	dat["internal_identifier"] = inc.IntID
	// remove irrelevant keys from payload
	inc.IntID = ""
	inc.Comment = ""
	inc.Priority = ""

	if inc.Status == "6" {
		inc.Resolution = "done"
	}

	dat["payload"] = inc

	progress, err := json.Marshal(dat)
	if err != nil {
		return fmt.Errorf("could not marshal updater payload: %v", err)
	}

	_, err = callSNOW(progress)
	if err != nil {
		return fmt.Errorf("could not invoke caller: %v", err)
	}

	return nil
}

func callSNOW(ms []byte) (string, error) {

	// check environment
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

	// create client and request
	c := &caller.Client{
		BaseURL:    surl,
		HTTPClient: &http.Client{Timeout: 5 * time.Second},
	}

	req, err := c.NewRequest("", "POST", user, pass, ms)
	if err != nil {
		return "", fmt.Errorf("could not make request: %v", err)
	}

	// make HTTP request to SNOW
	res, err := c.Do(req)
	if err != nil {
		return "", fmt.Errorf("could not call SNOW: %v", err)
	}
	defer res.Body.Close()

	// read HTTP response
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return "", fmt.Errorf("could not read SNOW response body %v", err)
	}

	fmt.Printf("sent request, SNOW replied with: %v", string(body))

	// dynamically decode response and check for SNOW assigned identifier
	var dat map[string]interface{}
	err = json.Unmarshal(body, &dat)
	if err != nil {
		return "", fmt.Errorf("could not decode SNOW response: %v", err)
	}
	rts, ok := dat["result"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("could not find a result in SNOW response")
	}
	ini, ok := rts["internal_identifier"].(string)
	if !ok {
		return "", fmt.Errorf("could not find an identifier in SNOW response")
	}

	// return internal identifier
	if ini != "" {
		fmt.Printf("SNOW returned an identifier: %v", ini)
		return ini, nil
	}
	return "", fmt.Errorf("request failed, SNOW did not return an identifier")

}
