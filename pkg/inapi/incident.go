package inapi

import (
	"fmt"
	"os"

	"github.com/tidwall/gjson"
)

// Incident is a type of ticket
type Incident struct {
	Comment        string `json:"comment,omitempty"`
	CommentID      string `json:"comment_sysid,omitempty"`
	IntComment     string `json:"internal_comment,omitempty"`
	IntCommentID   string `json:"internal_comment_sysid,omitempty"`
	Description    string `json:"description,omitempty"`
	ExtID          string `json:"external_identifier,omitempty"`
	IntID          string `json:"internal_identifier,omitempty"`
	Priority       string `json:"priority,omitempty"`
	Reporter       string `json:"reporter_name,omitempty"`
	Resolution     string `json:"resolution,omitempty"`
	ResolutionCode string `json:"resolution_code,omitempty"`
	Status         string `json:"status,omitempty"`
	Summary        string `json:"summary,omitempty"`
}

// NewIncident initialises an Incident
func NewIncident() *Incident {
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

// parseIncident gets values from inbound incident
func parseIncident(input string) (*Incident, error) {

	i := NewIncident()

	i.ExtID = gjson.Get(input, os.Getenv("EXTID_FIELD")).Str
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
	i.Status = gjson.Get(input, os.Getenv("STATUS_FIELD")).Str
	i.Summary = gjson.Get(input, os.Getenv("SUMMARY_FIELD")).Str

	// initialise comment id if nil as it's being used as sort key
	// treat both type of comment as customer visible comments
	switch {
	case i.CommentID == "" && i.IntCommentID == "":
		i.CommentID = "0"
	case i.IntCommentID != "":
		i.CommentID = i.IntCommentID
		i.Comment = i.IntComment
	}

	fmt.Printf("parsed incident: %v, status: %v, comment id: %v\n", i.IntID, i.Status, i.CommentID)

	return i, nil
}
