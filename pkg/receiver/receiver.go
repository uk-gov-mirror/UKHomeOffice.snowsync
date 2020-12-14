// Package receiver handles a webhook from SNOW, writes its payload to DynamoDB and and makes a HTTP request to JSD to update a ticket.
// This is a temporary all in one function until SNOW implements ACP bound transactions.
package receiver

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

	"github.com/UKHomeOffice/snowsync/pkg/client"
)

// DBUpdater is an abstraction
type DBUpdater interface {
	UpdateItem(*dynamodb.UpdateItemInput) (*dynamodb.UpdateItemOutput, error)
}

// Receiver adds SNOW comments to JSD tickets
type Receiver struct {
	ddb DBUpdater
}

// NewReceiver returns a new receiver
func NewReceiver(du DBUpdater) *Receiver {
	return &Receiver{ddb: du}
}

// AddUpdateToDB adds a SNOW generated update to DynamoDB
func (r *Receiver) AddUpdateToDB(b []byte) error {

	// get id and comments
	dat := map[string]string{}
	err := json.Unmarshal(b, &dat)
	if err != nil {
		return fmt.Errorf("failed to unmarshal update: %v", err)
	}
	eid := dat["supplier_reference"]
	com := dat["comments"]

	input := &dynamodb.UpdateItemInput{
		TableName:        aws.String(os.Getenv("TABLE_NAME")),
		UpdateExpression: aws.String("SET comments = :com"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":com": {
				S: aws.String(com),
			},
		},
		Key: map[string]*dynamodb.AttributeValue{
			"external_identifier": {
				S: aws.String(eid),
			},
		},
	}

	_, err = r.ddb.UpdateItem(input)
	if err != nil {
		return fmt.Errorf("failed to update db: %v", err)
	}

	fmt.Printf("%v updated on db", eid)
	return nil
}

// CallJSD adds a SNOW generated update to JSD issue
func (r *Receiver) CallJSD(b []byte) error {

	// get id and comments
	dat := map[string]string{}
	err := json.Unmarshal(b, &dat)
	if err != nil {
		return fmt.Errorf("failed to unmarshal JSD payload: %v", err)
	}
	eid := dat["supplier_reference"]
	com := dat["comments"]

	msg := struct {
		Body string `json:"body,omitempty"`
	}{
		Body: com,
	}

	out, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal JSD payload: %v", err)
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
		return fmt.Errorf("failed to make request: %v", err)
	}

	res, err := c.Do(req)
	if err != nil {
		return fmt.Errorf("failed to call JSD: %v", err)
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		return fmt.Errorf("JSD call failed: %v", res.StatusCode)
	}

	fmt.Printf("%v updated on JSD", eid)
	return nil
}

// Handle deals with the incoming request from SNOW
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
