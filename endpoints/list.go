package endpoints

import (
	"fmt"

	"github.com/MichaelPalmer1/simple-api-go/filterbuilder"
	"github.com/MichaelPalmer1/simple-api-go/models"
	"github.com/MichaelPalmer1/simple-api-go/utils"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/expression"
)

// ListTable : Lists all items in a table
func (api *SimpleAPI) ListTable(req models.Request, uniqueKey string, pathParams map[string]string, queryParams map[string]string) ([]models.Record, error) {
	// Get the user
	user, err := utils.InitializeRequest(req, *api.Client)
	if err != nil {
		// Bad user - pass the error through
		return nil, err
	}

	input := dynamodb.ScanInput{
		TableName: aws.String(api.DataTable),
	}

	// Generate dynamic search
	searchKey, hasSearchKey := pathParams["search_key"]
	searchValue, hasSearchValue := pathParams["search_value"]
	if hasSearchKey && hasSearchValue {
		// Map the search key and value into path params
		pathParams[searchKey] = searchValue
		delete(pathParams, "search_key")
		delete(pathParams, "search_value")
	}

	// Merge pathParams into queryParams
	for key, value := range pathParams {
		queryParams[key] = value
	}

	// Build filters
	conditions, hasConditions := filterbuilder.Filter(user, queryParams)
	if hasConditions {
		expr, err := expression.NewBuilder().WithFilter(conditions).Build()
		if err != nil {
			return nil, err
		}

		// Update scan input
		input.FilterExpression = expr.Filter()
		input.ExpressionAttributeNames = expr.Names()
		input.ExpressionAttributeValues = expr.Values()
	}

	// Download the data
	data, err := scan(&input, api.Client)
	if err != nil {
		fmt.Println("Error while attempting to list records", err)
		return nil, nil
	}

	// Filter the response
	fmt.Println(user)
	fmt.Println(data)
	filteredData := utils.PostProcess(data, user)
	fmt.Println(filteredData)

	// Sort the response if unique key was specified

	// Create audit log
	utils.AuditLog()

	return data, nil
}