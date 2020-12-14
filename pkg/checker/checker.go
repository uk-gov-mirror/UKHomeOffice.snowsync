// Package checker queries DynamoDB and returns a SNOW identifier if one exists.
package checker

import (
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
)

// DBChecker is an abstraction (helpful for testing)
type DBChecker interface {
	GetItem(*dynamodb.GetItemInput) (*dynamodb.GetItemOutput, error)
}

// NewChecker returns a new checker
func NewChecker(d DBChecker) *Checker {
	return &Checker{ddb: d}
}

// Checker is a record checker
type Checker struct {
	ddb DBChecker
}

// Result is the outcome of db check
type Result struct {
	ExtID string `json:"external_identifier"`
	IntID string `json:"internal_identifier"`
}

// Check checks if a prior db record has snow identifier
func (c *Checker) Check(payload map[string]interface{}) (Result, error) {

	eid := payload["external_identifier"].(string)

	if eid == "" {
		return Result{}, fmt.Errorf("no identifier in payload")
	}

	input := &dynamodb.GetItemInput{
		TableName: aws.String(os.Getenv("TABLE_NAME")),
		Key: map[string]*dynamodb.AttributeValue{
			"external_identifier": {
				S: aws.String(eid),
			},
		},
	}

	resp, err := c.ddb.GetItem(input)
	if err != nil {
		return Result{}, fmt.Errorf("failed to get item: %v", err)
	}

	// dynamically decode db item
	var itm map[string]interface{}
	err = dynamodbattribute.UnmarshalMap(resp.Item, &itm)
	if err != nil {
		return Result{}, fmt.Errorf("failed to unmarshal item: %v", err)
	}

	if itm["internal_identifier"] != nil {
		res := Result{ExtID: eid, IntID: itm["internal_identifier"].(string)}
		fmt.Printf("Issue id %v has a SNOW identifier: %v", eid, itm["internal_identifier"].(string))
		return res, nil
	}
	res := Result{ExtID: eid}
	fmt.Printf("Issue id %v has no SNOW identifier", eid)
	return res, nil
}
