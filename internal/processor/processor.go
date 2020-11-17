package processor

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/lambda"

	"github.com/UKHomeOffice/snowsync/internal/client"
)

// Invoker invokes another lambda
type Invoker interface {
	Invoke(*lambda.InvokeInput) (*lambda.InvokeOutput, error)
}

// Processor processes messages from queue
type Processor struct {
	inv Invoker
}

// NewProcessor returns a new Processor
func NewProcessor(i Invoker) *Processor {
	return &Processor{inv: i}
}

func (p *Processor) processCreateCall(eid string, payload []byte) (*lambda.InvokeOutput, error) {

	e := client.Envelope{
		MsgID:   "HO_SIAM_IN_REST_INC_POST_JSON_ACP_Incident_Create",
		ExtID:   eid,
		Payload: string(payload),
	}

	newTicket, err := json.Marshal(e)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal caller payload: %v", err)
	}

	input := &lambda.InvokeInput{
		FunctionName: aws.String(os.Getenv("CALLER_LAMBDA")),
		Payload:      newTicket,
	}

	output, err := p.inv.Invoke(input)
	if err != nil {
		return nil, fmt.Errorf("failed to invoke caller: %v", err)
	}

	return output, nil
}

func (p *Processor) processCreateRecord(payload []byte, io *lambda.InvokeOutput) error {

	// call dbputter lambda, adding SNOW identifier to payload
	dat := make(map[string]interface{})
	err := json.Unmarshal([]byte(payload), &dat)
	if err != nil {
		fmt.Printf("failed to unmarshal incoming payload: %v", err)
	}

	dat["internal_identifier"] = strings.Trim(string(io.Payload), `", \`)

	newItem, err := json.Marshal(dat)
	if err != nil {
		return fmt.Errorf("failed to marshal dbputter payload: %v", err)
	}

	input := &lambda.InvokeInput{
		FunctionName: aws.String(os.Getenv("DBPUTTER_LAMBDA")),
		Payload:      newItem,
	}

	_, err = p.inv.Invoke(input)
	if err != nil {
		return fmt.Errorf("failed to invoke dbputter: %v", err)
	}

	return nil
}

func (p *Processor) processUpdateCall(iid string, payload []byte) error {

	e := client.Envelope{
		MsgID:   "HO_SIAM_IN_REST_INC_UPDATE_JSON_ACP_Incident_Update",
		IntID:   iid,
		Payload: string(payload),
	}

	ticketUpdate, err := json.Marshal(e)
	if err != nil {
		return fmt.Errorf("failed to marshal caller payload: %v", err)
	}

	input := &lambda.InvokeInput{
		FunctionName: aws.String(os.Getenv("CALLER_LAMBDA")),
		Payload:      ticketUpdate,
	}

	_, err = p.inv.Invoke(input)
	if err != nil {
		return fmt.Errorf("failed to invoke caller: %v", err)
	}

	return nil
}

func (p *Processor) processUpdateRecord(payload []byte) error {

	var dat map[string]interface{}
	err := json.Unmarshal(payload, &dat)
	if err != nil {
		return fmt.Errorf("failed to unmarshal incoming payload: %v", err)
	}

	itemUpdate, err := json.Marshal(dat)
	if err != nil {
		return fmt.Errorf("failed to marshal dbupdater payload: %v", err)
	}

	input := &lambda.InvokeInput{
		FunctionName: aws.String(os.Getenv("DBUPDATER_LAMBDA")),
		Payload:      itemUpdate,
	}

	_, err = p.inv.Invoke(input)
	if err != nil {
		return fmt.Errorf("failed to invoke dbupdater: %v", err)
	}
	return nil
}

func (p *Processor) startCreate(eid string, payload []byte) error {
	fmt.Println("No SNOW id found, creating a new ticket...")
	out, err := p.processCreateCall(eid, payload)
	if err != nil {
		return fmt.Errorf("failed to invoke create caller: %v", err)
	}

	fmt.Println("No SNOW id found, creating a new record...")
	err = p.processCreateRecord(payload, out)
	if err != nil {
		return fmt.Errorf("failed to invoke dbputter: %v", err)
	}

	return nil
}

func (p *Processor) startUpdate(iid string, payload []byte) error {
	fmt.Println("A SNOW id exists, updating the existing ticket...")
	err := p.processUpdateCall(iid, payload)
	if err != nil {
		return fmt.Errorf("failed to invoke update caller: %v", err)
	}

	fmt.Println("A SNOW id exists, updating the existing db record...")
	err = p.processUpdateRecord(payload)
	if err != nil {
		return fmt.Errorf("failed to invoke dbupdater: %v", err)
	}
	return nil
}

func (p *Processor) processCheckerResponse(io *lambda.InvokeOutput) (string, string, error) {

	type result struct {
		ExtID string `json:"external_identifier"`
		IntID string `json:"internal_identifier"`
	}

	var res result
	err := json.Unmarshal(io.Payload, &res)
	if err != nil {
		return "", "", fmt.Errorf("failed to unmarshal checker response: %v", err)
	}
	eid := res.ExtID
	if iid := res.IntID; iid != "" {
		return eid, iid, nil
	}
	return eid, "", nil
}

// Process processes individual SQS messages
func (p *Processor) Process(event *events.SQSEvent) error {

	for _, message := range event.Records {
		fmt.Printf("Processing message %s | %s", message.MessageId, message.Body)
		raw := json.RawMessage(message.Body)
		payload, err := json.Marshal(&raw)
		if err != nil {
			return fmt.Errorf("failed to marshal checker payload: %v", err)
		}

		// call checker lambda, expect external_identifier if it exists
		input := &lambda.InvokeInput{
			FunctionName: aws.String(os.Getenv("CHECKER_LAMBDA")),
			Payload:      payload,
		}

		res, err := p.inv.Invoke(input)
		if err != nil {
			return fmt.Errorf("failed to invoke checker function: %v", err)
		}

		eid, iid, err := p.processCheckerResponse(res)
		if err != nil {
			return fmt.Errorf("failed to process checker response: %v", err)
		}
		if iid != "" {
			p.startUpdate(iid, payload)
			return nil
		}
		p.startCreate(eid, payload)
	}
	return nil
}
