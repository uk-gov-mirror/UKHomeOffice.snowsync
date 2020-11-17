package checker

import (
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbiface"
)

type mockDynamoDB struct {
	dynamodbiface.DynamoDBAPI
	err error
}

func (md *mockDynamoDB) GetItem(input *dynamodb.GetItemInput) (*dynamodb.GetItemOutput, error) {

	output := new(dynamodb.GetItemOutput)

	existing := &dynamodb.GetItemInput{
		Key: map[string]*dynamodb.AttributeValue{
			"external_identifier": {
				S: aws.String("abc-124"),
			},
		},
		TableName: aws.String(""),
	}

	if input.Key == nil {
		return nil, md.err
	} else if input.String() == existing.String() {
		return output, md.err
	}
	output = &dynamodb.GetItemOutput{
		Item: map[string]*dynamodb.AttributeValue{
			"internal_identifier": {
				S: aws.String("inc-123"),
			},
		},
	}
	return output, md.err
}

func TestCheck(t *testing.T) {

	tt := []struct {
		name  string
		extID string
		err   string
	}{
		{name: "happy_old", extID: "abc-123"},
		{name: "happy_new", extID: "abc-124"},
		{name: "unhappy", err: "no identifier in payload"},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			checker := NewChecker(&mockDynamoDB{})
			m := make(map[string]interface{})
			m["external_identifier"] = tc.extID

			_, err := checker.Check(m)

			if tc.err == "" {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
			if err != nil {
				if msg := err.Error(); !strings.Contains(msg, tc.err) {
					t.Errorf("expected error %q, got: %q", tc.err, msg)
				}
				return
			}
		})

	}
}
