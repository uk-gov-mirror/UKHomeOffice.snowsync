package outprocessor

import (
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/events/test"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbiface"
	"github.com/aws/aws-sdk-go/service/lambda"
	"github.com/aws/aws-sdk-go/service/lambda/lambdaiface"
	"github.com/google/go-cmp/cmp"
)

type mockDynamoDB struct {
	dynamodbiface.DynamoDBAPI
	err error
}

type mockCreateForwarder struct {
	lambdaiface.LambdaAPI
	//err error
}

type mockUpdateForwarder struct {
	lambdaiface.LambdaAPI
	//err error
}

func getMockDB() *Dynamo {
	return &Dynamo{DynamoDB: &mockDynamoDB{}}
}

func getMockCreateForwarder() *Forwarder {
	return &Forwarder{Lambda: &mockCreateForwarder{}}
}

func getMockUpdateForwarder() *Forwarder {
	return &Forwarder{Lambda: &mockUpdateForwarder{}}
}

func getMockCreateProcesor() *Processor {
	return &Processor{lm: *getMockCreateForwarder(), db: *getMockDB()}
}

func getMockUpdateProcesor() *Processor {
	return &Processor{lm: *getMockUpdateForwarder(), db: *getMockDB()}
}

func (m *mockCreateForwarder) Invoke(in *lambda.InvokeInput) (*lambda.InvokeOutput, error) {
	var out lambda.InvokeOutput
	out.Payload = []byte(`"inc-123"`)
	return &out, nil
}

func (m *mockUpdateForwarder) Invoke(in *lambda.InvokeInput) (out *lambda.InvokeOutput, err error) {
	return out, nil
}

func (md *mockDynamoDB) GetItem(in *dynamodb.GetItemInput) (out *dynamodb.GetItemOutput, err error) {

	existing := &dynamodb.GetItemInput{
		Key: map[string]*dynamodb.AttributeValue{
			"external_identifier": {
				S: aws.String("abc-124"),
			},
		},
		TableName: aws.String("foo"),
	}

	if in.Key == nil {
		return nil, md.err
	} else if in.String() == existing.String() {
		return out, md.err
	}

	out = &dynamodb.GetItemOutput{
		Item: map[string]*dynamodb.AttributeValue{
			"internal_identifier": {
				S: aws.String("inc-123"),
			},
		},
	}
	return out, md.err
}

func (md *mockDynamoDB) PutItem(in *dynamodb.PutItemInput) (out *dynamodb.PutItemOutput, err error) {

	null := &dynamodb.PutItemInput{
		Item: map[string]*dynamodb.AttributeValue{
			"external_identifier": {
				NULL: aws.Bool(true),
			},
		},
	}

	if cmp.Equal(in.Item, null.Item) {
		return nil, errors.New("could not put to db: ")
	}

	return out, md.err
}

func (md *mockDynamoDB) Query(in *dynamodb.QueryInput) (*dynamodb.QueryOutput, error) {

	out := &dynamodb.QueryOutput{
		Count: aws.Int64(0),
	}
	return out, md.err
}

func TestWrite(t *testing.T) {

	tt := []struct {
		name  string
		extid string
		err   string
	}{
		{name: "happy", extid: "abc-123"},
		{name: "unhappy", err: "could not put to db:"},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {

			db := getMockDB()

			pay := Incident{
				ExtID: tc.extid,
			}

			err := db.writeItem(pay)
			if err != nil {
				if msg := err.Error(); !strings.Contains(msg, tc.err) {
					t.Errorf("expected error %q, got: %q", tc.err, msg)
				}
				return
			}
		})
	}
}

func TestCreate(t *testing.T) {

	tt := []struct {
		name               string
		externalIdentifier string
		comments           string
		err                string
	}{
		{name: "happy", externalIdentifier: "abc-123", comments: "foo"},
		{name: "unhappy", err: "could not put to db:"},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {

			p := getMockCreateProcesor()

			pay := Incident{
				ExtID:   tc.externalIdentifier,
				Comment: tc.comments,
			}

			id, err := p.lm.create(pay)
			if tc.err == "" && err != nil {
				if msg := err.Error(); !strings.Contains(msg, tc.err) {
					t.Errorf("expected error %q, got: %q", tc.err, msg)
				}
			}

			if id != "inc-123" {
				t.Errorf("expected id inc-123, got %v", id)
			}
		})
	}
}

func TestUpdate(t *testing.T) {

	tt := []struct {
		name               string
		internalIdentifier string
		comments           string
		err                string
	}{
		{name: "happy", internalIdentifier: "inc-123", comments: "foo"},
		{name: "unhappy", err: "could not put to db:"},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {

			p := getMockUpdateProcesor()

			pay := Incident{
				IntID:   tc.internalIdentifier,
				Comment: tc.comments,
			}

			err := p.lm.update(pay)
			if tc.err == "" && err != nil {
				if msg := err.Error(); !strings.Contains(msg, tc.err) {
					t.Errorf("expected error %q, got: %q", tc.err, msg)
				}
			}
		})
	}
}

func TestProcess(t *testing.T) {

	tt := []struct {
		name               string
		externalIdentifier string
		comments           string
		err                string
	}{
		{name: "happy", externalIdentifier: "abc-123", comments: "foo"},
		{name: "unhappy", err: "could not check partial item"},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {

			p := getMockCreateProcesor()

			var inputEvent events.SQSEvent
			file := test.ReadJSONFromFile(t, "../../test_event.json")
			if err := json.Unmarshal(file, &inputEvent); err != nil {
				t.Fatalf("could not unmarshal event: %v", err)
			}

			err := Process(&inputEvent)
			if msg := err.Error(); !strings.Contains(msg, tc.err) {
				t.Errorf("expected error %q, got: %q", tc.err, msg)
			}

			os.Setenv("TABLE_NAME", "foo")
			pay := Incident{
				ExtID:   tc.externalIdentifier,
				Comment: tc.comments,
			}

			partial, iid, err := p.db.checkPartial(pay)
			if tc.err == "" {
				if err != nil {
					if msg := err.Error(); !strings.Contains(msg, tc.err) {
						t.Errorf("expected error %q, got: %q", tc.err, msg)
					}
					return
				}
			}
			if partial && iid == "" {
				t.Errorf("partial entry had no id")
			}

			exact, iid, err := p.db.checkExact(pay)
			if tc.err == "" {
				if err != nil {
					if msg := err.Error(); !strings.Contains(msg, tc.err) {
						t.Errorf("expected error %q, got: %q", tc.err, msg)
					}
					return
				}
			}
			if exact && iid == "" {
				t.Errorf("exact entry had no id")
			}

		})
	}
}
