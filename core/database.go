package core

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
)

func NewKeyPair(region string) (*KeyPair, error) {
	Logger.Info("Initializing DynamoDB client", "region", region)

	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(region),
	})
	if err != nil {
		return nil, fmt.Errorf("error creating AWS session: %w", err)
	}

	svc := dynamodb.New(sess)

	return &KeyPair{
		db: svc,
	}, nil
}

func (k *KeyPair) Put(tableName string, key map[string]*dynamodb.AttributeValue, item interface{}) error {
	av, err := dynamodbattribute.MarshalMap(item)
	if err != nil {
		return fmt.Errorf("error marshalling item: %w", err)
	}

	updateExpression := "SET "
	expressionAttributeNames := make(map[string]*string)
	expressionAttributeValues := make(map[string]*dynamodb.AttributeValue)

	i := 0
	for k, v := range av {
		if _, exists := key[k]; !exists {
			if i > 0 {
				updateExpression += ", "
			}
			placeholder := fmt.Sprintf(":val%d", i)
			namePlaceholder := fmt.Sprintf("#attr%d", i)
			updateExpression += fmt.Sprintf("%s = %s", namePlaceholder, placeholder)
			expressionAttributeNames[namePlaceholder] = aws.String(k)
			expressionAttributeValues[placeholder] = v
			i++
		}
	}

	input := &dynamodb.UpdateItemInput{
		Key:                       key,
		TableName:                 aws.String(tableName),
		UpdateExpression:          aws.String(updateExpression),
		ExpressionAttributeNames:  expressionAttributeNames,
		ExpressionAttributeValues: expressionAttributeValues,
		ReturnValues:              aws.String("UPDATED_NEW"),
	}

	_, err = k.db.UpdateItem(input)
	if err != nil {
		return fmt.Errorf("error updating item in table %s: %w", tableName, err)
	}

	Logger.Info("Successfully updated item in table", "tableName", tableName)
	return nil
}

func (k *KeyPair) Get(tableName string, key map[string]*dynamodb.AttributeValue, item interface{}) error {
	input := &dynamodb.GetItemInput{
		Key:       key,
		TableName: aws.String(tableName),
	}

	result, err := k.db.GetItem(input)
	if err != nil {
		return fmt.Errorf("error getting item from table %s: %w", tableName, err)
	}

	if result.Item == nil {
		return fmt.Errorf("item not found in table %s", tableName)
	}

	err = dynamodbattribute.UnmarshalMap(result.Item, item)
	if err != nil {
		return fmt.Errorf("error unmarshalling item: %w", err)
	}

	return nil
}

func (k *KeyPair) Delete(tableName string, key map[string]*dynamodb.AttributeValue) error {
	input := &dynamodb.DeleteItemInput{
		Key:       key,
		TableName: aws.String(tableName),
	}

	_, err := k.db.DeleteItem(input)
	if err != nil {
		return fmt.Errorf("error deleting item from table %s: %w", tableName, err)
	}

	return nil
}

func (k *KeyPair) Query(tableName string, keyConditionExpression string, expressionAttributeValues map[string]*dynamodb.AttributeValue, items interface{}) error {
	input := &dynamodb.QueryInput{
		TableName:                 aws.String(tableName),
		KeyConditionExpression:    aws.String(keyConditionExpression),
		ExpressionAttributeValues: expressionAttributeValues,
	}

	result, err := k.db.Query(input)
	if err != nil {
		return fmt.Errorf("error querying table %s: %w", tableName, err)
	}

	err = dynamodbattribute.UnmarshalListOfMaps(result.Items, items)
	if err != nil {
		return fmt.Errorf("error unmarshalling query results: %w", err)
	}

	return nil
}

func (k *KeyPair) Scan(tableName string, items interface{}) error {
	input := &dynamodb.ScanInput{
		TableName: aws.String(tableName),
	}

	result, err := k.db.Scan(input)
	if err != nil {
		return fmt.Errorf("error scanning table %s: %w", tableName, err)
	}

	err = dynamodbattribute.UnmarshalListOfMaps(result.Items, items)
	if err != nil {
		return fmt.Errorf("error unmarshalling scan results: %w", err)
	}

	return nil
}
