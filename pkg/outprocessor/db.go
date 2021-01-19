package outprocessor

import (
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
)

func (d *Dynamo) checkPartial(p Payload) (bool, string, error) {

	partial := &dynamodb.QueryInput{
		TableName:              aws.String(os.Getenv("TABLE_NAME")),
		KeyConditionExpression: aws.String("external_identifier = :eid"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":eid": {
				S: aws.String(p.ExtID),
			},
		},
	}

	// look for just external_id match
	resp, err := d.DynamoDB.Query(partial)
	if err != nil {
		return false, "", fmt.Errorf("could not get item: %v", err)
	}

	if int(*resp.Count) != 0 {
		var pld Payload
		err = dynamodbattribute.UnmarshalMap(resp.Items[0], &pld)
		if err != nil {
			return false, "", fmt.Errorf("could not unmarshal item: %v", err)
		}
		if pld.IntID != "" {
			return true, pld.IntID, nil
		}
		return false, "", fmt.Errorf("partial entry has no internal identifier")
	}
	return false, "", nil
}

func (d *Dynamo) checkExact(p Payload) (bool, string, error) {

	exact := &dynamodb.GetItemInput{
		TableName: aws.String(os.Getenv("TABLE_NAME")),
		Key: map[string]*dynamodb.AttributeValue{
			"external_identifier": {
				S: aws.String(p.ExtID),
			},
			"comments": {
				S: aws.String(p.Comment),
			},
		},
	}

	// look for external_id and comment match
	resp, err := d.DynamoDB.GetItem(exact)
	if err != nil {
		return false, "", fmt.Errorf("could not get item: %v", err)
	}

	if resp.Item != nil {
		var pld Payload
		err = dynamodbattribute.UnmarshalMap(resp.Item, &pld)
		if err != nil {
			return false, "", fmt.Errorf("could not unmarshal item: %v", err)
		}
		if pld.IntID != "" {
			return true, pld.IntID, nil
		}
		return false, "", fmt.Errorf("exact entry has no internal identifier")
	}
	return false, "", nil
}

func (d *Dynamo) writeItem(p Payload) error {

	item, err := dynamodbattribute.MarshalMap(p)
	if err != nil {
		return fmt.Errorf("could not marshal db record: %s", err)
	}

	input := &dynamodb.PutItemInput{
		Item:      item,
		TableName: aws.String(os.Getenv("TABLE_NAME")),
	}

	_, err = d.DynamoDB.PutItem(input)
	if err != nil {
		return fmt.Errorf("could not put to db: %v", err)
	}

	fmt.Printf("new item added with external identifier: %v", p.ExtID)
	return nil
}
