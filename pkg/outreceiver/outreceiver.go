// Package outreceiver handles comments added to tickets initially created in JSD
package outreceiver

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"

	"github.com/UKHomeOffice/snowsync/pkg/client"
)

// DBUpdater is an abstraction
type DBUpdater interface {
	PutItem(*dynamodb.PutItemInput) (*dynamodb.PutItemOutput, error)
}

// Receiver adds SNow comments to JSD tickets
type Receiver struct {
	ddb DBUpdater
}

// Values make up the JSD payload
type Values struct {
	Transition *transition `json:"transition,omitempty"`
}

type transition struct {
	ID string `json:"id,omitempty"`
}

// NewReceiver returns a new receiver
func NewReceiver(du DBUpdater) *Receiver {
	return &Receiver{ddb: du}
}

func (r *Receiver) add2DB(b []byte) error {

	// get id and comments
	dat := map[string]string{}
	err := json.Unmarshal(b, &dat)
	if err != nil {
		return fmt.Errorf("could not unmarshal update: %v", err)
	}

	_, ok := dat["status"]
	if ok {
		return nil
	}

	rec := struct {
		ExtID      string `json:"external_identifier,omitempty"`
		Comment    string `json:"comment,omitempty"`
		Comid      string `json:"comment_sysid,omitempty"`
		Intcomment string `json:"internal_comment,omitempty"`
		Intcomid   string `json:"internal_comment_sysid,omitempty"`
	}{
		ExtID:      dat["external_identifier"],
		Comment:    dat["comment"],
		Comid:      dat["comment_sysid"],
		Intcomment: dat["internal_comment"],
		Intcomid:   dat["internal_comment_sysid"],
	}

	if rec.Comid == "" {
		rec.Comid = rec.Intcomid
		rec.Comment = rec.Intcomment
	}

	item, err := dynamodbattribute.MarshalMap(rec)
	if err != nil {
		return fmt.Errorf("could not marshal db record: %s", err)
	}

	input := &dynamodb.PutItemInput{
		Item:      item,
		TableName: aws.String(os.Getenv("TABLE_NAME")),
	}

	_, err = r.ddb.PutItem(input)
	if err != nil {
		return fmt.Errorf("could not put to db: %v", err)
	}

	fmt.Printf("%v updated on db", dat["external_identifier"])
	return nil
}

func (r *Receiver) progressJSD(b []byte) error {

	dat := map[string]string{}
	err := json.Unmarshal(b, &dat)
	if err != nil {
		return fmt.Errorf("could not unmarshal JSD payload: %v", err)
	}

	eid := dat["external_identifier"]
	sta := dat["status"]

	// only allowing Investigating and Resolved at MVP stage
	var t string
	switch sta {
	case "1":
		fmt.Printf("ignoring status: %v", sta)
		return nil
	case "10100":
		t = "11"
	case "3":
		t = "71"
	default:
		return fmt.Errorf("unexpected ticket status: %v", sta)
	}

	v := Values{
		Transition: &transition{ID: t},
	}

	out, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("could not marshal JSD payload: %v", err)
	}

	base, ok := os.LookupEnv("JSD_URL")
	if !ok {
		return fmt.Errorf("no JSD URL provided: %v", err)
	}

	user, ok := os.LookupEnv("ADMIN_USER")
	if !ok {
		return fmt.Errorf("missing username")
	}
	pass, ok := os.LookupEnv("ADMIN_PASS")
	if !ok {
		return fmt.Errorf("missing password")
	}

	jurl, err := url.Parse(base + "/rest/api/2/issue/" + eid + "/transitions")
	if err != nil {
		return fmt.Errorf("could not form JSD URL: %v", err)
	}

	c := &client.Client{
		BaseURL:    jurl,
		HTTPClient: &http.Client{Timeout: 5 * time.Second},
	}

	req, err := c.NewRequest("", "POST", user, pass, out)
	if err != nil {
		return fmt.Errorf("could not make request: %v", err)
	}

	fmt.Printf("debug request in progressJSD %+v", req)

	res, err := c.Do(req)
	if err != nil {
		return fmt.Errorf("could not call JSD: %v", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusCreated && res.StatusCode != http.StatusOK {
		return fmt.Errorf("JSD call failed with status code: %v", res.StatusCode)
	}

	fmt.Printf("%v updated on JSD", eid)
	return nil

}

func (r *Receiver) callJSD(b []byte) error {

	// get id and comments
	dat := map[string]string{}
	err := json.Unmarshal(b, &dat)
	if err != nil {
		return fmt.Errorf("could not unmarshal JSD payload: %v", err)
	}
	eid := dat["external_identifier"]
	com := dat["comment"]
	comid := dat["comment_sysid"]
	icom := dat["internal_comment"]
	icomid := dat["internal_comment_sysid"]

	if comid == "" {
		comid = icomid
		com = icom
	}

	msg := struct {
		Body string `json:"body,omitempty"`
	}{
		Body: com,
	}

	out, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("could not marshal JSD payload: %v", err)
	}

	base, ok := os.LookupEnv("JSD_URL")
	if !ok {
		return fmt.Errorf("no JSD URL provided: %v", err)
	}

	user, ok := os.LookupEnv("ADMIN_USER")
	if !ok {
		return fmt.Errorf("missing username")
	}
	pass, ok := os.LookupEnv("ADMIN_PASS")
	if !ok {
		return fmt.Errorf("missing password")
	}

	jurl, err := url.Parse(base + "/rest/api/2/issue/" + eid + "/comment")
	if err != nil {
		return fmt.Errorf("could not form JSD URL: %v", err)
	}

	c := &client.Client{
		BaseURL:    jurl,
		HTTPClient: &http.Client{Timeout: 5 * time.Second},
	}

	req, err := c.NewRequest("", "POST", user, pass, out)
	if err != nil {
		return fmt.Errorf("could not make request: %v", err)
	}

	fmt.Printf("debug request in callJSD %+v", req)

	res, err := c.Do(req)
	if err != nil {
		return fmt.Errorf("could not call JSD: %v", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusCreated && res.StatusCode != http.StatusOK {
		return fmt.Errorf("JSD call failed with status code: %v", res.StatusCode)
	}

	fmt.Printf("%v updated on JSD", eid)
	return nil
}

// Handle deals with the incoming request from SNow
func (r *Receiver) Handle(request *events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	fmt.Printf("debug incoming payload: %v", request.Body)

	err := r.add2DB([]byte(request.Body))
	if err != nil {
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusBadRequest,
			Body:       err.Error(),
		}, nil
	}

	dat := map[string]string{}
	err = json.Unmarshal([]byte(request.Body), &dat)
	if err != nil {
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusInternalServerError,
			Body:       err.Error(),
		}, nil
	}

	_, ok := dat["status"]
	if ok {
		err = r.progressJSD([]byte(request.Body))
		if err != nil {
			return events.APIGatewayProxyResponse{
				StatusCode: http.StatusBadRequest,
				Body:       err.Error(),
			}, nil
		}
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusOK,
		}, nil
	}

	err = r.callJSD([]byte(request.Body))
	if err != nil {
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusBadRequest,
			Body:       err.Error(),
		}, nil
	}
	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
	}, nil

}
