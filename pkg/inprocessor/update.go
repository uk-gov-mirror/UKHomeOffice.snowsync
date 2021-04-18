package inprocessor

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/UKHomeOffice/snowsync/pkg/client"
)

func addComment(i Incident) error {

	user, pass, base, err := getEnv()
	if err != nil {
		return fmt.Errorf("environment error: %v", err)
	}

	surl, err := url.Parse(base)
	if err != nil {
		return fmt.Errorf("could not form JSD URL: %v", err)
	}

	// create payload
	dat := make(map[string]interface{})
	dat["body"] = fmt.Sprintf("Comment added on ServiceNow (%v): %v", i.CommentID, i.Comment)

	out, err := json.Marshal(&dat)
	if err != nil {
		return fmt.Errorf("could marshal JSD payload: %v", err)
	}

	// create HTTP request
	c := &client.Client{
		BaseURL:    surl,
		HTTPClient: &http.Client{Timeout: 5 * time.Second},
	}
	path, err := url.Parse("/rest/api/2/issue/" + i.ExtID + "/comment")
	if err != nil {
		return fmt.Errorf("could not form JSD URL: %v", err)
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

	if res.StatusCode != http.StatusCreated && res.StatusCode != http.StatusOK && res.StatusCode != http.StatusNoContent {
		return fmt.Errorf("JSD call failed with status code: %v", res.StatusCode)
	}

	fmt.Printf("%v updated on JSD\n", i.ExtID)
	return nil
}

func updatePriority(i Incident) error {

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

	switch i.Comment {
	case "ServiceNow updated Priority to 1":
		i.Priority = "P1 - Production system down"
	case "ServiceNow updated Priority to 2":
		i.Priority = "P2 - Production system impaired"
	case "ServiceNow updated Priority to 3":
		i.Priority = "P3 - Non production system impaired"
	case "ServiceNow updated Priority to 4":
		i.Priority = "P4 - General request"
	case "ServiceNow updated Priority to 5":
		i.Priority = "P4 - General request"
	default:
		return fmt.Errorf("unexpected priority: %v", i.Priority)
	}

	var pu priorityUpdate
	var pa priority

	pa = make(priority, 0)
	pa = append(pa, struct {
		Action set "json:\"set,omitempty\""
	}{set{Name: i.Priority}})
	pu.Pri = pa

	// create payload
	dat := make(map[string]interface{})
	dat["update"] = pu

	out, err := json.Marshal(&dat)
	if err != nil {
		return fmt.Errorf("could marshal JSD payload: %v", err)
	}

	// create HTTP request
	c := &client.Client{
		BaseURL:    surl,
		HTTPClient: &http.Client{Timeout: 5 * time.Second},
	}
	path, err := url.Parse("/rest/api/2/issue/" + i.ExtID)
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

	fmt.Printf("%v updated on JSD", i.ExtID)
	return nil
}

func (p *Processor) update(i Incident) error {

	fmt.Printf("debug payload into update %+v", i)

	if strings.Contains(i.Comment, "ServiceNow updated Priority") {
		err := updatePriority(i)
		if err != nil {
			return fmt.Errorf("could not make priority payload: %v", err)
		}
		return nil
	}
	err := addComment(i)
	if err != nil {
		return fmt.Errorf("could not make comment payload: %v", err)
	}

	return nil
}
