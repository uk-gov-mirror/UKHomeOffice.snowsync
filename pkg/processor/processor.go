// Package processor receives a SQS event and invokes other functions to handle JSD webhooks.
package processor

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/lambda"
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

	dat := make(map[string]interface{})
	dat["messageid"] = "HO_SIAM_IN_REST_INC_POST_JSON_ACP_Incident_Create"
	dat["external_identifier"] = eid

	pld := make(map[string]interface{})

	err := json.Unmarshal([]byte(payload), &pld)
	if err != nil {
		fmt.Printf("failed to unmarshal creator payload: %v", err)
	}
	dat["payload"] = pld

	newTicket, err := json.Marshal(dat)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal creator payload: %v", err)
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
		fmt.Printf("failed to unmarshal dbputter payload: %v", err)
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

	dat := make(map[string]interface{})
	dat["messageid"] = "HO_SIAM_IN_REST_INC_UPDATE_JSON_ACP_Incident_Update"
	dat["internal_identifier"] = iid

	pld := make(map[string]interface{})

	err := json.Unmarshal([]byte(payload), &pld)
	if err != nil {
		fmt.Printf("failed to unmarshal updater payload: %v", err)
	}
	dat["payload"] = pld

	ticketUpdate, err := json.Marshal(dat)
	if err != nil {
		return fmt.Errorf("failed to marshal updater payload: %v", err)
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
		return fmt.Errorf("failed to unmarshal dbupdater payload: %v", err)
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

func (p *Processor) check(pld []byte) (*lambda.InvokeOutput, error) {

	// call checker lambda, expect external_identifier if it exists
	input := &lambda.InvokeInput{
		FunctionName: aws.String(os.Getenv("CHECKER_LAMBDA")),
		Payload:      pld,
	}

	out, err := p.inv.Invoke(input)
	if err != nil {
		return nil, fmt.Errorf("failed to invoke checker function: %v", err)
	}
	return out, nil
}

// Process processes individual SQS messages
func (p *Processor) Process(event *events.SQSEvent) error {

	for _, message := range event.Records {
		fmt.Printf("Processing message %s | %s\n", message.MessageId, message.Body)

		// note that we have to copy the slice and make a separate call per comment because 'comments' is
		// currently a string, not an array. this results in duplication and needs fixing by SNOW.
		var dat, copy map[string]interface{}
		var payload []byte
		err := json.Unmarshal([]byte(message.Body), &dat)
		if err != nil {
			return fmt.Errorf("failed to unmarshal SQS messsage: %v", err)
		}
		copy = dat
		comments, ok := dat["comments"].([]interface{})
		if !ok {
			payload, err = json.Marshal(&dat)
			if err != nil {
				return fmt.Errorf("failed to marshal checker payload: %v", err)
			}

			out, err := p.check(payload)
			if err != nil {
				return fmt.Errorf("failed to call checker: %v", err)
			}

			eid, iid, err := p.processCheckerResponse(out)
			if err != nil {
				return fmt.Errorf("failed to process checker response: %v", err)
			}

			if iid == "" {
				err = p.startCreate(eid, payload)
				if err != nil {
					return fmt.Errorf("failed to start creator process: %v", err)
				}
				continue
			}
			err = p.startUpdate(iid, payload)
			if err != nil {
				return fmt.Errorf("failed to start updater process: %v", err)
			}
			continue
		}
		for _, comment := range comments {
			copy["comments"] = fmt.Sprintf("%v", comment)
			payload, err = json.Marshal(&copy)
			if err != nil {
				return fmt.Errorf("failed to marshal checker payload: %v", err)
			}

			out, err := p.check(payload)
			if err != nil {
				return fmt.Errorf("failed to call checker: %v", err)
			}

			eid, iid, err := p.processCheckerResponse(out)
			if err != nil {
				return fmt.Errorf("failed to process checker response: %v", err)
			}

			if iid == "" {
				err = p.startCreate(eid, payload)
				if err != nil {
					return fmt.Errorf("failed to start creator process: %v", err)
				}
				continue
			}
			err = p.startUpdate(iid, payload)
			if err != nil {
				return fmt.Errorf("failed to start updater process: %v", err)
			}
		}
	}
	return nil
}
