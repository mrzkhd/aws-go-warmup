package main

import (
	"reflect"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/aws"
	"encoding/json"
	"os"
	"fmt"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
)

const (
	TABLE_NAME        = "deviceTable"
	MSG_CODE          = "code"
	MSG_MESSAGE       = "message"
	MSG_MISSING_FIELD = "Missing required field: "

	RESPONSE_CREATED_CODE               = 201
	RESPONSE_BAD_REQUEST_CODE           = 400
	RESPONSE_INTERNAL_SERVER_ERROR_CODE = 500
	RESPONSE_OK_CODE                    = 200

	RESPONSE_INTERNAL_SERVER_ERROR_MSG  = "UNEXPECTED INTERNAL ERROR "
	RESPONSE_BAD_REQUEST_MSG            = "INVALID REQUEST DATA "
	RESPONSE_DDB_EXCEPTION_MSG          = "DDB INIT EXCEPTION"
	RESPONSE_DDB_SAVE_EXCEPTION_MSG     = "DDB SAVE EXCEPTION"
	RESPONSE_DDB_LOAD_EXCEPTION_MSG     = "DDB LOAD EXCEPTION"
	RESPONSE_DDB_DATA_ALREADY_EXIST_MSG = "DATA ALREADY EXIST EXCEPTION"
)

type internalError struct {
	code    int
	message string
}

type Device struct {
	Id          string `json:"id"`
	DeviceModel string `json:"device_model"`
	Name        string `json:"name"`
	Note        string `json:"note"`
	Serial      string `json:"serial"`
}

var svc *dynamodb.DynamoDB

func Handler(request events.APIGatewayProxyRequest) (res events.APIGatewayProxyResponse, err error) {

	defer handlerRecover(&res)

	dev := getDevice(request) // unmarshalling

	validationMsg, validationErr := validationRequest(dev) //check existence of all fields

	if validationErr {
		return getResponse(RESPONSE_BAD_REQUEST_CODE, validationMsg), nil
	}

	initDdb() //create service client of ddb

	if !alreadyExist(dev) {
		saveToDdb(dev)
	} else {
		panic(internalError{RESPONSE_BAD_REQUEST_CODE, RESPONSE_DDB_DATA_ALREADY_EXIST_MSG})
	}

	return getResponse(RESPONSE_CREATED_CODE, "Done."), nil

}

func alreadyExist(device Device) bool {
	dev := loadFromDdb(device.Id)
	if (dev == Device{} && reflect.DeepEqual(Device{}, dev)) {
		return false
	}
	return true
}

func saveToDdb(dev Device) {

	// Write to DynamoDB
	item, _ := dynamodbattribute.MarshalMap(dev)
	input := &dynamodb.PutItemInput{
		Item:      item,
		TableName: aws.String(TABLE_NAME),
	}

	if _, err := svc.PutItem(input); err != nil {
		panic(internalError{RESPONSE_INTERNAL_SERVER_ERROR_CODE, err.Error()})
	}
}

func loadFromDdb(key string) (Device) {

	result, err := svc.GetItem(&dynamodb.GetItemInput{
		TableName: aws.String(TABLE_NAME),
		Key: map[string]*dynamodb.AttributeValue{
			"id": {S: aws.String(key)},
		},
	})
	if err != nil {
		panic(internalError{RESPONSE_INTERNAL_SERVER_ERROR_CODE, err.Error()})
	}

	device := Device{}
	if err := dynamodbattribute.UnmarshalMap(result.Item, &device); err != nil {
		panic(internalError{RESPONSE_INTERNAL_SERVER_ERROR_CODE, RESPONSE_DDB_LOAD_EXCEPTION_MSG})
	}

	return device

}

func initDdb() {

	region := os.Getenv("AWS_REGION")
	if session, err := session.NewSession(&aws.Config{ // Use aws sdk to connect to dynamoDB
		Region: &region,
	}); err != nil {
		fmt.Println(fmt.Sprintf("Failed to connect to AWS: %s", err.Error()))
		panic(internalError{RESPONSE_INTERNAL_SERVER_ERROR_CODE, RESPONSE_DDB_EXCEPTION_MSG})
	} else {
		svc = dynamodb.New(session) // Create DynamoDB client
	}

}

func getDevice(request events.APIGatewayProxyRequest) (Device) {
	var device Device

	if (reflect.DeepEqual(request, events.APIGatewayProxyRequest{})) || (len(request.Body) < 1) {
		panic(internalError{RESPONSE_BAD_REQUEST_CODE, RESPONSE_BAD_REQUEST_MSG})
	}
	err := json.Unmarshal([]byte(request.Body), &device)

	if err != nil {
		panic(internalError{RESPONSE_BAD_REQUEST_CODE, RESPONSE_BAD_REQUEST_MSG})
	} else {
		return device
	}

}

func validationRequest(device Device) (msg string, error bool) {

	devVal := reflect.ValueOf(device)

	for i := 0; i < devVal.NumField(); i++ {
		if devVal.Field(i).Len() == 0 {
			return MSG_MISSING_FIELD + devVal.Type().Field(i).Name, true
		}
	}
	return "", false

}

func getResponse(code int, result string) events.APIGatewayProxyResponse {
	return events.APIGatewayProxyResponse{
		StatusCode: code,
		Body:       result,
	}
}

func handlerRecover(res *events.APIGatewayProxyResponse) {

	r := recover()

	if r != nil {

		errorType := reflect.TypeOf(r)

		if errorType == reflect.TypeOf(internalError{}) {
			codeStr := reflect.ValueOf(r).FieldByName(MSG_CODE).Int()
			messageStr := reflect.ValueOf(r).FieldByName(MSG_MESSAGE).String()
			res.Body = string(messageStr)
			res.StatusCode = int(codeStr)
		} else {
			res.Body = RESPONSE_INTERNAL_SERVER_ERROR_MSG
			res.StatusCode = RESPONSE_INTERNAL_SERVER_ERROR_CODE
		}
	} else {

	}
	//res.Headers = map[string]string{"Content-Type": string("application/json")}

}

func main() {
	lambda.Start(Handler)
}
