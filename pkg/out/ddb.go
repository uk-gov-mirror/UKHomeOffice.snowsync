package out

import (
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
)

func (d *Dynamo) checkPartial(inc *Incident) (bool, string, error) {

	fmt.Printf("\nlooking up an existing record with id: %v\n", inc.Identifier)

	partial := &dynamodb.QueryInput{
		TableName:              aws.String(os.Getenv("TABLE_NAME")),
		KeyConditionExpression: aws.String("id = :id"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":id": {
				S: aws.String(inc.Identifier),
			},
		},
	}

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
		if pld.IntID != "" {
			return true, pld.IntID, nil
		}
		return false, "", fmt.Errorf("partial entry has no internal identifier")
	}
	fmt.Println("no partial match found")
	return false, "", nil
}

func (d *Dynamo) checkExact(inc *Incident) (bool, error) {

	fmt.Printf("\nlooking up an existing record with id: %v and comment id: %v\n", inc.Identifier, inc.CommentID)

	exact := &dynamodb.GetItemInput{
		TableName: aws.String(os.Getenv("TABLE_NAME")),
		Key: map[string]*dynamodb.AttributeValue{
			"id": {
				S: aws.String(inc.Identifier),
			},
			"comment_sysid": {
				S: aws.String(inc.CommentID),
			},
		},
	}

	resp, err := d.DynamoDB.GetItem(exact)
	if err != nil {
		return false, fmt.Errorf("could not get item: %v", err)
	}

	if resp.Item != nil {
		var pld Incident
		err = dynamodbattribute.UnmarshalMap(resp.Item, &pld)
		if err != nil {
			return false, fmt.Errorf("could not unmarshal item: %v", err)
		}
		if pld.IntID != "" {
			return true, nil
		}
		return false, fmt.Errorf("exact entry has no internal identifier")
	}
	fmt.Println("no exact match found")
	return false, nil
}

func (d *Dynamo) writeItem(inc *Incident) error {

	item, err := dynamodbattribute.MarshalMap(inc)
	if err != nil {
		return fmt.Errorf("could not marshal db record: %s", err)
	}

	input := &dynamodb.PutItemInput{
		TableName: aws.String(os.Getenv("TABLE_NAME")),
		Item:      item,
	}

	_, err = d.DynamoDB.PutItem(input)
	if err != nil {
		return err
	}

	fmt.Printf("\nnew item added with identifier: %v\n", inc.Identifier)
	return nil
}
