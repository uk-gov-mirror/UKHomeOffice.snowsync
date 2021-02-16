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

// NewReceiver returns a new receiver
func NewReceiver(du DBUpdater) *Receiver {
	return &Receiver{ddb: du}
}

// AddUpdateToDB adds a SNow generated update to DynamoDB
func (r *Receiver) AddUpdateToDB(b []byte) error {

	// get id and comments
	dat := map[string]string{}
	err := json.Unmarshal(b, &dat)
	if err != nil {
		return fmt.Errorf("could not unmarshal update: %v", err)
	}

	item, err := dynamodbattribute.MarshalMap(dat)
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

// CallJSD adds a SNow generated update to JSD issue
func (r *Receiver) CallJSD(b []byte) error {

	// get id and comments
	dat := map[string]string{}
	err := json.Unmarshal(b, &dat)
	if err != nil {
		return fmt.Errorf("could not unmarshal JSD payload: %v", err)
	}
	eid := dat["external_identifier"]
	com := dat["comment"]
	icom := dat["internal_comment"]

	if icom != "" {
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

	err := r.AddUpdateToDB([]byte(request.Body))
	if err != nil {
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusBadRequest,
			Body:       err.Error(),
		}, nil
	}

	err = r.CallJSD([]byte(request.Body))
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
