// Package dynamobrain is a simple AWS DynamoDB implementation of the bot.SimpleBrain
// interface, which gives the robot a place to permanently store it's memories.
package dynamobrain

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/lnxjedi/gopherbot/robot"
)

var handler robot.Handler
var svc *dynamodb.DynamoDB

type brainConfig struct {
	TableName, Region, AccessKeyID, SecretAccessKey string
}

type dynaMemory struct {
	Memory  string
	Content []byte
}

var dynamocfg brainConfig

func (db *brainConfig) Store(k string, b *[]byte) error {
	input := &dynamodb.PutItemInput{
		Item: map[string]*dynamodb.AttributeValue{
			"Memory": {
				S: aws.String(k),
			},
			"Content": {
				B: *b,
			},
		},
		TableName: aws.String(dynamocfg.TableName),
	}

	_, err := svc.PutItem(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case dynamodb.ErrCodeConditionalCheckFailedException:
				handler.Log(robot.Error, "Error storing memory: %v, %v", dynamodb.ErrCodeConditionalCheckFailedException, aerr.Error())
			case dynamodb.ErrCodeProvisionedThroughputExceededException:
				handler.Log(robot.Error, "Error storing memory: %v, %v", dynamodb.ErrCodeProvisionedThroughputExceededException, aerr.Error())
			case dynamodb.ErrCodeResourceNotFoundException:
				handler.Log(robot.Error, "Error storing memory: %v, %v", dynamodb.ErrCodeResourceNotFoundException, aerr.Error())
			case dynamodb.ErrCodeItemCollectionSizeLimitExceededException:
				handler.Log(robot.Error, "Error storing memory: %v, %v", dynamodb.ErrCodeItemCollectionSizeLimitExceededException, aerr.Error())
			case dynamodb.ErrCodeInternalServerError:
				handler.Log(robot.Error, "Error storing memory: %v, %v", dynamodb.ErrCodeInternalServerError, aerr.Error())
			default:
				handler.Log(robot.Error, "Error storing memory: %v", aerr.Error())
			}
			return aerr
		}
		// Print the error, cast err to awserr.Error to get the Code and
		// Message from an error.
		handler.Log(robot.Error, "Error storing memory: %v", err.Error())
		return err
	}

	return nil
}

func (db *brainConfig) Retrieve(k string) (datum *[]byte, exists bool, err error) {
	consistent := true
	result, err := svc.GetItem(&dynamodb.GetItemInput{
		TableName: aws.String(dynamocfg.TableName),
		Key: map[string]*dynamodb.AttributeValue{
			"Memory": {
				S: aws.String(k),
			},
		},
		ConsistentRead: &consistent,
	})

	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case dynamodb.ErrCodeProvisionedThroughputExceededException:
				handler.Log(robot.Error, "Error retrieving memory: %v, %v", dynamodb.ErrCodeProvisionedThroughputExceededException, aerr.Error())
			case dynamodb.ErrCodeResourceNotFoundException:
				handler.Log(robot.Error, "Error retrieving memory: %v, %v", dynamodb.ErrCodeResourceNotFoundException, aerr.Error())
			case dynamodb.ErrCodeInternalServerError:
				handler.Log(robot.Error, "Error retrieving memory: %v, %v", dynamodb.ErrCodeInternalServerError, aerr.Error())
			default:
				handler.Log(robot.Error, "Error retrieving memory: %v", aerr.Error())
			}
			return nil, false, aerr
		}
		handler.Log(robot.Error, "Error retrieving memory: %v", err.Error())
		return nil, false, err
	}

	m := dynaMemory{}

	err = dynamodbattribute.UnmarshalMap(result.Item, &m)

	if err != nil {
		handler.Log(robot.Error, "Failed to unmarshal Record, %v", err)
		return nil, false, err
	}

	if m.Memory == "" {
		return nil, false, nil
	}

	return &m.Content, true, nil
}

func provider(r robot.Handler) robot.SimpleBrain {
	handler = r
	handler.GetBrainConfig(&dynamocfg)
	var sess *session.Session
	var err error
	AccessKeyID := dynamocfg.AccessKeyID
	SecretAccessKey := dynamocfg.SecretAccessKey
	// ec2 provided credentials
	if len(AccessKeyID) == 0 {
		sess, err = session.NewSession(&aws.Config{
			Region: aws.String(dynamocfg.Region),
		})
		if err != nil {
			handler.Log(robot.Fatal, "Unable to establish AWS session: %v", err)
		}
	} else {
		sess, err = session.NewSession(&aws.Config{
			Region:      aws.String(dynamocfg.Region),
			Credentials: credentials.NewStaticCredentials(AccessKeyID, SecretAccessKey, ""),
		})
		if err != nil {
			handler.Log(robot.Fatal, "Unable to establish AWS session: %v", err)
		}
	}
	// Create DynamoDB client
	svc = dynamodb.New(sess)
	input := &dynamodb.DescribeTableInput{
		TableName: aws.String(dynamocfg.TableName),
	}
	_, err = svc.DescribeTable(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case dynamodb.ErrCodeResourceNotFoundException:
				handler.Log(robot.Fatal, "Error describing table '%s': %v, %v", dynamocfg.TableName, dynamodb.ErrCodeResourceNotFoundException, aerr.Error())
			case dynamodb.ErrCodeInternalServerError:
				handler.Log(robot.Fatal, "Error describing table '%s': %v, %v", dynamocfg.TableName, dynamodb.ErrCodeInternalServerError, aerr.Error())
			default:
				handler.Log(robot.Fatal, "Error describing table '%s': %v", dynamocfg.TableName, aerr.Error())
			}
		} else {
			handler.Log(robot.Fatal, "Error describing table '%s': %v", dynamocfg.TableName, err.Error())
		}
	}

	return &dynamocfg
}
