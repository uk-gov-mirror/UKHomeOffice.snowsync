// Function processor starts a Lambda session and hands over to package processor.
package main

import (
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"

	"github.com/UKHomeOffice/snowsync/pkg/outprocessor"
)

func handler(sqsEvent *events.SQSEvent) error {
	return outprocessor.Process(sqsEvent)
}

func main() {
	lambda.Start(handler)
}
