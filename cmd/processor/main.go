package main

import (
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"

	"github.com/UKHomeOffice/snowsync/internal/processor"
	service "github.com/aws/aws-sdk-go/service/lambda"
)

var sess *session.Session
var svc *service.Lambda

func init() {
	sess = session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))
	svc = service.New(sess, &aws.Config{Region: aws.String(os.Getenv("AWS_REGION"))})
}

func handler(sqsEvent *events.SQSEvent) error {
	return processor.NewProcessor(svc).Process(sqsEvent)
}

func main() {
	lambda.Start(handler)
}
