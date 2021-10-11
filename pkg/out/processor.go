package out

import (
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbiface"
)

// DB implements db client methods
type DB interface {
	GetItem(*dynamodb.GetItemInput) (*dynamodb.GetItemOutput, error)
	PutItem(*dynamodb.PutItemInput) (*dynamodb.PutItemOutput, error)
	Query(*dynamodb.QueryInput) (*dynamodb.QueryOutput, error)
}

// Dynamo is a DB client
type Dynamo struct {
	DynamoDB dynamodbiface.DynamoDBAPI
}

// Processor represents clients
type Processor struct {
	db Dynamo
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

func process(inc *Incident) error {

	p := newProcessor(*newDBClient())

	// check if external id exists in DB, expect internal identifier in return
	partial, iid, err := p.db.checkPartial(inc)
	if err != nil {
		return fmt.Errorf("could not check partial item: %v", err)
	}

	// add internal identifier
	inc.IntID = iid

	// check if both external id and comment exist, expect internal identifier in return
	exact, err := p.db.checkExact(inc)
	if err != nil {
		return fmt.Errorf("could not check exact item: %v", err)
	}

	switch {
	case !exact && !partial:
		fmt.Println("creating new ticket...")
		// create ticket on SNOW
		iid, err := create(inc)
		if err != nil {
			return fmt.Errorf("could not create ticket: %v", err)
		}
		// add returned internal identifier
		inc.IntID = iid
		// create a new DB record
		err = p.db.writeItem(inc)
		if err != nil {
			return fmt.Errorf("could not put DB item: %v", err)
		}
		return nil
	case !exact && partial:
		fmt.Println("updating ticket with new comments...")
		// update DB with existing key
		err := p.db.writeItem(inc)
		if err != nil {
			return fmt.Errorf("could not update DB item: %v", err)
		}
		// remove irrelevant keys and update ticket on SNOW
		inc.Priority = ""
		inc.Description = ""
		err = update(inc)
		if err != nil {
			return fmt.Errorf("could not update ticket: %v", err)
		}
		return nil
	case exact:
		fmt.Println("no new comments, updating status only...")
		// update DB with existing key
		err := p.db.writeItem(inc)
		if err != nil {
			return fmt.Errorf("could not update DB item: %v", err)
		}
		// progress ticket on SNOW
		err = progress(inc)
		if err != nil {
			return fmt.Errorf("could not update ticket: %v", err)
		}
		return nil
	default:
		fmt.Printf("nothing to update, quitting!\n")
	}
	return nil
}
