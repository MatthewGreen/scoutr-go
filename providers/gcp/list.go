package gcp

import (
	"cloud.google.com/go/firestore"
	"github.com/MichaelPalmer1/scoutr-go/models"
	log "github.com/sirupsen/logrus"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// List : List all records
func (api FirestoreAPI) List(req models.Request) ([]models.Record, error) {
	// Get the user
	user, err := api.InitializeRequest(api, req)
	if err != nil {
		// Bad user - pass the error through
		return nil, err
	}

	// Build filters
	collection := api.Client.Collection(api.Config.DataTable)
	f := FirestoreFiltering{
		Query: collection.Query,
	}
	filters, _, err := api.Filter(&f, user, api.BuildParams(req))
	if err != nil {
		return nil, err
	}
	query := collection.Query
	if filters != nil {
		query = filters.(firestore.Query)
	}

	// Query the data
	iter := query.Documents(api.context)
	records := []models.Record{}

	// Iterate through the results
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		} else if err != nil {
			// Attempt to convert error to a status code
			code, ok := status.FromError(err)

			// Check if the status conversion was successful
			if ok {
				switch code.Code() {
				case codes.InvalidArgument:
					// Return bad request on invalid argument errors
					return nil, &models.BadRequest{
						Message: code.Message(),
					}
				}
			}

			// Fallback to just returning the raw error
			return nil, err
		}

		// Add item to records
		records = append(records, doc.Data())
	}

	// Filter the response
	api.PostProcess(records, user)

	// Create audit log
	if err := api.auditLog("LIST", req, *user, nil, nil); err != nil {
		log.Warnf("Failed to create audit log: %v", err)
	}

	return records, nil
}
