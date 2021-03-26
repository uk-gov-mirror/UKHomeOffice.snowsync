// Package outprocessor receives a SQS event, queries and writes to DynamoDB,
// and invokes other Lambda functions.
package outprocessor

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbiface"
	"github.com/aws/aws-sdk-go/service/lambda"
	"github.com/aws/aws-sdk-go/service/lambda/lambdaiface"
)

// App implements db and lambda client methods
type App interface {
	GetItem(*dynamodb.GetItemInput) (*dynamodb.GetItemOutput, error)
	Invoke(*lambda.InvokeInput) (*lambda.InvokeOutput, error)
	PutItem(*dynamodb.PutItemInput) (*dynamodb.PutItemOutput, error)
	Query(*dynamodb.QueryInput) (*dynamodb.QueryOutput, error)
	UpdateItem(*dynamodb.UpdateItemInput) (*dynamodb.UpdateItemOutput, error)
}

// Dynamo is a DB client
type Dynamo struct {
	DynamoDB dynamodbiface.DynamoDBAPI
}

// Forwarder is a Lambda client
type Forwarder struct {
	Lambda lambdaiface.LambdaAPI
}

// Processor represents clients
type Processor struct {
	lm Forwarder
	db Dynamo
}

// Incident represents outbound messsages
type Incident struct {
	Comment     string `json:"comments,omitempty"`
	CommentID   string `json:"comment_sysid,omitempty"`
	Description string `json:"description,omitempty"`
	ExtID       string `json:"external_identifier,omitempty"`
	IntID       string `json:"internal_identifier,omitempty"`
	MsgID       string `json:"messageid,omitempty"`
	Priority    string `json:"priority,omitempty"`
	Resolution  string `json:"resolution_code,omitempty"`
	Service     string `json:"business_service,omitempty"`
	Status      string `json:"state,omitempty"`
	Summary     string `json:"title,omitempty"`
}

func newProcessor(f Forwarder, d Dynamo) *Processor {
	return &Processor{lm: f, db: d}
}

func newDBClient() *Dynamo {
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))
	ddb := dynamodb.New(sess, &aws.Config{Region: aws.String(os.Getenv("AWS_REGION"))})
	return &Dynamo{DynamoDB: ddb}
}

func newForwarder() *Forwarder {
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))
	alb := lambda.New(sess, &aws.Config{Region: aws.String(os.Getenv("AWS_REGION"))})
	return &Forwarder{Lambda: alb}
}

// Process loops over SQS events
func Process(event *events.SQSEvent) error {

	p := newProcessor(*newForwarder(), *newDBClient())

	for _, message := range event.Records {
		fmt.Printf("Processing message %s | %s\n", message.MessageId, message.Body)

		err := p.subProcess(&message)
		if err != nil {
			return fmt.Errorf("could not call subprocessor: %v", err)
		}
	}
	return nil
}

// subProcess processes individual SQS messages
func (p Processor) subProcess(m *events.SQSMessage) error {
	// convert inbound SQS message to custom payload type
	var pay Incident
	err := json.Unmarshal([]byte(m.Body), &pay)
	if err != nil {
		return fmt.Errorf("could not unmarshal SQS message: %v", err)
	}

	// check if external id exists in DB, expect internal identifier in return
	partial, iid, err := p.db.checkPartial(pay)
	if err != nil {
		return fmt.Errorf("could not check partial item: %v", err)
	}

	// add internal identifier
	pay.IntID = iid

	// check if both external id and comment exist, expect internal identifier in return
	exact, iid, err := p.db.checkExact(pay)
	if err != nil {
		return fmt.Errorf("could not check exact item: %v", err)
	}

	switch {
	case !exact && !partial:
		fmt.Println("creating new ticket...")
		// create ticket on SNow
		iid, err := p.lm.create(pay)
		if err != nil {
			return fmt.Errorf("could not create ticket: %v", err)
		}
		// add returned internal identifier
		pay.IntID = iid
		// create a new DB record
		err = p.db.writeItem(pay)
		if err != nil {
			return fmt.Errorf("could not put DB item: %v", err)
		}
		return nil
	case !exact && partial:
		fmt.Println("updating ticket with new comments...")
		// update DB with existing key
		err := p.db.writeItem(pay)
		if err != nil {
			return fmt.Errorf("could not update DB item: %v", err)
		}
		// remove irrelevant keys and update ticket on SNow
		pay.Priority = ""
		pay.Description = ""
		err = p.lm.update(pay)
		if err != nil {
			return fmt.Errorf("could not update ticket: %v", err)
		}
		return nil
	case exact:
		fmt.Println("no new comments, updating status only...")
		// update DB with existing key
		err := p.db.writeItem(pay)
		if err != nil {
			return fmt.Errorf("could not update DB item: %v", err)
		}
		// progress ticket on SNow
		err = p.lm.progress(pay)
		if err != nil {
			return fmt.Errorf("could not update ticket: %v", err)
		}
		return nil
	default:
		fmt.Printf("nothing to update, quitting!\n")
	}
	return nil
}
