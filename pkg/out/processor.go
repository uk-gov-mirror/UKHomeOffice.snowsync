package out

import (
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

// DB implements db client methods
type DB interface {
	GetItem(context.Context, *dynamodb.GetItemInput, ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error)
	PutItem(context.Context, *dynamodb.PutItemInput, ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error)
	Query(context.Context, *dynamodb.QueryInput, ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error)
}

// Dynamo is a DB client
type Dynamo struct {
	DynamoDB DB
}

// Processor represents clients
type Processor struct {
	db Dynamo
}

func newProcessor(d Dynamo) *Processor {
	return &Processor{db: d}
}

func newDBClient() *Dynamo {
	cfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithRegion(os.Getenv("AWS_REGION")),
	)
	if err != nil {
		panic(fmt.Sprintf("unable to load SDK config: %v", err))
	}

	ddb := dynamodb.NewFromConfig(cfg)
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
