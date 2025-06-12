package in

import (
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

func (d *Dynamo) checkPartial(inc *Incident) (bool, string, error) {

	fmt.Printf("\nlooking up an existing record with id: %v\n", inc.Identifier)

	partial := &dynamodb.QueryInput{
		TableName:              aws.String(os.Getenv("TABLE_NAME")),
		KeyConditionExpression: aws.String("id = :id"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":id": &types.AttributeValueMemberS{
				Value: inc.Identifier,
			},
		},
	}

	resp, err := d.DynamoDB.Query(context.Background(), partial)
	if err != nil {
		return false, "", fmt.Errorf("could not get item: %v", err)
	}

	if int(resp.Count) != 0 {
		var pld Incident
		err = attributevalue.UnmarshalMap(resp.Items[0], &pld)
		if err != nil {
			return false, "", fmt.Errorf("could not unmarshal item: %v", err)
		}

		if pld.ExtID != "" {
			fmt.Printf("\npartial match found for %v\n", pld.ExtID)
			return true, pld.ExtID, nil
		}
		return false, "", fmt.Errorf("partial entry has no external identifier")
	}
	fmt.Println("no partial match found")
	return false, "", nil
}

func (d *Dynamo) checkExact(inc *Incident) (bool, error) {

	fmt.Printf("\nlooking up an existing record with id: %v and comment id: %v\n", inc.Identifier, inc.CommentID)

	// look for internal_id and comment match
	exact := &dynamodb.GetItemInput{
		TableName: aws.String(os.Getenv("TABLE_NAME")),
		Key: map[string]types.AttributeValue{
			"id": &types.AttributeValueMemberS{
				Value: inc.Identifier,
			},
			"comment_sysid": &types.AttributeValueMemberS{
				Value: inc.CommentID,
			},
		},
	}

	resp, err := d.DynamoDB.GetItem(context.Background(), exact)
	if err != nil {
		return false, fmt.Errorf("could not get item: %v", err)
	}

	if resp.Item != nil {
		var pld Incident
		err = attributevalue.UnmarshalMap(resp.Item, &pld)
		if err != nil {
			return false, fmt.Errorf("could not unmarshal item: %v", err)
		}
		if pld.ExtID != "" {
			fmt.Printf("\nexact match found for %v with comment id %v\n", pld.ExtID, pld.CommentID)
			return true, nil
		}
		return false, fmt.Errorf("exact entry has no external identifier")
	}
	fmt.Println("no exact match found")
	return false, nil
}

func (d *Dynamo) writeItem(inc *Incident) error {

	item, err := attributevalue.MarshalMap(inc)
	if err != nil {
		return fmt.Errorf("could not marshal db record: %s", err)
	}

	input := &dynamodb.PutItemInput{
		Item:      item,
		TableName: aws.String(os.Getenv("TABLE_NAME")),
	}

	_, err = d.DynamoDB.PutItem(context.Background(), input)
	if err != nil {
		return fmt.Errorf("could not put to db: %v", err)
	}

	fmt.Printf("\nitem added to db with internal identifier: %v\n", inc.Identifier)
	return nil
}
