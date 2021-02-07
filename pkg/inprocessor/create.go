package inprocessor

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"

	"github.com/UKHomeOffice/snowsync/pkg/client"
)

func transformCreate(p Incident) (map[string]interface{}, error) {

	dat := make(map[string]interface{})
	dat["serviceDeskId"] = "1"
	dat["requestTypeId"] = "14"

	// priority sync is out of scope for MVP so hardcoding
	pri := priority{
		Name: "P4 - General request",
	}

	v := Values{
		Priority: &pri,
		Summary:  p.Summary,
		Description: fmt.Sprintf("Incident %v raised on ServiceNow by %v with priority %v. Description: %v. First comment (%v): %v",
			p.IntID, p.Reporter, p.Priority, p.Description, p.CommentID, p.Comment),
	}

	dat["requestFieldValues"] = v
	return dat, nil

}

func createIncident(b []byte) (string, error) {

	user, pass, base, err := getEnv()
	if err != nil {
		return "", fmt.Errorf("environment error: %v", err)
	}

	surl, err := url.Parse(base)
	if err != nil {
		return "", fmt.Errorf("could not form JSD URL: %v", err)
	}

	// create client and request
	c := &client.Client{
		BaseURL:    surl,
		HTTPClient: &http.Client{Timeout: 5 * time.Second},
	}

	// todo: remove request bin suffix
	req, err := c.NewRequest("/p61p774/rest/servicedeskapi/request/", "POST", user, pass, b)
	if err != nil {
		return "", fmt.Errorf("could not make request: %v", err)
	}

	// make HTTP request to JSD
	res, err := c.Do(req)
	if err != nil {
		return "", fmt.Errorf("could not call JSD: %v", err)
	}
	defer res.Body.Close()

	// read HTTP response
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return "", fmt.Errorf("could not read JSD response body %v", err)
	}

	fmt.Printf("sent request, JSD replied with: %v", string(body))

	// dynamically decode response and check for JSD assigned identifier
	var dat map[string]interface{}
	err = json.Unmarshal(body, &dat)
	if err != nil {
		return "", fmt.Errorf("could not decode JSD response: %v", err)
	}

	eid, ok := dat["issueKey"].(string)
	if !ok && eid == "" {
		return "", fmt.Errorf("could not find an identifier in JSD response")
	}

	fmt.Printf("JSD returned an identifier: %v", eid)
	return eid, nil

}

func (p *Processor) create(in Incident) (string, error) {

	v, err := transformCreate(in)
	if err != nil {
		return "", fmt.Errorf("could not transform creator payload: %v", err)
	}

	new, err := json.Marshal(v)
	if err != nil {
		return "", fmt.Errorf("could not marshal creator payload: %v", err)
	}

	out, err := createIncident(new)
	if err != nil {
		return "", fmt.Errorf("could not invoke a create call: %v", err)
	}

	return out, nil
}
