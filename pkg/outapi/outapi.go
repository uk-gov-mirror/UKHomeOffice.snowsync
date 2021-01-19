// Package outapi receives a webhook from JSD, parses its payload and writes it to SQS.
package outapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sqs"
)

// Ticket is an abstraction for an incident or change
type Ticket interface {
	publish() error
}

// Messenger is an abstraction for a SQS client
type Messenger interface {
	SendMessage(*sqs.SendMessageInput) (*sqs.SendMessageOutput, error)
}

// Handler respresents the handler type
type Handler struct {
	mgr Messenger
}

// ticketUpdate is an event
type ticketUpdate struct {
	// placeholder for future change integration
	//change *Change
	incident *Incident
	sqs      Messenger
}

// NewHandler returns a new Handler
func NewHandler(m Messenger) *Handler {
	return &Handler{mgr: m}
}

// parseTicket parses an incident or change
func (h *Handler) parseTicket(input string) (Ticket, error) {

	ia, err := parseIncident(input)
	if err == nil {
		ia.sqs = h.mgr
		return ia, err
	}

	// placeholder for future change integration
	// ca, err := parseChange(input)
	// if err == nil {
	// 	ca.sqs = h.mgr
	// 	return ca, err
	// }

	return nil, fmt.Errorf("could not parse the ticket: %v", err)
}

// publish writes an update to SQS
func (tu *ticketUpdate) publish() error {

	sm, err := json.Marshal(&tu.incident)
	if err != nil {
		return fmt.Errorf("could not marshal SQS payload: %s", err)
	}

	in := sqs.SendMessageInput{
		MessageBody: aws.String(string(sm)),
		QueueUrl:    aws.String(os.Getenv("QUEUE_URL")),
	}

	_, err = tu.sqs.SendMessage(&in)
	if err != nil {
		return fmt.Errorf("could not publish incident: %s", err)
	}

	return nil

}

// Handle deals with the incoming request
func (h *Handler) Handle(request *events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	tk, err := h.parseTicket(request.Body)
	if err != nil {
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusBadRequest,
			Body:       err.Error(),
		}, nil
	}

	err = tk.publish()
	if err != nil {
		fmt.Println(err)
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusInternalServerError,
			Body:       err.Error(),
		}, nil
	}

	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
	}, nil
}
