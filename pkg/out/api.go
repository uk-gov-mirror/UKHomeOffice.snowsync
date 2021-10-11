package out

import (
	"fmt"
	"net/http"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/tidwall/gjson"
)

// Incident is a type of ticket
type Incident struct {
	Comment     string `json:"comments,omitempty"`
	CommentID   string `json:"comment_sysid,omitempty"`
	Description string `json:"description,omitempty"`
	ExtID       string `json:"external_identifier,omitempty"`
	Identifier  string `json:"id,omitempty"`
	IntID       string `json:"internal_identifier,omitempty"`
	Priority    string `json:"priority,omitempty"`
	Resolution  string `json:"resolution_code,omitempty"`
	Status      string `json:"state,omitempty"`
	Service     string `json:"business_service,omitempty"`
	Summary     string `json:"title,omitempty"`
}

// newIncident initialises an Incident
func newIncident() *Incident {
	return &Incident{}
}

// checkVars checks incoming payload has the required field values
func checkIncidentVars(input string) error {

	vars := []string{
		"DESCRIPTION_FIELD",
		"ISSUE_ID_FIELD",
		"PRIORITY_FIELD",
		"STATUS_FIELD",
		"SUMMARY_FIELD",
	}

	for _, v := range vars {
		field, ok := os.LookupEnv(v)
		if !ok {
			return fmt.Errorf("missing environment variable: %v", v)
		}
		value := gjson.Get(input, field)
		if !value.Exists() {
			return fmt.Errorf("missing value in payload: %v", field)
		}
	}
	return nil
}

// parseIncident gets values from an inbound incident
func parseIncident(input string) (*Incident, error) {

	err := checkIncidentVars(input)
	if err != nil {
		return nil, err
	}

	i := newIncident()

	i.Comment = gjson.Get(input, os.Getenv("COMMENT_FIELD")).Str
	i.CommentID = gjson.Get(input, os.Getenv("COMMENT_ID_FIELD")).Str
	i.Description = gjson.Get(input, os.Getenv("DESCRIPTION_FIELD")).Str
	i.ExtID = gjson.Get(input, os.Getenv("ISSUE_ID_FIELD")).Str
	i.IntID = gjson.Get(input, os.Getenv("SNOW_ID_FIELD")).Str
	i.Priority = gjson.Get(input, os.Getenv("PRIORITY_FIELD")).Str
	i.Service = gjson.Get(input, os.Getenv("SERVICE_FIELD")).Str
	i.Status = gjson.Get(input, os.Getenv("STATUS_FIELD")).Str
	i.Summary = gjson.Get(input, os.Getenv("SUMMARY_FIELD")).Str

	// assign to an organisation in SNOW
	switch i.Service {
	case "9":
		i.Service = "Cyclamen IT Platform Local"
	case "58":
		i.Service = "I-LEAP"
	case "45":
		i.Service = "Semaphore"
	default:
		i.Service = "AWS ACP"
	}

	// initialise comment id if nil as it will be used as sort key & transform
	if i.CommentID == "" {
		i.CommentID = "0"
	}

	// transform comments to fit target schema
	commentAuthor := gjson.Get(input, os.Getenv("COMMENT_AUTHOR_FIELD")).Str
	// TODO - check if this step is still needed.
	if commentAuthor == "ServiceNow" {
		fmt.Println("ignoring comment left on JSD by ServiceNow service account")
		return i, nil
	}

	commentBody := gjson.Get(input, os.Getenv("COMMENT_BODY_FIELD")).Str
	i.Comment = fmt.Sprintf("%v %v", commentAuthor, commentBody)

	// transform status
	switch i.Status {
	case "Open":
		i.Status = "2"
	case "Investigating", "Identified", "Monitoring", "Escalated":
		i.Status = "22"
	case "Resolved", "Closed":
		i.Status = "6"
	default:
		return nil, fmt.Errorf("invalid ticket status %v", i.Status)
	}

	// transform priority
	switch i.Priority {
	case "P1 - Production system down":
		i.Priority = "1"
	case "P2 - Production system impaired":
		i.Priority = "2"
	case "P3 - Non production system impaired":
		i.Priority = "3"
	case "P4 - General request":
		i.Priority = "4"
	default:
		fmt.Printf("ignoring blank or unexpected priority: %v", i.Priority)
		return nil, nil
	}

	fmt.Printf("parsed incident: %+v\n", i)

	return i, nil
}

// Handle sends an incoming request to parser and processor, and returns a http response
func Handle(request *events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	inc, err := parseIncident(request.Body)
	if err != nil {
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusBadRequest,
			Body:       err.Error(),
		}, err
	}

	// assign identifier according to the api endpoint used
	// this is to differentiate between tickets initially created in JSD vs SNOW
	switch request.Resource {
	case "/v2/reverse":
		if inc.IntID != "" {
			inc.Identifier = inc.IntID
		}
	case "/v2/out":
		if inc.ExtID != "" {
			inc.Identifier = inc.ExtID
		}
	default:
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusBadRequest,
			Body:       err.Error(),
		}, err
	}

	err = process(inc)
	if err != nil {
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusInternalServerError,
			Body:       err.Error(),
		}, err
	}

	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
	}, nil
}
