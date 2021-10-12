package in

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/UKHomeOffice/snowsync/pkg/caller"
)

// Values make up the JSD payload
type Values struct {
	Comment     string      `json:"comment,omitempty"`
	Description string      `json:"description,omitempty"`
	Service     []int       `json:"customfield_10002,omitempty"`
	SnowID      string      `json:"customfield_11824,omitempty"`
	Summary     string      `json:"summary,omitempty"`
	Priority    *priority   `json:"priority,omitempty"`
	Resolution  *resolution `json:"update,omitempty"`
	Transition  *transition `json:"transition,omitempty"`
}

type priority struct {
	Name string `json:"name,omitempty"`
}

type resolution struct {
	Com comment `json:"comment,omitempty"`
}

type add struct {
	Body string `json:"body,omitempty"`
}

type comment []struct {
	Action add `json:"add,omitempty"`
}

type transition struct {
	ID string `json:"id,omitempty"`
}

func transformCreate(inc *Incident) (map[string]interface{}, error) {

	dat := make(map[string]interface{})
	dat["serviceDeskId"] = "1"
	dat["requestTypeId"] = "14"

	var pri priority

	switch inc.Priority {
	case "1":
		pri.Name = "P1 - Production system down"
	case "2":
		pri.Name = "P2 - Production system impaired"
	case "3":
		pri.Name = "P3 - Non production system impaired"
	case "4":
		pri.Name = "P4 - General request"
	case "5":
		pri.Name = "P4 - General request"
	default:
		fmt.Printf("ignoring blank or unexpected priority: %v", inc.Priority)
		return nil, nil
	}

	// convert org code to int slice as that's what JSD expects
	d, err := strconv.Atoi(inc.Service)
	if err != nil {
		return nil, fmt.Errorf("could not convert organisation code: %v", err)
	}
	var org []int
	org = append(org, d)

	v := Values{
		Priority: &pri,
		Summary:  inc.Summary,
		Description: fmt.Sprintf("Incident %v raised on ServiceNow by %v with priority %v.\n %v\n %v %v",
			inc.IntID, inc.Reporter, inc.Priority, inc.Description, inc.Comment, inc.IntComment),
		SnowID:  inc.IntID,
		Service: org,
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
	c := &caller.Client{
		BaseURL:    surl,
		HTTPClient: &http.Client{Timeout: 5 * time.Second},
	}

	req, err := c.NewRequest("/rest/servicedeskapi/request/", "POST", user, pass, b)
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

func (p *Processor) create(in *Incident) (string, error) {

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
		return "", fmt.Errorf("could not make a create call: %v", err)
	}

	return out, nil
}
