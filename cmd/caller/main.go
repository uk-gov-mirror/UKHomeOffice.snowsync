// Function caller hands over to package caller.
package main

import (
	"github.com/aws/aws-lambda-go/lambda"

	"github.com/UKHomeOffice/snowsync/pkg/caller"
)

func handler(ms map[string]interface{}) (string, error) {
	return caller.CallSNOW(ms)
}

func main() {
	lambda.Start(handler)
}
