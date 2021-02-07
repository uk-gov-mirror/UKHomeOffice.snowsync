// Package inapi receives a webhook from SNOW, parses its payload and calls inprocessor
package inapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/lambda"
	"github.com/aws/aws-sdk-go/service/lambda/lambdaiface"
)

// App defines client methods
type App interface {
	Invoke(*lambda.InvokeInput) (*lambda.InvokeOutput, error)
}

// Forwarder is a lambda client
type Forwarder struct {
	Lambda lambdaiface.LambdaAPI
}

// Handler is our API
type Handler struct {
	fwd Forwarder
}

// NewHandler returns a new Handler
func NewHandler(f Forwarder) *Handler {
	return &Handler{fwd: f}
}

func newForwarder() *Forwarder {
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))
	alb := lambda.New(sess, &aws.Config{Region: aws.String(os.Getenv("AWS_REGION"))})
	return &Forwarder{Lambda: alb}
}

// Handle deals with the incoming request
func Handle(request *events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	h := NewHandler(*newForwarder())

	inc, err := parseIncident(request.Body)
	if err != nil {
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusBadRequest,
			Body:       err.Error(),
		}, nil
	}

	out, err := json.Marshal(&inc)
	if err != nil {
		fmt.Println(err)
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusInternalServerError,
			Body:       err.Error(),
		}, nil
	}

	res, err := h.fwd.forward(out)
	if err != nil {
		fmt.Println(err)
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusInternalServerError,
			Body:       err.Error(),
		}, nil
	}

	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Body:       fmt.Sprintf("incident %v updated\n", res),
	}, nil
}

func (f *Forwarder) forward(b []byte) (string, error) {

	input := &lambda.InvokeInput{
		FunctionName: aws.String(os.Getenv("PROCESSOR_LAMBDA")),
		Payload:      b,
	}

	out, err := f.Lambda.Invoke(input)
	if err != nil {
		return "", fmt.Errorf("could not call inprocessor: %v", err)
	}

	return string(out.Payload), nil

}
