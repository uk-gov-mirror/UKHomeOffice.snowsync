// Package inapi receives a webhook from SNOW, parses its payload and calls inprocessor
package inapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

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

	fmt.Printf("debug into inapi %v", request.Body)

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

	msg := struct {
		ExtID string `json:"external_identifier,omitempty"`
	}{
		ExtID: strings.Trim(res, `", \`),
	}

	bmsg, err := json.Marshal(msg)
	if err != nil {
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusInternalServerError,
			Body:       err.Error(),
		}, nil
	}

	//fmt.Printf("debug msg %v\n", string(bmsg))

	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Body:       string(bmsg),
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
