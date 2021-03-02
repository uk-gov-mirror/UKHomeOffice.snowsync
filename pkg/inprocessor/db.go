package inprocessor

import (
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
)

func (d *Dynamo) checkPartial(p Incident) (bool, string, error) {

	partial := &dynamodb.QueryInput{
		TableName:              aws.String(os.Getenv("TABLE_NAME")),
		KeyConditionExpression: aws.String("internal_identifier = :iid"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":iid": {
				S: aws.String(p.IntID),
			},
		},
	}

	// look for just external_id match
	resp, err := d.DynamoDB.Query(partial)
	if err != nil {
		return false, "", fmt.Errorf("could not get item: %v", err)
	}

	if int(*resp.Count) != 0 {
		var pld Incident
		err = dynamodbattribute.UnmarshalMap(resp.Items[0], &pld)
		if err != nil {
			return false, "", fmt.Errorf("could not unmarshal item: %v", err)
		}
		if pld.ExtID != "" {
			fmt.Printf("partial match found for %v", pld.ExtID)
			return true, pld.ExtID, nil
		}
		return false, "", fmt.Errorf("partial entry has no external identifier")
	}
	return false, "", nil
}

func (d *Dynamo) checkExact(p Incident) (bool, string, error) {

	exact := &dynamodb.GetItemInput{
		TableName: aws.String(os.Getenv("TABLE_NAME")),
		Key: map[string]*dynamodb.AttributeValue{
			"internal_identifier": {
				S: aws.String(p.IntID),
			},
			"comment_sysid": {
				S: aws.String(p.CommentID),
			},
		},
	}

	// look for internal_id and comment match
	resp, err := d.DynamoDB.GetItem(exact)
	if err != nil {
		return false, "", fmt.Errorf("could not get item: %v", err)
	}

	if resp.Item != nil {
		var pld Incident
		err = dynamodbattribute.UnmarshalMap(resp.Item, &pld)
		if err != nil {
			return false, "", fmt.Errorf("could not unmarshal item: %v", err)
		}
		if pld.ExtID != "" {
			fmt.Printf("exact match found for %v with comment id %v", pld.ExtID, pld.CommentID)
			return true, pld.ExtID, nil
		}
		return false, "", fmt.Errorf("exact entry has no external identifier")
	}
	return false, "", nil
}

func (d *Dynamo) writeItem(p Incident) error {

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

	// add to other table as well
	input = &dynamodb.PutItemInput{
		Item:      item,
		TableName: aws.String(os.Getenv("OUT_TABLE_NAME")),
	}

	_, err = d.DynamoDB.PutItem(input)
	if err != nil {
		return fmt.Errorf("could not put to db: %v", err)
	}

	fmt.Printf("new item added with internal identifier: %v", p.IntID)
	return nil
}
