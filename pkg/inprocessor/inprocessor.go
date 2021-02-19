// Package inprocessor receives a SQS event, queries and writes to DynamoDB
// and calls JSD
package inprocessor

import (
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbiface"
)

// App defines client methods
type App interface {
	GetItem(*dynamodb.GetItemInput) (*dynamodb.GetItemOutput, error)
	PutItem(*dynamodb.PutItemInput) (*dynamodb.PutItemOutput, error)
	Query(*dynamodb.QueryInput) (*dynamodb.QueryOutput, error)
	UpdateItem(*dynamodb.UpdateItemInput) (*dynamodb.UpdateItemOutput, error)
}

// Dynamo is a DB client
type Dynamo struct {
	DynamoDB dynamodbiface.DynamoDBAPI
}

// Processor can implement client methods
type Processor struct {
	db Dynamo
}

// Incident is a type of ticket
type Incident struct {
	Comment      string `json:"comment,omitempty"`
	CommentID    string `json:"comment_sysid,omitempty"`
	IntComment   string `json:"internal_comment,omitempty"`
	IntCommentID string `json:"internal_comment_sysid,omitempty"`
	Description  string `json:"description,omitempty"`
	ExtID        string `json:"external_identifier,omitempty"`
	IntID        string `json:"internal_identifier,omitempty"`
	Priority     string `json:"priority,omitempty"`
	Reporter     string `json:"reporter_name,omitempty"`
	Status       string `json:"status,omitempty"`
	Summary      string `json:"summary,omitempty"`
}

// Values make up the JSD payload
type Values struct {
	Comment     string      `json:"comment,omitempty"`
	Description string      `json:"description,omitempty"`
	Summary     string      `json:"summary,omitempty"`
	Priority    *priority   `json:"priority,omitempty"`
	Transition  *transition `json:"transition,omitempty"`
}

type priority struct {
	Name string `json:"name,omitempty"`
}

type transition struct {
	ID string `json:"id,omitempty"`
}

func newProcessor(d Dynamo) *Processor {
	return &Processor{db: d}
}

func newDBClient() *Dynamo {
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))
	ddb := dynamodb.New(sess, &aws.Config{Region: aws.String(os.Getenv("AWS_REGION"))})
	return &Dynamo{DynamoDB: ddb}
}

func getEnv() (string, string, string, error) {

	user, ok := os.LookupEnv("ADMIN_USER")
	if !ok {
		return "", "", "", fmt.Errorf("missing username")
	}

	pass, ok := os.LookupEnv("ADMIN_PASS")
	if !ok {
		return "", "", "", fmt.Errorf("missing password")
	}

	base, ok := os.LookupEnv("JSD_URL")
	if !ok {
		return "", "", "", fmt.Errorf("missing JSD URL")
	}

	return user, pass, base, nil
}

// Process applies workflow logic
func Process(in Incident) (string, error) {

	p := newProcessor(*newDBClient())

	// check if internal id exists in DB,
	// expect external identifier in return
	partial, eid, err := p.db.checkPartial(in)
	if err != nil {
		return "", fmt.Errorf("could not check partial item: %v", err)
	}

	//add identifier
	in.ExtID = eid

	// check if both internal id and comment id exist in DB,
	// expect external identifier in return
	exact, eid, err := p.db.checkExact(in)
	if err != nil {
		return "", fmt.Errorf("could not check exact item: %v", err)
	}

	switch {
	case !exact && !partial:
		fmt.Println("creating new ticket...")
		// create ticket on JSD
		eid, err := p.create(in)
		if err != nil {
			return "", fmt.Errorf("could not create ticket: %v", err)
		}
		// add returned external identifier
		in.ExtID = eid
		// create a new DB record
		err = p.db.writeItem(in)
		if err != nil {
			return "", fmt.Errorf("could not put DB item: %v", err)
		}
		return eid, nil
	case !exact && partial:
		fmt.Println("updating ticket with new comments...")
		// update ticket on SNow
		eid, err = p.update(in)
		if err != nil {
			return "", fmt.Errorf("could not update ticket: %v", err)
		}
		// update DB with existing key
		err := p.db.writeItem(in)
		if err != nil {
			return "", fmt.Errorf("could not update DB item: %v", err)
		}

		// this is a workaround
		_, err = p.progress(in)
		if err != nil {
			return "", fmt.Errorf("could not update ticket: %v", err)
		}
		return eid, nil
	case exact:
		fmt.Println("no new comments, updating status only...")
		// update DB with existing key
		err := p.db.writeItem(in)
		if err != nil {
			return "", fmt.Errorf("could not update DB item: %v", err)
		}
		// remove comments and update ticket
		in.Comment = ""
		eid, err = p.progress(in)
		if err != nil {
			return "", fmt.Errorf("could not update ticket: %v", err)
		}
		return eid, nil
	default:
		fmt.Printf("nothing to update, quitting!\n")
	}
	return "", nil
}
