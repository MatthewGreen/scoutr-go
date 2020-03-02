package aws

import (
	"fmt"

	"github.com/MichaelPalmer1/simple-api-go/models"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	log "github.com/sirupsen/logrus"
)

// InitializeRequest : Given a request, get the corresponding user and perform
// user and request validation.
func (api DynamoAPI) InitializeRequest(req models.Request) (*models.User, error) {
	user, err := api.GetUser(req.User.ID, req.User.Data)
	if err != nil {
		return nil, err
	}

	if err := api.ValidateUser(user); err != nil {
		log.Warnf("[%s] Bad User - %s", api.UserIdentifier(user), err)
		return nil, err
	}

	if err := api.ValidateRequest(req, user); err != nil {
		log.Warnf("[%s] %s", api.UserIdentifier(user), err)
		return nil, err
	}

	return user, nil
}

// GetUser : Fetch the user from the backend
func (api DynamoAPI) GetUser(id string, userData *models.UserData) (*models.User, error) {
	isUser := true
	user := models.User{ID: id}

	// Try to find user in the auth table
	result, err := api.Client.GetItem(&dynamodb.GetItemInput{
		TableName: aws.String(api.Config.AuthTable),
		Key: map[string]*dynamodb.AttributeValue{
			"id": {S: aws.String(id)},
		},
	})
	if err != nil {
		log.Infof("Failed to get user: %v", err)
		return nil, err
	} else if result.Item == nil {
		// Failed to find user in the table
		isUser = false
	} else {
		// Found a user, unmarshal into user object
		err := dynamodbattribute.UnmarshalMap(result.Item, &user)
		if err != nil {
			return nil, err
		}
	}

	// Try to find groups in the auth table
	groupIDs := []string{}
	if userData != nil {
		for _, groupID := range userData.Groups {
			var group models.User
			result, err := api.Client.GetItem(&dynamodb.GetItemInput{
				TableName: aws.String(api.Config.AuthTable),
				Key: map[string]*dynamodb.AttributeValue{
					"id": {S: aws.String(groupID)},
				},
			})
			if err != nil {
				log.Errorln("Failed to get group", err)
				return nil, err
			} else if result.Item == nil {
				// Group is not in the table
				continue
			} else {
				// Found group, unmarshal into group object
				err := dynamodbattribute.UnmarshalMap(result.Item, &group)
				if err != nil {
					return nil, err
				}
			}

			// Store this as a real group
			groupIDs = append(groupIDs, groupID)

			// Add sub-groups
			user.Groups = append(user.Groups, group.Groups...)

			// Merge permitted endpoints
			user.PermittedEndpoints = append(user.PermittedEndpoints, group.PermittedEndpoints...)

			// Merge exclude fields
			user.ExcludeFields = append(user.ExcludeFields, group.ExcludeFields...)

			// Merge update fields restricted
			user.UpdateFieldsRestricted = append(user.UpdateFieldsRestricted, group.UpdateFieldsRestricted...)

			// Merge update fields permitted
			user.UpdateFieldsPermitted = append(user.UpdateFieldsPermitted, group.UpdateFieldsPermitted...)

			// Merge filter fields
			user.FilterFields = append(user.FilterFields, group.FilterFields...)
		}
	}

	// Check that a user was found
	if !isUser && len(groupIDs) == 0 {
		return nil, &models.Unauthorized{
			Message: fmt.Sprintf("Auth id '%s' is not authorized", id),
		}
	}

	// If the user is a member of a group, merge in the group's permissions
	for _, groupID := range user.Groups {
		var group models.Group
		result, err := api.Client.GetItem(&dynamodb.GetItemInput{
			TableName: aws.String(api.Config.GroupTable),
			Key: map[string]*dynamodb.AttributeValue{
				"group_id": {S: aws.String(groupID)},
			},
		})
		if err != nil {
			log.Errorln("Failed to get group", err)
			return nil, err
		} else if result.Item == nil {
			// Group is not in the table
			return nil, &models.Unauthorized{
				Message: fmt.Sprintf("Group '%s' does not exist", groupID),
			}
		} else {
			// Found group, unmarshal into group object
			err := dynamodbattribute.UnmarshalMap(result.Item, &group)
			if err != nil {
				return nil, err
			}
		}

		// Merge user and group permissions together
		api.MergePermissions(&user, &group)
	}

	// Save user groups before applying metadata
	userGroups := user.Groups

	// Update user object with metadata
	if userData != nil {
		if userData.Username != "" {
			user.Username = userData.Username
		}
		if userData.Name != "" {
			user.Name = userData.Name
		}
		if userData.Email != "" {
			user.Email = userData.Email
		}
		if len(userData.Groups) > 0 {
			user.Groups = userData.Groups
		}
	}

	// Update user object with all applied OIDC groups
	if len(groupIDs) > 0 {
		var groups []string
		groups = append(groups, userGroups...)
		groups = append(groups, groupIDs...)
		user.Groups = groups
	}

	return &user, nil
}
