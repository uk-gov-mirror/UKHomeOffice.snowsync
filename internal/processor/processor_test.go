package processor

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/events/test"
	"github.com/aws/aws-sdk-go/service/lambda"
	"github.com/aws/aws-sdk-go/service/lambda/lambdaiface"
)

type mockCreateInvoker struct {
	lambdaiface.LambdaAPI
	err error
}

type mockUpdateInvoker struct {
	lambdaiface.LambdaAPI
	err error
}

func getMockCreateInvoker() lambdaiface.LambdaAPI {
	return &mockCreateInvoker{}
}
func getMockUpdateInvoker() lambdaiface.LambdaAPI {
	return &mockUpdateInvoker{}
}

func (m *mockCreateInvoker) Invoke(*lambda.InvokeInput) (*lambda.InvokeOutput, error) {
	var out lambda.InvokeOutput
	out.Payload = []byte(`""`)
	return &out, nil
}

func (m *mockUpdateInvoker) Invoke(*lambda.InvokeInput) (*lambda.InvokeOutput, error) {
	var out lambda.InvokeOutput
	out.Payload = []byte(`"inc-123"`)
	return &out, nil
}

func TestProcess(t *testing.T) {

	tt := []struct {
		name               string
		externalIdentifier string
		internalIdentifier string
		err                string
	}{
		//{name: "create", externalIdentifier: "abc-123"},
		{name: "update", externalIdentifier: "abc-123", internalIdentifier: "inc-123"},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {

			var inputEvent events.SQSEvent
			input := test.ReadJSONFromFile(t, "../../test_event.json")
			if err := json.Unmarshal(input, &inputEvent); err != nil {
				t.Fatalf("could not unmarshal event: %v", err)
			}

			var payload []byte
			// p := make(map[string]interface{})
			// p["external_identifier"] = "abc-123"
			// payload, err := json.Marshal(&p)
			// if err != nil {
			// 	t.Fatalf("could not marshal payload: %v", err)
			// }

			if tc.name != "create" {
				mui := NewProcessor(getMockUpdateInvoker())
				err := mui.Process(&inputEvent)
				if msg := err.Error(); !strings.Contains(msg, tc.err) {
					t.Errorf("expected error %q, got: %q", tc.err, msg)
				}
				err = mui.startUpdate(tc.internalIdentifier, payload)
				if msg := err.Error(); !strings.Contains(msg, tc.err) {
					t.Errorf("expected error %q, got: %q", tc.err, msg)
				}
				return
			}
			mci := NewProcessor(getMockCreateInvoker())
			err := mci.Process(&inputEvent)
			if msg := err.Error(); !strings.Contains(msg, tc.err) {
				t.Errorf("expected error %q, got: %q", tc.err, msg)
			}
			err = mci.startCreate(tc.externalIdentifier, payload)
			if msg := err.Error(); !strings.Contains(msg, tc.err) {
				t.Errorf("expected error %q, got: %q", tc.err, msg)
			}
		})
	}
}
