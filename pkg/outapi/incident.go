package outapi

import (
	"fmt"
	"os"

	"github.com/tidwall/gjson"
)

// Incident is a type of ticket
type Incident struct {
	Comment     string `json:"comments,omitempty"`
	CommentID   string `json:"comment_sysid,omitempty"`
	Description string `json:"description,omitempty"`
	Identifier  string `json:"external_identifier,omitempty"`
	Priority    string `json:"priority,omitempty"`
	Status      string `json:"state,omitempty"`
	Service     string `json:"business_service,omitempty"`
	Summary     string `json:"title,omitempty"`
}

// NewIncident initialises an Incident
func NewIncident() *Incident {
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

// parseIncident gets values from inbound incident
func parseIncident(input string) (*ticketUpdate, error) {

	err := checkIncidentVars(input)
	if err != nil {
		return nil, err
	}

	i := NewIncident()

	i.Comment = gjson.Get(input, os.Getenv("COMMENT_FIELD")).Str
	i.CommentID = gjson.Get(input, os.Getenv("COMMENT_ID_FIELD")).Str
	i.Description = gjson.Get(input, os.Getenv("DESCRIPTION_FIELD")).Str
	i.Identifier = gjson.Get(input, os.Getenv("ISSUE_ID_FIELD")).Str
	i.Priority = gjson.Get(input, os.Getenv("PRIORITY_FIELD")).Str
	i.Status = gjson.Get(input, os.Getenv("STATUS_FIELD")).Str
	i.Summary = gjson.Get(input, os.Getenv("SUMMARY_FIELD")).Str

	commentAuthor := gjson.Get(input, os.Getenv("COMMENT_AUTHOR_FIELD")).Str
	commentBody := gjson.Get(input, os.Getenv("COMMENT_BODY_FIELD")).Str
	i.Comment = fmt.Sprintf("%v %v %v", commentAuthor, i.CommentID, commentBody)

	// make SNOW required modifications
	i.Service = "AWS ACP"

	if commentAuthor == "ServiceNow" {
		fmt.Printf("ignoring comment left by ServiceNow")
		return nil, nil
	}

	// initialise comment id if nil as it's being used as sort key
	if i.CommentID == "" {
		i.CommentID = "0"
	}
	// transform status
	switch i.Status {
	case "Open":
		i.Status = "2"
	case "Investigating", "Identified", "Monitoring":
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
	}

	u := ticketUpdate{incident: i}
	fmt.Printf("parsed incident: %v, status: %v\n", i.Identifier, i.Status)

	return &u, nil
}
