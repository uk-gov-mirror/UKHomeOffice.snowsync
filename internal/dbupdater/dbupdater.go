package dbupdater

import (
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
)

// DBUpdater is an abstraction (helpful for testing)
type DBUpdater interface {
	UpdateItem(*dynamodb.UpdateItemInput) (*dynamodb.UpdateItemOutput, error)
}

// Updater is a ticket updater
type Updater struct {
	ddb DBUpdater
}

// NewUpdater returns a new updater
func NewUpdater(u DBUpdater) *Updater {
	return &Updater{ddb: u}
}

// DBUpdate updates a db record
func (u *Updater) DBUpdate(ms map[string]interface{}) error {

	// get id and comments as slice
	eid := ms["external_identifier"].(string)
	comments := ms["comments"].([]interface{})
	s := make([]string, 0)
	for _, item := range comments {
		s = append(s, fmt.Sprintf("%v", item))
	}

	input := &dynamodb.UpdateItemInput{
		TableName:        aws.String(os.Getenv("TABLE_NAME")),
		UpdateExpression: aws.String("SET comments = :com"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":com": {
				SS: aws.StringSlice(s),
			},
		},
		Key: map[string]*dynamodb.AttributeValue{
			"external_identifier": {
				S: aws.String(eid),
			},
		},
	}

	_, err := u.ddb.UpdateItem(input)
	if err != nil {
		return fmt.Errorf("failed to update db: %v", err)
	}
	fmt.Printf("%v updated on db", eid)
	return nil
}
