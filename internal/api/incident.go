package api

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/tidwall/gjson"
)

// Incident is a type of ticket
type Incident struct {
	Cluster     string    `json:"cluster,omitempty"`
	Comments    []Comment `json:"comments,omitempty"`
	Component   string    `json:"business_service,omitempty"`
	Description string    `json:"description,omitempty"`
	Identifier  string    `json:"external_identifier,omitempty"`
	Priority    string    `json:"priority,omitempty"`
	Status      string    `json:"status,omitempty"`
	Summary     string    `json:"summary,omitempty"`
}

// Comment is an individual post
type Comment struct {
	ID     string `json:"id,omitempty"`
	Author string `json:"owner,omitempty"`
	Body   string `json:"body,omitempty"`
}

// incidentUpdate is an event
type incidentUpdate struct {
	incident *Incident
	sqs      Messenger
}

// NewIncident initialises an Incident
func NewIncident() *Incident {
	return &Incident{}
}

// checkVars checks incoming payload has necessary fields
func checkVars(input string) error {

	vars := []string{
		"COMPONENT_FIELD",
		"DESCRIPTION_FIELD",
		"ISSUE_ID_FIELD",
		"PRIORITY_FIELD",
		"STATUS_FIELD",
		"SUMMARY_FIELD",
	}

	for _, v := range vars {
		field, ok := os.LookupEnv(v)
		if !ok {
			return fmt.Errorf("missing environment variable")
		}
		value := gjson.Get(input, field)
		if !value.Exists() {
			return fmt.Errorf("missing value in payload")
		}
	}
	return nil
}

func parseComment(g gjson.Result) Comment {
	var c Comment
	c.ID = gjson.Get(g.String(), "id").String()
	c.Author = gjson.Get(g.String(), "author.name").String()
	c.Body = gjson.Get(g.String(), "body").String()
	return c
}

// parseIncident gets some values from inbound request
func parseIncident(input string) (*incidentUpdate, error) {

	err := checkVars(input)
	if err != nil {
		return nil, err
	}

	i := NewIncident()
	i.Cluster = gjson.Get(input, os.Getenv("CLUSTER_FIELD")).Str
	i.Component = gjson.Get(input, os.Getenv("COMPONENT_FIELD")).Str
	i.Description = gjson.Get(input, os.Getenv("DESCRIPTION_FIELD")).Str
	i.Identifier = gjson.Get(input, os.Getenv("ISSUE_ID_FIELD")).Str
	i.Priority = gjson.Get(input, os.Getenv("PRIORITY_FIELD")).Str
	i.Status = gjson.Get(input, os.Getenv("STATUS_FIELD")).Str
	i.Summary = gjson.Get(input, os.Getenv("SUMMARY_FIELD")).Str

	comments := gjson.Get(input, os.Getenv("COMMENT_FIELD"))
	comments.ForEach(func(key, value gjson.Result) bool {
		i.Comments = append(i.Comments, parseComment(value))
		return true
	})

	u := incidentUpdate{incident: i}
	fmt.Printf("parsed incident: %v, status: %v\n", i.Identifier, i.Status)
	return &u, nil
}

// publish writes an incident update to SQS
func (i *incidentUpdate) publish() error {

	sm, err := json.Marshal(i.incident)
	if err != nil {
		return fmt.Errorf("failed to marshal SQS payload: %s", err)
	}

	in := sqs.SendMessageInput{
		MessageBody: aws.String(string(sm)),
		QueueUrl:    aws.String(os.Getenv("QUEUE_URL")),
	}

	_, err = i.sqs.SendMessage(&in)
	if err != nil {
		return fmt.Errorf("failed to publish incident: %s", err)
	}

	return nil

}
