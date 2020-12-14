package dbupdater

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

func (md *mockDynamoDB) UpdateItem(input *dynamodb.UpdateItemInput) (*dynamodb.UpdateItemOutput, error) {
	output := new(dynamodb.UpdateItemOutput)

	null := &dynamodb.UpdateItemInput{
		Key: map[string]*dynamodb.AttributeValue{
			"external_identifier": {
				S: aws.String(""),
			},
		},
	}

	if cmp.Equal(input.Key, null.Key) {
		return nil, errors.New("failed to update db: ")
	}

	return output, md.err
}

func TestDBUpdate(t *testing.T) {

	tt := []struct {
		name          string
		issueID       string
		commentID     string
		commentAuthor string
		commentBody   string
		err           string
	}{
		{name: "happy", issueID: "abc-123", commentID: "1", commentAuthor: "bob", commentBody: "first comment"},
		// why error duplicated?
		{name: "unhappy", err: "failed to update db: failed to update db:"},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {

			updater := NewUpdater(&mockDynamoDB{})
			slice := make([]string, 1)
			slice[0] = tc.commentID + tc.commentAuthor + tc.commentBody
			var interfaceSlice []interface{} = make([]interface{}, len(slice))
			for i, d := range slice {
				interfaceSlice[i] = d
			}
			in := map[string]interface{}{"external_identifier": "", "comments": interfaceSlice}

			err := updater.DBUpdate(in)
			if err != nil {
				if msg := err.Error(); !strings.Contains(msg, tc.err) {
					t.Errorf("expected error %q, got: %q", tc.err, msg)
				}
				return
			}
		})
	}
}
