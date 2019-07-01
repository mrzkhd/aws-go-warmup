package main

import (
	"reflect"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/aws"
	"os"
	"fmt"
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

var svc *dynamodb.DynamoDB

func Handler(request events.APIGatewayProxyRequest) (res events.APIGatewayProxyResponse, err error) {

	defer handlerRecover(&res)

	initDdb() //create service client of ddb

	deleteAll()

	return getResponse(RESPONSE_CREATED_CODE, "Done."), nil

}
func deleteAll() {

	input := &dynamodb.DeleteTableInput{
		TableName: aws.String(TABLE_NAME),
	}
	svc.DeleteTable(input)
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
