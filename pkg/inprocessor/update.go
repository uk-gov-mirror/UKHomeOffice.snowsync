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

func transformUpdate(p Incident) (map[string]interface{}, error) {

	dat := make(map[string]interface{})

	v := Values{
		Comment: fmt.Sprintf("Comment added on ServiceNow (%v): %v", p.CommentID, p.Comment),
	}

	dat["external_identifier"] = p.ExtID
	dat["body"] = v
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
	c := &client.Client{
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
		// todo: remove request bin suffix
		path, err := url.Parse("/rest/api/2/issue/" + eid + "/comment")
		if err != nil {
			return "", fmt.Errorf("could not form JSD URL: %v", err)
		}
		out, err := json.Marshal(&dat)
		if err != nil {
			return "", fmt.Errorf("could marshal JSD payload: %v", err)
		}
		fmt.Printf("debug payload: %+v", string(out))

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
		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return "", fmt.Errorf("could not read JSD response body %v", err)
		}

		fmt.Printf("sent request, JSD replied with: %v", string(body))
		return eid, nil
	}
	return "", fmt.Errorf("no identifier in payload")
}

func (p *Processor) update(pay Incident) (string, error) {

	v, err := transformUpdate(pay)
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
