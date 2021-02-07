package in

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"

	"github.com/UKHomeOffice/snowsync/pkg/caller"
)

func transformUpdate(inc *Incident) (map[string]interface{}, error) {

	dat := make(map[string]interface{})

	dat["external_identifier"] = inc.ExtID
	dat["body"] = fmt.Sprintf("Comment added on ServiceNow (%v): %v", inc.CommentID, inc.Comment)

	return dat, nil
}

func updateIncident(b []byte) (string, error) {

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

	// remove the need for this switcheroo
	var dat map[string]interface{}
	err = json.Unmarshal(b, &dat)
	if err != nil {
		return "", fmt.Errorf("could not decode payload to get external id: %v", err)
	}

	eid, ok := dat["external_identifier"].(string)
	if ok {
		delete(dat, "external_identifier")
		path, err := url.Parse("/rest/api/2/issue/" + eid + "/comment")
		if err != nil {
			return "", fmt.Errorf("could not form JSD URL: %v", err)
		}
		out, err := json.Marshal(&dat)
		if err != nil {
			return "", fmt.Errorf("could marshal JSD payload: %v", err)
		}

		req, err := c.NewRequest(path.Path, "POST", user, pass, out)
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
		_, err = ioutil.ReadAll(res.Body)
		if err != nil {
			return "", fmt.Errorf("could not read JSD response body %v", err)
		}

		//fmt.Printf("sent request, JSD replied with: %v", string(body))
		return eid, nil
	}
	return "", fmt.Errorf("no identifier in payload")
}

func (p *Processor) update(inc *Incident) (string, error) {

	v, err := transformUpdate(inc)
	if err != nil {
		return "", fmt.Errorf("could not transform creator payload: %v", err)
	}

	upd, err := json.Marshal(v)
	if err != nil {
		return "", fmt.Errorf("could not marshal updater payload: %v", err)
	}

	out, err := updateIncident(upd)
	if err != nil {
		return "", fmt.Errorf("could not invoke a create call: %v", err)
	}

	return out, nil
}

func (p *Processor) setStatus(inc *Incident) error {

	user, pass, base, err := getEnv()
	if err != nil {
		return fmt.Errorf("environment error: %v", err)
	}

	surl, err := url.Parse(base)
	if err != nil {
		return fmt.Errorf("could not form JSD URL: %v", err)
	}

	// create client and request
	c := &caller.Client{
		BaseURL:    surl,
		HTTPClient: &http.Client{Timeout: 5 * time.Second},
	}

	// t holds the transition code
	var t string
	switch inc.Status {
	case "":
		fmt.Printf("\nignoring blank status %v\n", inc.Status)
		return nil
	case "1":
		fmt.Printf("\nignoring status %v\n", inc.Status)
		return nil
	case "10100":
		t = "11"
	case "3":
		t = "121"
	default:
		return fmt.Errorf("\nunexpected ticket status: %v", inc.Status)
	}

	// add resolution comments
	var rc resolution
	var co comment

	co = make(comment, 0)
	co = append(co, struct {
		Action add "json:\"add,omitempty\""
	}{add{Body: inc.Resolution}})
	rc.Com = co

	v := Values{
		Resolution: &rc,
		Transition: &transition{ID: t},
	}

	path, err := url.Parse("/rest/api/2/issue/" + inc.ExtID + "/transitions")
	if err != nil {
		return fmt.Errorf("could not form JSD URL: %v", err)
	}
	out, err := json.Marshal(&v)
	if err != nil {
		return fmt.Errorf("could marshal JSD payload: %v", err)
	}

	req, err := c.NewRequest(path.Path, "POST", user, pass, out)
	if err != nil {
		return fmt.Errorf("could not make request: %v", err)
	}
	// make HTTP request to JSD
	res, err := c.Do(req)
	if err != nil {
		return fmt.Errorf("could not call JSD: %v", err)
	}
	defer res.Body.Close()

	return nil
}

func (p *Processor) setPriority(inc *Incident) error {

	user, pass, base, err := getEnv()
	if err != nil {
		return fmt.Errorf("environment error: %v", err)
	}

	surl, err := url.Parse(base)
	if err != nil {
		return fmt.Errorf("could not form JSD URL: %v", err)
	}

	// transform priority
	type set struct {
		Name string `json:"name"`
	}

	type priority []struct {
		Action set `json:"set,omitempty"`
	}

	type priorityUpdate struct {
		Pri priority `json:"priority,omitempty"`
	}

	switch inc.Comment {
	case "ServiceNow updated Priority to 1":
		inc.Priority = "P1 - Production system down"
	case "ServiceNow updated Priority to 2":
		inc.Priority = "P2 - Production system impaired"
	case "ServiceNow updated Priority to 3":
		inc.Priority = "P3 - Non production system impaired"
	case "ServiceNow updated Priority to 4":
		inc.Priority = "P4 - General request"
	case "ServiceNow updated Priority to 5":
		inc.Priority = "P4 - General request"
	default:
		fmt.Printf("ignoring blank or unexpected priority: %v", inc.Priority)
		return nil
	}

	var pu priorityUpdate
	var pa priority

	pa = make(priority, 0)
	pa = append(pa, struct {
		Action set "json:\"set,omitempty\""
	}{set{Name: inc.Priority}})
	pu.Pri = pa

	// create payload
	dat := make(map[string]interface{})
	dat["update"] = pu

	out, err := json.Marshal(&dat)
	if err != nil {
		return fmt.Errorf("could marshal JSD payload: %v", err)
	}

	// create HTTP request
	c := &caller.Client{
		BaseURL:    surl,
		HTTPClient: &http.Client{Timeout: 5 * time.Second},
	}
	path, err := url.Parse("/rest/api/2/issue/" + inc.ExtID)
	if err != nil {
		return fmt.Errorf("could not form JSD URL: %v", err)
	}
	req, err := c.NewRequest(path.Path, "PUT", user, pass, out)
	if err != nil {
		return fmt.Errorf("could not make request: %v", err)
	}

	// make HTTP request to JSD
	res, err := c.Do(req)
	if err != nil {
		return fmt.Errorf("could not call JSD: %v", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusCreated && res.StatusCode != http.StatusOK && res.StatusCode != http.StatusNoContent {
		return fmt.Errorf("JSD call failed with status code: %v", res.StatusCode)
	}

	fmt.Printf("%v updated on JSD", inc.ExtID)
	return nil
}
