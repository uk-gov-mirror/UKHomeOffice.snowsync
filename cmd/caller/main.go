package main

import (
	"github.com/aws/aws-lambda-go/lambda"

	"github.com/UKHomeOffice/snowsync/internal/caller"
	"github.com/UKHomeOffice/snowsync/internal/client"
)

func handler(e client.Envelope) (string, error) {
	return caller.Call(e)
}

func main() {
	lambda.Start(handler)
}
