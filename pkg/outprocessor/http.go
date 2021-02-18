package outprocessor

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/lambda"
)

// create invokes the caller Lambda function to create a new ticket
func (f *Forwarder) create(pay Incident) (string, error) {

	// construct payload with SNow required headers
	dat := make(map[string]interface{})
	dat["messageid"] = "HO_SIAM_IN_REST_INC_POST_JSON_ACP_Incident_Create"
	dat["external_identifier"] = pay.ExtID
	// avoid repeating external identifier in payload
	pay.ExtID = ""
	dat["payload"] = pay

	newTicket, err := json.Marshal(dat)
	if err != nil {
		return "", fmt.Errorf("could not marshal creator payload: %v", err)
	}

	// invoke caller Lambda function
	input := &lambda.InvokeInput{
		FunctionName: aws.String(os.Getenv("CALLER_LAMBDA")),
		Payload:      newTicket,
	}

	out, err := f.Lambda.Invoke(input)
	if err != nil {
		return "", fmt.Errorf("could not invoke a create call: %v", err)
	}

	// check for and return internal identifier
	iid := strings.Trim(string(out.Payload), `", \`)
	if iid != "" {
		return iid, nil
	}
	return "", fmt.Errorf("no identifier in SNow response")
}

// update invokes the caller Lambda function to update an existing ticket
func (f *Forwarder) update(pay Incident) error {

	// construct payload with SNow required headers
	dat := make(map[string]interface{})
	dat["messageid"] = "HO_SIAM_IN_REST_INC_UPDATE_JSON_ACP_Incident_Update"
	dat["internal_identifier"] = pay.IntID
	// avoid repeating internal identifier in payload
	pay.IntID = ""
	dat["payload"] = pay

	ticketUpdate, err := json.Marshal(dat)
	if err != nil {
		return fmt.Errorf("could not marshal updater payload: %v", err)
	}

	// invoke caller Lambda function
	input := &lambda.InvokeInput{
		FunctionName: aws.String(os.Getenv("CALLER_LAMBDA")),
		Payload:      ticketUpdate,
	}

	_, err = f.Lambda.Invoke(input)
	if err != nil {
		return fmt.Errorf("could not invoke caller: %v", err)
	}
	return nil
}

func (f *Forwarder) progress(pay Incident) error {

	// construct payload with SNow required headers
	dat := make(map[string]interface{})
	dat["messageid"] = "HO_SIAM_IN_REST_INC_UPDATE_JSON_ACP_Incident_Update"
	dat["internal_identifier"] = pay.IntID
	// remove irrelevant keys from payload
	pay.IntID = ""
	pay.Comment = ""
	pay.Priority = ""

	if pay.Status == "6" {
		pay.Resolution = "done"
	}

	dat["payload"] = pay

	progressUpdate, err := json.Marshal(dat)
	if err != nil {
		return fmt.Errorf("could not marshal updater payload: %v", err)
	}

	// invoke caller Lambda function
	input := &lambda.InvokeInput{
		FunctionName: aws.String(os.Getenv("CALLER_LAMBDA")),
		Payload:      progressUpdate,
	}

	_, err = f.Lambda.Invoke(input)
	if err != nil {
		return fmt.Errorf("could not invoke caller: %v", err)
	}
	return nil
}
