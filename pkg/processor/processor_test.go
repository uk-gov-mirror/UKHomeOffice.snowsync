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

type mockCreatorInvoker struct {
	lambdaiface.LambdaAPI
	err error
}

type mockUpdaterInvoker struct {
	lambdaiface.LambdaAPI
	err error
}

func getMockCreatorInvoker() lambdaiface.LambdaAPI {
	return &mockCreatorInvoker{}
}

func getMockUpdaterInvoker() lambdaiface.LambdaAPI {
	return &mockUpdaterInvoker{}
}

func (m *mockCreatorInvoker) check(in *lambda.InvokeInput) (*lambda.InvokeOutput, error) {
	var out lambda.InvokeOutput
	if string(in.Payload) == `{"external_identifier":"abc-123","internal_identifier":""}` {
		out.Payload = []byte(`"inc-124"`)
		return &out, nil
	}
	out.Payload = []byte(`""`)
	return &out, nil
}

func (m *mockCreatorInvoker) Invoke(in *lambda.InvokeInput) (*lambda.InvokeOutput, error) {
	var out lambda.InvokeOutput
	if string(in.Payload) == "abc-123" {
		out.Payload = []byte(`"inc-124"`)
		return &out, nil
	}
	return &out, nil
}

func (m *mockUpdaterInvoker) check(in *lambda.InvokeInput) (*lambda.InvokeOutput, error) {
	var out lambda.InvokeOutput
	out.Payload = []byte(`""`)
	return &out, nil
}

func (m *mockUpdaterInvoker) Invoke(in *lambda.InvokeInput) (*lambda.InvokeOutput, error) {
	var out lambda.InvokeOutput
	return &out, nil
}

func TestProcessor(t *testing.T) {

	tt := []struct {
		name               string
		externalIdentifier string
		internalIdentifier string
		err                string
	}{
		{name: "create", externalIdentifier: "abc-123"},
		{name: "create", externalIdentifier: "", err: "failed to unmarshal checker response: unexpected end of JSON input"},
		{name: "update", externalIdentifier: "abc-123", internalIdentifier: "inc-124"},
		{name: "update", externalIdentifier: "", internalIdentifier: "", err: "failed to unmarshal checker response: unexpected end of JSON input"},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {

			var inputEvent events.SQSEvent
			file := test.ReadJSONFromFile(t, "../../test_event.json")
			if err := json.Unmarshal(file, &inputEvent); err != nil {
				t.Fatalf("could not unmarshal event: %v", err)
			}

			p := make(map[string]interface{})
			p["external_identifier"] = tc.externalIdentifier
			p["internal_identifier"] = tc.internalIdentifier
			payload, err := json.Marshal(&p)
			if err != nil {
				t.Fatalf("could not marshal payload: %v", err)
			}

			if tc.name != "create" {
				mui := NewProcessor(getMockUpdaterInvoker())
				err := mui.Process(&inputEvent)
				if msg := err.Error(); !strings.Contains(msg, tc.err) {
					t.Errorf("expected error %q, got: %q", tc.err, msg)
				}

				invOut, err := mui.check(payload)
				if err != nil {
					if msg := err.Error(); !strings.Contains(msg, tc.err) {
						t.Errorf("expected error %q, got: %q", tc.err, msg)
					}
				}

				_, iid, err := mui.processCheckerResponse(invOut)
				if err != nil {
					if msg := err.Error(); !strings.Contains(msg, tc.err) {
						t.Errorf("expected error %q, got: %q", tc.err, msg)
					}
				}

				if iid != "" && iid != tc.internalIdentifier {
					t.Errorf("expected eid: %v, got %v", tc.internalIdentifier, iid)
				}

				err = mui.startUpdate(tc.internalIdentifier, payload)
				if err != nil {
					if msg := err.Error(); !strings.Contains(msg, tc.err) {
						t.Errorf("expected error %q, got: %q", tc.err, msg)
					}
				}
				return
			}
			mci := NewProcessor(getMockCreatorInvoker())
			err = mci.Process(&inputEvent)
			if msg := err.Error(); !strings.Contains(msg, tc.err) {
				t.Errorf("expected error %q, got: %q", tc.err, msg)
			}

			invOut, err := mci.check(payload)
			if err != nil {
				if msg := err.Error(); !strings.Contains(msg, tc.err) {
					t.Errorf("expected error %q, got: %q", tc.err, msg)
				}
			}

			eid, _, err := mci.processCheckerResponse(invOut)
			if err != nil {
				if msg := err.Error(); !strings.Contains(msg, tc.err) {
					t.Errorf("expected error %q, got: %q", tc.err, msg)
				}
			}

			if eid != "" && eid != tc.externalIdentifier {
				t.Errorf("expected iid: %v, got %v", tc.externalIdentifier, eid)
			}

			err = mci.startCreate(tc.externalIdentifier, payload)
			if err != nil {
				if msg := err.Error(); !strings.Contains(msg, tc.err) {
					t.Errorf("expected error %q, got: %q", tc.err, msg)
				}
			}
		})
	}
}
