package receiver

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/aws/aws-lambda-go/events"
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

func TestHandle(t *testing.T) {

	tt := []struct {
		name              string
		supplierReference string
		commentID         string
		commentAuthor     string
		commentBody       string
		err               string
	}{
		{name: "happy", supplierReference: "abc-123", commentID: "2", commentAuthor: "alice", commentBody: "second comment"},
		{name: "unhappy", err: "failed to update db: failed to update db: "},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {

			testSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

				if tc.err != "" {
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				ct := r.Header.Get("Content-Type")
				if ct != "application/json" {
					t.Errorf("wrong content type: %v", ct)
				}

				sa := r.Header.Get("Authorization")
				if sa != "Basic Zm9vOmJhcg==" {
					t.Errorf("wrong auth header: %v", sa)
				}

				body, err := ioutil.ReadAll(r.Body)
				if err != nil {
					t.Fatalf("could not read request body: %v", sa)
				}

				ts := `{"body":"second comment"}`

				if string(body) != ts {
					t.Errorf("expected %v, got %v", ts, string(body))
				}

				w.WriteHeader(http.StatusOK)
			}))

			os.Setenv("TABLE_NAME", "table")
			os.Setenv("JSD_URL", testSrv.URL)
			os.Setenv("ADMIN_USER", "foo")
			os.Setenv("ADMIN_PASS", "bar")

			rec := NewReceiver(&mockDynamoDB{})

			msg := struct {
				SupplierReference string `json:"supplier_reference,omitempty"`
				Comments          string `json:"comments,omitempty"`
			}{
				SupplierReference: tc.supplierReference,
				Comments:          tc.commentBody,
			}

			p, err := json.Marshal(msg)
			if err != nil {
				t.Fatalf("could not make incoming payload: %v", err)
			}

			req := events.APIGatewayProxyRequest{
				Path: "/",
				Body: string(p),
			}

			res, err := rec.Handle(&req)
			if err != nil {
				t.Fatalf("could not call Handle: %v", err)
			}

			if tc.err == "" {
				if res.StatusCode != http.StatusOK {
					t.Errorf("expected status OK, got %v", res.StatusCode)
				}
			}
			if msg := res.Body; msg != tc.err {
				t.Errorf("expected error %q, got: %q", tc.err, msg)
			}
		})
	}
}
