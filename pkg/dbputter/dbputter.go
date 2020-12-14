// Package dbputter writes a new ticket to DynamoDB.
package dbputter

import (
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
)

// DBPutter is an abstraction (helpful for testing)
type DBPutter interface {
	PutItem(*dynamodb.PutItemInput) (*dynamodb.PutItemOutput, error)
}

// Putter is a ticket saver
type Putter struct {
	ddb DBPutter
}

// NewPutter returns a new putter
func NewPutter(d DBPutter) *Putter {
	return &Putter{ddb: d}
}

// DBPut creates a new db record
func (p *Putter) DBPut(ms map[string]interface{}) error {

	item, err := dynamodbattribute.MarshalMap(ms)
	if err != nil {
		return fmt.Errorf("failed to marshal db record: %s", err)
	}

	input := &dynamodb.PutItemInput{
		Item:      item,
		TableName: aws.String(os.Getenv("TABLE_NAME")),
	}

	_, err = p.ddb.PutItem(input)
	if err != nil {
		return fmt.Errorf("failed to put to db: %v", err)
	}
	fmt.Println("Record created")
	return nil
}
