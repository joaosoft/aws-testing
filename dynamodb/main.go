package main

import (
	"fmt"

	"os"

	"encoding/json"
	"io/ioutil"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/dynamodb/expression"
	"github.com/aws/aws-sdk-go/service/dynamodbstreams"
)

type Details struct {
	Description string  `json:"description"`
	Price       float64 `json:"price"`
}

type Item struct {
	Name    string  `json:"name"`
	Age     int     `json:"age"`
	Details Details `json:"details"`
}

func main() {
	// Initialize a session in eu-central-1 that the SDK will use to load
	// credentials from the shared credentials file ~/.aws/credentials.
	session, err := session.NewSession(
		&aws.Config{
			Region:   aws.String("eu-central-1"),
			Endpoint: aws.String("http://localhost:8001"),
		},
	)

	if err != nil {
		fmt.Printf("error creating session: %s\n", err)
		return
	}

	table := "testing"

	fmt.Println(":: starting")

	createdTable := CreateTable(table, session)

	CreateTableRecord(table, session)

	ListTables(table, session)

	PutTableItems(table, session)

	ReadTableItem(table, session)

	ReadTableItems(table, session)

	DeleteTableItem(table, session)

	stream := DescribeStream(table, createdTable, session)

	ListStreams(table, session)

	shardIterator := GetStreamShardIterator(table, createdTable, stream, session)

	GetStreamRecords(table, session, shardIterator)

	DeleteTable(table, session)

	fmt.Println(":: done")
}

func CreateTable(table string, session *session.Session) *dynamodb.CreateTableOutput {
	fmt.Println("CREATE TABLE")

	// create DynamoDB client
	svc := dynamodb.New(session)

	// create table
	input := &dynamodb.CreateTableInput{
		AttributeDefinitions: []*dynamodb.AttributeDefinition{
			{
				AttributeName: aws.String("name"),
				AttributeType: aws.String("S"),
			},
			{
				AttributeName: aws.String("age"),
				AttributeType: aws.String("N"),
			},
		},
		KeySchema: []*dynamodb.KeySchemaElement{
			{
				AttributeName: aws.String("name"),
				KeyType:       aws.String("HASH"),
			},
			{
				AttributeName: aws.String("age"),
				KeyType:       aws.String("RANGE"),
			},
		},
		ProvisionedThroughput: &dynamodb.ProvisionedThroughput{
			ReadCapacityUnits:  aws.Int64(10),
			WriteCapacityUnits: aws.Int64(10),
		},
		StreamSpecification: &dynamodb.StreamSpecification{
			StreamEnabled:  aws.Bool(true),
			StreamViewType: aws.String("NEW_AND_OLD_IMAGES"),
		},
		TableName: aws.String(table),
	}

	result, err := svc.CreateTable(input)

	if err != nil {
		fmt.Printf("error calling CreateTable: %s\n", err)
		return nil
	}

	fmt.Println(result)

	return result
}

func CreateTableRecord(table string, session *session.Session) {
	fmt.Println("CREATE TABLE RECORD")

	// create DynamoDB client
	svc := dynamodb.New(session)

	// create table record
	details := Details{
		Description: "home rend",
		Price:       150.0,
	}

	item := Item{
		Name:    "joao ribeiro",
		Age:     30,
		Details: details,
	}

	av, err := dynamodbattribute.MarshalMap(item)

	if err != nil {
		fmt.Printf("error marshalling map: %s\n", err)
		return
	}

	// create item in table
	input := &dynamodb.PutItemInput{
		Item:      av,
		TableName: aws.String(table),
	}

	result, err := svc.PutItem(input)

	if err != nil {
		fmt.Printf("error calling CreateTableRecord: %s\n", err)
		return
	}

	fmt.Println(result)
}

func ListTables(table string, session *session.Session) {
	fmt.Println("LIST TABLES")

	// create DynamoDB client
	svc := dynamodb.New(session)

	// list tables
	result, err := svc.ListTables(&dynamodb.ListTablesInput{})

	if err != nil {
		fmt.Printf("error calling ListTables: %s\n", err)
		return
	}

	for _, n := range result.TableNames {
		fmt.Printf("table: %s\n", *n)
	}
}

func PutTableItems(table string, session *session.Session) {
	fmt.Println("PUT TABLE ITEM")
	// create DynamoDB client
	svc := dynamodb.New(session)

	// get table items from data.json
	raw, err := ioutil.ReadFile("./dynamodb/data.json")

	if err != nil {
		fmt.Printf("error calling PutTableItems: %s\n", err)
		return
	}

	var items []Item
	json.Unmarshal(raw, &items)

	// add each item to items table:
	for _, item := range items {
		av, err := dynamodbattribute.MarshalMap(item)

		if err != nil {
			fmt.Printf("error marshalling map: %s\n", err)
			return
		}

		// create item in table
		input := &dynamodb.PutItemInput{
			Item:      av,
			TableName: aws.String(table),
		}

		result, err := svc.PutItem(input)

		if err != nil {
			fmt.Printf("error calling PutTableItems: %s\n", err)
			return
		}

		fmt.Println(result)
	}
}

func ReadTableItem(table string, session *session.Session) {
	fmt.Println("READ TABLE ITEM")

	// create DynamoDB client
	svc := dynamodb.New(session)

	// read table item
	result, err := svc.GetItem(&dynamodb.GetItemInput{
		TableName: aws.String(table),
		Key: map[string]*dynamodb.AttributeValue{
			"name": {
				S: aws.String("joao ribeiro"),
			},
			"age": {
				N: aws.String("30"),
			},
		},
	})

	if err != nil {
		fmt.Printf("error calling ReadTableItem: %s\n", err)
		return
	}

	item := Item{}

	err = dynamodbattribute.UnmarshalMap(result.Item, &item)

	if err != nil {
		panic(fmt.Sprintf("failed to unmarshal Record, %v", err))
	}

	if item.Name == "" {
		fmt.Println("could not find anything")
		return
	}

	fmt.Printf("found item: %+v\n", item)
}

func ReadTableItems(table string, session *session.Session) {
	fmt.Println("READ TABLE ITEMS")

	min_price := -1.0
	age := 30

	// create DynamoDB client
	svc := dynamodb.New(session)

	// create the Expression to fill the input struct with.
	// get all items in that age; we'll pull out those with a higher price later
	filter := expression.Name("age").Equal(expression.Value(age))

	// or we could get by prices and pull out those with the right age later
	// filter := expression.Name("details.price").GreaterThan(expression.Value(min_price))

	// get back the name, age, and price
	proj := expression.NamesList(expression.Name("name"), expression.Name("age"), expression.Name("details.price"))

	expr, err := expression.NewBuilder().WithFilter(filter).WithProjection(proj).Build()

	if err != nil {
		fmt.Printf("error building expression: %s\n", err)
		return
	}

	// build the query input parameters
	params := &dynamodb.ScanInput{
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
		FilterExpression:          expr.Filter(),
		ProjectionExpression:      expr.Projection(),
		TableName:                 aws.String(table),
	}

	// make the DynamoDB query api call
	result, err := svc.Scan(params)

	if err != nil {
		fmt.Printf("query api call failed: %s\n", err)
		os.Exit(1)
	}

	count := 0
	for _, i := range result.Items {
		item := Item{}

		err = dynamodbattribute.UnmarshalMap(i, &item)

		if err != nil {
			fmt.Printf("error unmarshalling: %s\n", err)
			return
		}

		// which ones had a higher price?
		if item.Details.Price > min_price {
			// or it we had filtered by price previously:
			// if item.Age == age {
			count += 1

			fmt.Printf("found: %+v", item)
		}
	}

	fmt.Println("found:", count, "with a price above", min_price, "in", age)
}

func DeleteTableItem(table string, session *session.Session) {
	fmt.Println("DELETE TABLE ITEM")

	// create DynamoDB client
	svc := dynamodb.New(session)

	// delete table item
	input := &dynamodb.DeleteItemInput{
		Key: map[string]*dynamodb.AttributeValue{
			"age": {
				N: aws.String("30"),
			},
			"name": {
				S: aws.String("joao ribeiro"),
			},
		},
		TableName: aws.String(table),
	}

	result, err := svc.DeleteItem(input)

	if err != nil {
		fmt.Printf("error calling DescribeStream: %s\n", err)
		return
	}

	fmt.Println(result)
}

func DescribeStream(table string, createdTable *dynamodb.CreateTableOutput, session *session.Session) *dynamodbstreams.DescribeStreamOutput {
	fmt.Println("DESCRIBE STREAM")

	// create DynamoDB client
	svc := dynamodbstreams.New(session)

	// describe stream
	input := &dynamodbstreams.DescribeStreamInput{
		StreamArn: createdTable.TableDescription.LatestStreamArn,
	}

	result, err := svc.DescribeStream(input)
	if err != nil {
		fmt.Printf("error calling DescribeStream: %s\n", err)
		return nil
	}

	fmt.Println(result)

	return result
}

func ListStreams(table string, session *session.Session) {
	fmt.Println("LIST STREAMS")

	// create DynamoDB client
	svc := dynamodbstreams.New(session)

	// list streams
	input := &dynamodbstreams.ListStreamsInput{
		TableName: aws.String(table),
	}
	result, err := svc.ListStreams(input)

	if err != nil {
		fmt.Printf("error calling GetStreamShardIterator: %s\n", err)
		return
	}

	fmt.Println(result)
}

func GetStreamShardIterator(table string, createdTable *dynamodb.CreateTableOutput, stream *dynamodbstreams.DescribeStreamOutput, session *session.Session) *dynamodbstreams.GetShardIteratorOutput {
	fmt.Println("GETTING STREAM SHARD ITERATOR")

	// create DynamoDB client
	svc := dynamodbstreams.New(session)

	// get stream shard iterator
	input := &dynamodbstreams.GetShardIteratorInput{
		ShardId:           stream.StreamDescription.Shards[0].ShardId,
		ShardIteratorType: aws.String("TRIM_HORIZON"),
		StreamArn:         createdTable.TableDescription.LatestStreamArn,
	}

	result, err := svc.GetShardIterator(input)
	if err != nil {
		fmt.Printf("error calling GetStreamShardIterator: %s\n", err)
		return nil
	}

	fmt.Println(result)

	return result
}

func GetStreamRecords(table string, session *session.Session, iter *dynamodbstreams.GetShardIteratorOutput) {
	fmt.Println("GET STREAM RECORDS")

	// create DynamoDB client
	svc := dynamodbstreams.New(session)

	// get stream records
	input := &dynamodbstreams.GetRecordsInput{
		ShardIterator: iter.ShardIterator,
	}

	result, err := svc.GetRecords(input)
	if err != nil {
		fmt.Printf("error calling GetStreamRecords: %s\n", err)
		return
	}

	fmt.Println(result)
}

func DeleteTable(table string, session *session.Session) {
	fmt.Println("DELETE TABLE")

	// create DynamoDB client
	svc := dynamodb.New(session)

	// delete table
	input := &dynamodb.DeleteTableInput{
		TableName: aws.String(table),
	}

	result, err := svc.DeleteTable(input)

	if err != nil {
		fmt.Printf("error calling DeleteTable: %s\n", err)
		return
	}

	fmt.Println(result)
}
