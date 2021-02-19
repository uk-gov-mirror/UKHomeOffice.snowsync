package inprocessor

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/UKHomeOffice/snowsync/pkg/client"
)

func (p *Processor) progress(pay Incident) (string, error) {

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

	// only allowing Investtigating and Resolved at MVP stage
	var t string
	switch pay.Status {
	case "":
		fmt.Printf("ignoring blank status %v", pay.Status)
		return pay.ExtID, nil
	case "1":
		fmt.Printf("ignoring status %v", pay.Status)
		return pay.ExtID, nil
	case "10100":
		t = "11"
	case "3":
		t = "71"
	default:
		return "", fmt.Errorf("unexpected ticket status: %v", pay.Status)
	}

	v := Values{
		Transition: &transition{ID: t},
	}

	path, err := url.Parse("/rest/api/2/issue/" + pay.ExtID + "/transitions")
	if err != nil {
		return "", fmt.Errorf("could not form JSD URL: %v", err)
	}
	out, err := json.Marshal(&v)
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

	return pay.ExtID, nil
}
