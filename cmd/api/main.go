// Function api starts a SQS session and hands over to package api.
package main

import (
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"

	"github.com/UKHomeOffice/snowsync/pkg/api"
)

var sess *session.Session
var esqs *sqs.SQS

func init() {
	sess = session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))
	esqs = sqs.New(sess, &aws.Config{Region: aws.String(os.Getenv("AWS_REGION"))})
}

func handler(req *events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	return api.NewHandler(esqs).Handle(req)
}

func main() {
	lambda.Start(handler)
}
