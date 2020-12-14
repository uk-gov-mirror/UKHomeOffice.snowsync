package dbputter

import (
	"errors"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbiface"
	"github.com/google/go-cmp/cmp"
)

type mockDynamoDB struct {
	dynamodbiface.DynamoDBAPI
	err error
}

func (md *mockDynamoDB) PutItem(input *dynamodb.PutItemInput) (*dynamodb.PutItemOutput, error) {
	output := new(dynamodb.PutItemOutput)

	null := &dynamodb.PutItemInput{
		Item: map[string]*dynamodb.AttributeValue{
			"issueID": {
				NULL: aws.Bool(true),
			},
		},
	}

	if cmp.Equal(input.Item, null.Item) {
		return nil, errors.New("failed to put to db: ")
	}

	return output, md.err
}

func TestDBPut(t *testing.T) {

	tt := []struct {
		name    string
		issueID string
		err     string
	}{
		{name: "happy", issueID: "abc-123"},
		{name: "unhappy", err: "failed to put to db:"},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {

			putter := NewPutter(&mockDynamoDB{})
			m := map[string]interface{}{"issueID": tc.issueID}
			err := putter.DBPut(m)
			if err != nil {
				if msg := err.Error(); !strings.Contains(msg, tc.err) {
					t.Errorf("expected error %q, got: %q", tc.err, msg)
				}
				return
			}
		})
	}
}
