package main

import (
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"

	"github.com/UKHomeOffice/snowsync/pkg/out"
)

func handler(req *events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	return out.Handle(req)
}

func main() {
	lambda.Start(handler)
}
