package api

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/aws/aws-sdk-go/service/sqs/sqsiface"
	"github.com/tidwall/gjson"
)

type mockSQS struct {
	sqsiface.SQSAPI
	err error
}

func getMockSQS() sqsiface.SQSAPI {
	return &mockSQS{}
}

func (ms *mockSQS) SendMessage(*sqs.SendMessageInput) (*sqs.SendMessageOutput, error) {
	return &sqs.SendMessageOutput{}, nil
}

// setEnv sets test envars
func setEnv() {

	os.Setenv("CLUSTER_FIELD", "issue.fields.cluster")
	os.Setenv("COMPONENT_FIELD", "issue.fields.component")
	os.Setenv("COMMENT_FIELD", "issue.fields.comment.comments")
	os.Setenv("DESCRIPTION_FIELD", "issue.fields.description")
	os.Setenv("ISSUE_ID_FIELD", "issue.key")
	os.Setenv("PRIORITY_FIELD", "issue.fields.priority")
	os.Setenv("STATUS_FIELD", "issue.fields.status.name")
	os.Setenv("SUMMARY_FIELD", "issue.fields.summary")
}

//  getMsg gets  test input
func getMsg(p int) (string, error) {

	body, err := ioutil.ReadFile("../../test_payloads.json")
	if err != nil {
		return "", err
	}

	path := fmt.Sprintf("cases.%v", p)
	res := gjson.GetManyBytes(body, path)

	return res[0].Raw, nil
}

// TestIncident tests incident parse and publish methods
func TestIncident(t *testing.T) {

	tt := []struct {
		name          string
		input         int
		issueID       string
		commentID     string
		commentAuthor string
		commentBody   string
		component     string
		cluster       string
		description   string
		priority      string
		status        string
		summary       string
		err           string
	}{
		{name: "happy", input: 0, issueID: "abc-1", status: "open", summary: "system down",
			description: "not responding for 10 mins", priority: "P1", component: "system",
			cluster: "prod", commentID: "1", commentAuthor: "bob", commentBody: "first comment"},
		{name: "unhappy", input: 1, err: "missing value in payload"},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {

			setEnv()
			msg, err := getMsg(tc.input)
			if err != nil {
				t.Fatalf("could not get message: %v", err)
			}

			ia, err := parseIncident(msg)
			if err != nil {
				if e := err.Error(); !strings.Contains(e, tc.err) {
					t.Errorf("expected error %q, got: %q", tc.err, e)
				}
				return
			}

			if ia.incident.Description != tc.description {
				t.Errorf("expected %v, got %v", tc.description, ia.incident.Description)
			}
			if ia.incident.Identifier != tc.issueID {
				t.Errorf("expected %v, got %v", tc.issueID, ia.incident.Identifier)
			}
			if ia.incident.Cluster != tc.cluster {
				t.Errorf("expected %v, got %v", tc.cluster, ia.incident.Cluster)
			}
			if ia.incident.Component != tc.component {
				t.Errorf("expected %v, got %v", tc.status, ia.incident.Status)
			}
			if ia.incident.Priority != tc.priority {
				t.Errorf("expected %v, got %v", tc.priority, ia.incident.Priority)
			}
			if ia.incident.Status != tc.status {
				t.Errorf("expected %v, got %v", tc.status, ia.incident.Status)
			}
			if ia.incident.Summary != tc.summary {
				t.Errorf("expected %v, got %v", tc.summary, ia.incident.Summary)
			}
			if ia.incident.Comments[0].ID != tc.commentID {
				t.Errorf("expected %v, got %v", tc.commentID, ia.incident.Comments[0].ID)
			}
			if ia.incident.Comments[0].Author != tc.commentAuthor {
				t.Errorf("expected %v, got %v", tc.commentAuthor, ia.incident.Comments[0].Author)
			}
			if ia.incident.Comments[0].Body != tc.commentBody {
				t.Errorf("expected %v, got %v", tc.commentBody, ia.incident.Comments[0].Body)
			}

			ia.sqs = getMockSQS()
			err = ia.publish()
			if err != nil {
				if e := err.Error(); !strings.Contains(e, tc.err) {
					t.Errorf("expected error %q, got: %q", tc.err, e)
				}
			}
		})
	}
}
