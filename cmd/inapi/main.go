// Function receiver starts a DynamoDB session and hands over to package receiver.
package main

import (
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"

	"github.com/UKHomeOffice/snowsync/pkg/inapi"
)

func handler(req *events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	return inapi.Handle(req)
}

func main() {
	lambda.Start(handler)
}
