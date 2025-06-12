package in

import (
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

// DB defines client methods
type DB interface {
	GetItem(context.Context, *dynamodb.GetItemInput, ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error)
	PutItem(context.Context, *dynamodb.PutItemInput, ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error)
	Query(context.Context, *dynamodb.QueryInput, ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error)
}

// Dynamo is a DB client
type Dynamo struct {
	DynamoDB DB
}

// Processor can implement client methods
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

func process(inc *Incident) (string, error) {

	p := newProcessor(*newDBClient())

	// check if internal id exists in DB, expect external identifier in return
	partial, eid, err := p.db.checkPartial(inc)
	if err != nil {
		return "", fmt.Errorf("could not check partial item: %v", err)
	}

	//add external identifier
	inc.ExtID = eid
	// check if both internal id and comment id exist in DB, expect external identifier in return
	exact, err := p.db.checkExact(inc)
	if err != nil {
		return "", fmt.Errorf("could not check exact item: %v", err)
	}

	switch {
	case !exact && !partial:
		fmt.Println("creating new ticket...")
		// create ticket on JSD
		eid, err := p.create(inc)
		if err != nil {
			return "", fmt.Errorf("could not create ticket: %v", err)
		}
		// add returned external identifier
		inc.ExtID = eid
		// create a new DB record
		err = p.db.writeItem(inc)
		if err != nil {
			return "", fmt.Errorf("could not put DB item: %v", err)
		}
		return eid, nil
	case !exact && partial:
		fmt.Println("updating ticket with new comments...")
		// update ticket on SNOW
		eid, err = p.update(inc)
		if err != nil {
			return "", fmt.Errorf("could not update ticket: %v", err)
		}
		// update DB with existing key
		err := p.db.writeItem(inc)
		if err != nil {
			return "", fmt.Errorf("could not update DB item: %v", err)
		}
		err = p.setPriority(inc)
		if err != nil {
			return "", fmt.Errorf("could not set priority: %v", err)
		}
		err = p.setStatus(inc)
		if err != nil {
			return "", fmt.Errorf("could not update ticket: %v", err)
		}
		return eid, nil
	case exact:
		fmt.Println("no new comments, updating status only...")
		// update DB with existing key
		err := p.db.writeItem(inc)
		if err != nil {
			return "", fmt.Errorf("could not update DB item: %v", err)
		}
		// remove comments and update ticket
		inc.Comment = ""
		err = p.setStatus(inc)
		if err != nil {
			return "", fmt.Errorf("could not update ticket: %v", err)
		}
		return eid, nil
	default:
		fmt.Printf("nothing to update, quitting!\n")
	}
	return "", nil
}
