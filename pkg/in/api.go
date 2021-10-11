package in

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/tidwall/gjson"
)

// Incident is a type of ticket
type Incident struct {
	Comment        string `json:"comment,omitempty"`
	CommentID      string `json:"comment_sysid,omitempty"`
	Identifier     string `json:"id,omitempty"`
	IntComment     string `json:"internal_comment,omitempty"`
	IntCommentID   string `json:"internal_comment_sysid,omitempty"`
	Description    string `json:"description,omitempty"`
	ExtID          string `json:"external_identifier,omitempty"`
	IntID          string `json:"internal_identifier,omitempty"`
	Priority       string `json:"priority,omitempty"`
	Reporter       string `json:"reporter_name,omitempty"`
	Resolution     string `json:"resolution,omitempty"`
	ResolutionCode string `json:"resolution_code,omitempty"`
	Service        string `json:"business_service,omitempty"`
	Status         string `json:"status,omitempty"`
	Summary        string `json:"summary,omitempty"`
}

// newIncident initialises an Incident
func newIncident() *Incident {
	return &Incident{}
}

// checkVars checks incoming payload has the required field values
func checkIncidentVars(input string) error {

	vars := []string{
		"DESCRIPTION_FIELD",
		"INTID_FIELD",
		"PRIORITY_FIELD",
		"REPORTER_FIELD",
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

	i := newIncident()

	i.ExtID = gjson.Get(input, os.Getenv("EXTID_FIELD")).Str

	// check for required values in new tickets only
	if i.ExtID == "" {
		err := checkIncidentVars(input)
		if err != nil {
			return nil, err
		}
	}

	i.Description = gjson.Get(input, os.Getenv("DESCRIPTION_FIELD")).Str
	i.Comment = gjson.Get(input, os.Getenv("COMMENT_FIELD")).Str
	i.CommentID = gjson.Get(input, os.Getenv("COMMENT_ID_FIELD")).Str
	i.IntComment = gjson.Get(input, os.Getenv("INTERNAL_COMMENT_FIELD")).Str
	i.IntCommentID = gjson.Get(input, os.Getenv("INTERNAL_COMMENT_ID_FIELD")).Str
	i.IntID = gjson.Get(input, os.Getenv("INTID_FIELD")).Str
	i.Priority = gjson.Get(input, os.Getenv("PRIORITY_FIELD")).Str
	i.Reporter = gjson.Get(input, os.Getenv("REPORTER_FIELD")).Str
	i.Resolution = gjson.Get(input, os.Getenv("RESOLUTION_FIELD")).Str
	i.Service = gjson.Get(input, os.Getenv("SERVICE_FIELD")).Str
	i.Status = gjson.Get(input, os.Getenv("STATUS_FIELD")).Str
	i.Summary = gjson.Get(input, os.Getenv("SUMMARY_FIELD")).Str

	// treat both type of comment as customer visible comments on JSD
	// initialise comment id if nil as it's being used as sort key
	switch {
	case i.IntCommentID == "" && i.CommentID == "":
		i.CommentID = "0"
	case i.IntCommentID != "" && i.CommentID == "":
		i.CommentID = i.IntCommentID
		i.Comment = i.IntComment
		i.IntComment = ""
	case i.IntCommentID == "" && i.CommentID != "":
		break
	case i.IntCommentID != "" && i.CommentID != "":
		break
	}

	// assign to an organisation in JSD
	switch i.Service {
	case "Cyclamen IT Platform Local":
		i.Service = "9"
	case "I-LEAP":
		i.Service = "58"
	case "Semaphore":
		i.Service = "45"
	default:
		i.Service = "65"
	}

	fmt.Printf("parsed incident: %v, status: %v, comment id: %v\n", i.IntID, i.Status, i.CommentID)

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
	case "/v2/add":
		inc.Identifier = inc.ExtID
	case "/v2/in":
		inc.Identifier = inc.IntID
	default:
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusBadRequest,
			Body:       err.Error(),
		}, err
	}

	res, err := process(inc)
	if err != nil {
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusInternalServerError,
			Body:       err.Error(),
		}, err
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
		}, err
	}

	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Body:       string(bmsg),
	}, nil
}
