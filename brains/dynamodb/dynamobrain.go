// Package dynamobrain is a simple AWS DynamoDB implementation of the bot.SimpleBrain
// interface, which gives the robot a place to permanently store it's memories.
package dynamobrain

import (
	"context"
	"errors"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/smithy-go"
	"github.com/lnxjedi/gopherbot/robot"
)

var handler robot.Handler
var svc *dynamodb.Client

type brainConfig struct {
	TableName, Region, AccessKeyID, SecretAccessKey string
}

type dynaMemory struct {
	Memory  string
	Content []byte
}

var dynamocfg brainConfig

func (db *brainConfig) Store(k string, b *[]byte) error {
	item, err := attributevalue.MarshalMap(dynaMemory{
		Memory:  k,
		Content: *b,
	})
	if err != nil {
		handler.Log(robot.Error, "Error storing memory: %v", err)
		return err
	}

	input := &dynamodb.PutItemInput{
		Item:      item,
		TableName: aws.String(dynamocfg.TableName),
	}

	_, err = svc.PutItem(context.Background(), input)
	if err != nil {
		logDynamoError("storing memory", err)
		return err
	}

	return nil
}

func (db *brainConfig) Retrieve(k string) (datum *[]byte, exists bool, err error) {
	consistent := true
	result, err := svc.GetItem(context.Background(), &dynamodb.GetItemInput{
		TableName:      aws.String(dynamocfg.TableName),
		Key:            map[string]types.AttributeValue{"Memory": &types.AttributeValueMemberS{Value: k}},
		ConsistentRead: &consistent,
	})

	if err != nil {
		logDynamoError("retrieving memory", err)
		return nil, false, err
	}

	m := dynaMemory{}

	err = attributevalue.UnmarshalMap(result.Item, &m)

	if err != nil {
		handler.Log(robot.Error, "Failed to unmarshal Record, %v", err)
		return nil, false, err
	}

	if m.Memory == "" {
		return nil, false, nil
	}

	return &m.Content, true, nil
}

func (db *brainConfig) Delete(key string) error {
	delete := &dynamodb.DeleteItemInput{
		Key:       map[string]types.AttributeValue{"Memory": &types.AttributeValueMemberS{Value: key}},
		TableName: aws.String(dynamocfg.TableName),
	}
	_, err := svc.DeleteItem(context.Background(), delete)
	return err
}

func (db *brainConfig) List() ([]string, error) {
	keys := make([]string, 0)
	keyName := "Memory"
	scan := &dynamodb.ScanInput{
		ProjectionExpression: &keyName,
		TableName:            aws.String(dynamocfg.TableName),
	}
	res, err := svc.Scan(context.Background(), scan)
	if err != nil {
		return keys, err
	}
	for _, av := range res.Items {
		for _, item := range av {
			var m string
			err := attributevalue.Unmarshal(item, &m)
			if err != nil {
				return keys, err
			}
			keys = append(keys, m)
		}
	}
	return keys, nil
}

func (db *brainConfig) Shutdown() {
	// nothing to do, everything is synchronous
}

func provider(r robot.Handler) robot.SimpleBrain {
	handler = r
	handler.GetBrainConfig(&dynamocfg)
	ctx := context.Background()
	accessKeyID := dynamocfg.AccessKeyID
	secretAccessKey := dynamocfg.SecretAccessKey
	var cfg aws.Config
	var err error
	if len(accessKeyID) == 0 {
		cfg, err = config.LoadDefaultConfig(ctx, config.WithRegion(dynamocfg.Region))
		if err != nil {
			handler.Log(robot.Fatal, "Unable to establish AWS session: %v", err)
		}
	} else {
		creds := credentials.NewStaticCredentialsProvider(accessKeyID, secretAccessKey, "")
		cfg, err = config.LoadDefaultConfig(ctx, config.WithRegion(dynamocfg.Region), config.WithCredentialsProvider(creds))
		if err != nil {
			handler.Log(robot.Fatal, "Unable to establish AWS session: %v", err)
		}
	}
	// Create DynamoDB client
	svc = dynamodb.NewFromConfig(cfg)
	input := &dynamodb.DescribeTableInput{
		TableName: aws.String(dynamocfg.TableName),
	}
	_, err = svc.DescribeTable(ctx, input)
	if err != nil {
		logDynamoError("describing table", err)
		handler.Log(robot.Fatal, "Error describing table '%s': %v", dynamocfg.TableName, err)
	}

	return &dynamocfg
}

func logDynamoError(action string, err error) {
	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		handler.Log(robot.Error, "Error %s: %s, %s", action, apiErr.ErrorCode(), apiErr.ErrorMessage())
		return
	}

	var resourceNotFound *types.ResourceNotFoundException
	if errors.As(err, &resourceNotFound) {
		handler.Log(robot.Error, "Error %s: %v", action, resourceNotFound)
		return
	}
	var throughput *types.ProvisionedThroughputExceededException
	if errors.As(err, &throughput) {
		handler.Log(robot.Error, "Error %s: %v", action, throughput)
		return
	}
	var internal *types.InternalServerError
	if errors.As(err, &internal) {
		handler.Log(robot.Error, "Error %s: %v", action, internal)
		return
	}
	var itemSize *types.ItemCollectionSizeLimitExceededException
	if errors.As(err, &itemSize) {
		handler.Log(robot.Error, "Error %s: %v", action, itemSize)
		return
	}
	var cond *types.ConditionalCheckFailedException
	if errors.As(err, &cond) {
		handler.Log(robot.Error, "Error %s: %v", action, cond)
		return
	}

	handler.Log(robot.Error, "Error %s: %v", action, err)
}
