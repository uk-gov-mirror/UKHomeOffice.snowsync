package main

import (
	"github.com/UKHomeOffice/snowsync/pkg/in"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

func handler(req *events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	return in.Handle(req)
}

func main() {
	lambda.Start(handler)
}
