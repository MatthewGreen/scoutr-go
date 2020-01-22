package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os/user"
	"path/filepath"

	"github.com/MichaelPalmer1/simple-api-go/config"
	"github.com/MichaelPalmer1/simple-api-go/endpoints"
	"github.com/MichaelPalmer1/simple-api-go/httpserver"
	"github.com/MichaelPalmer1/simple-api-go/models"
	"github.com/MichaelPalmer1/simple-api-go/utils"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/julienschmidt/httprouter"
)

// Record : Item in Dynamo
type Record map[string]interface{}

var api endpoints.SimpleAPI
var validation map[string]utils.FieldValidation

func init() {
	validation = map[string]utils.FieldValidation{
		"value": func(value string, item map[string]string, existingItem map[string]string) (bool, string, error) {
			if value != "hello" {
				return false, "Invalid value", nil
			}

			return true, "", nil
		},
	}
}

// Initialize - Creates connection to DynamoDB
func Initialize(config *config.Config) *dynamodb.DynamoDB {
	usr, _ := user.Current()

	creds := credentials.NewSharedCredentials(filepath.Join(usr.HomeDir, ".aws/credentials"), "default")
	sess := session.Must(session.NewSession(&aws.Config{
		Region:      aws.String("us-east-1"),
		Credentials: creds,
	}))

	svc := dynamodb.New(sess)

	return svc
}

func create(w http.ResponseWriter, req *http.Request, params httprouter.Params) {
	requestUser := models.RequestUser{
		ID: "michael",
	}

	// Parse the request body
	var body map[string]string
	err := json.NewDecoder(req.Body).Decode(&body)
	if err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Build the request model
	request := models.Request{
		User:   requestUser,
		Method: req.Method,
		Path:   req.URL.Path,
		Body:   body,
	}

	// Create the item
	data, err := api.Create(request, body, validation)

	// Check for errors in the response
	if httpserver.HTTPErrorHandler(err, w) {
		return
	}

	// Marshal the response and write it to output
	out, _ := json.Marshal(data)
	w.Header().Add("Content-Type", "application/json")
	w.Write(out)
}

func get(w http.ResponseWriter, req *http.Request, params httprouter.Params) {
	requestUser := models.RequestUser{
		ID: "michael",
	}

	// Build the request model
	request := models.Request{
		User:   requestUser,
		Method: req.Method,
		Path:   req.URL.Path,
	}

	// Fetch the item
	data, err := api.Get(request, params.ByName("id"))

	// Check for errors in the response
	if httpserver.HTTPErrorHandler(err, w) {
		return
	}

	// Marshal the response and write it to output
	out, _ := json.Marshal(data)
	w.Header().Add("Content-Type", "application/json")
	w.Write(out)
}

func update(w http.ResponseWriter, req *http.Request, params httprouter.Params) {
	requestUser := models.RequestUser{
		ID: "michael",
	}

	// Parse the request body
	var body map[string]string
	err := json.NewDecoder(req.Body).Decode(&body)
	if err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Build the request model
	request := models.Request{
		User:   requestUser,
		Method: req.Method,
		Path:   req.URL.Path,
		Body:   body,
	}

	// Get key schema
	tableInfo, err := api.Client.DescribeTable(&dynamodb.DescribeTableInput{
		TableName: aws.String(api.DataTable),
	})
	if err != nil {
		fmt.Println("Failed to describe table", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Build partition key
	partitionKey := make(map[string]string)
	for _, schema := range tableInfo.Table.KeySchema {
		if *schema.KeyType == "HASH" {
			partitionKey[*schema.AttributeName] = params.ByName("id")
			break
		}
	}

	// Update the item
	data, err := api.Update(request, partitionKey, body, validation)

	// Check for errors in the response
	if httpserver.HTTPErrorHandler(err, w) {
		return
	}

	// Marshal the response and write it to output
	out, _ := json.Marshal(data)
	w.Header().Add("Content-Type", "application/json")
	w.Write(out)
}

func main() {
	// Command line arguments
	var config config.Config
	flag.StringVar(&config.DataTable, "data-table", "", "Data table")
	flag.StringVar(&config.AuthTable, "auth-table", "", "Auth table")
	flag.StringVar(&config.GroupTable, "group-table", "", "Group table")
	flag.StringVar(&config.AuditTable, "audit-table", "", "Audit table")
	flag.IntVar(&config.LogRetentionDays, "log-retention-days", 30, "Days to retain read logs")
	flag.Parse()

	svc := Initialize(&config)
	api.DataTable = config.DataTable
	api.Client = svc

	// Initialize http server
	router, err := httpserver.InitHTTPServer(api, "id", "/items/", []string{"CREATE", "UPDATE"})
	if err != nil {
		panic(err)
	}

	// Add get/create/update endpoints
	router.POST("/item/", create)
	router.GET("/item/:id", get)
	router.PUT("/item/:id", update)

	// Start the server
	log.Fatal(http.ListenAndServe(":8000", router))
}
