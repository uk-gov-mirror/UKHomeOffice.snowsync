// Function receiver starts a DynamoDB session and hands over to package receiver.
package main

import (
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"

	"github.com/UKHomeOffice/snowsync/pkg/receiver"
)

var sess *session.Session
var ddb *dynamodb.DynamoDB

func init() {
	sess = session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))
	ddb = dynamodb.New(sess, &aws.Config{Region: aws.String(os.Getenv("AWS_REGION"))})
}

func handler(req *events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	return receiver.NewReceiver(ddb).Handle(req)
}

func main() {
	lambda.Start(handler)
}
