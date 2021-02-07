// Function processor starts a Lambda session and hands over to package processor.
package main

import (
	"github.com/aws/aws-lambda-go/lambda"

	"github.com/UKHomeOffice/snowsync/pkg/inprocessor"
)

func handler(i inprocessor.Incident) (string, error) {

	return inprocessor.Process(i)
}

func main() {
	lambda.Start(handler)
}
