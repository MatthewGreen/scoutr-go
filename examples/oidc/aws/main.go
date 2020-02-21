package main

import (
	"flag"
	"net/http"
	"os/user"
	"path/filepath"

	"github.com/MichaelPalmer1/simple-api-go/config"
	"github.com/MichaelPalmer1/simple-api-go/helpers"
	dynamo "github.com/MichaelPalmer1/simple-api-go/providers/aws"
	"github.com/MichaelPalmer1/simple-api-go/providers/base"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	log "github.com/sirupsen/logrus"
)

func init() {
	api = dynamo.DynamoAPI{
		SimpleAPI: &base.SimpleAPI{
			Config: config.Config{},
		},
	}
}

func main() {
	// Command line arguments
	flag.StringVar(&api.Config.DataTable, "data-table", "", "Data table")
	flag.StringVar(&api.Config.AuthTable, "auth-table", "", "Auth table")
	flag.StringVar(&api.Config.GroupTable, "group-table", "", "Group table")
	flag.StringVar(&api.Config.AuditTable, "audit-table", "", "Audit table")
	flag.IntVar(&api.Config.LogRetentionDays, "log-retention-days", 30, "Days to retain read logs")
	flag.StringVar(&api.Config.OIDCUsernameClaim, "oidc-username-claim", "Sub", "Username claim from OIDC")
	flag.StringVar(&api.Config.OIDCNameClaim, "oidc-name-claim", "Name", "Name claim from OIDC")
	flag.StringVar(&api.Config.OIDCEmailClaim, "oidc-email-claim", "Mail", "Email claim from OIDC")
	flag.StringVar(&api.Config.OIDCGroupClaim, "oidc-group-claim", "", "Group claim from OIDC")
	flag.Parse()

	// Make sure required fields are provided
	if api.Config.DataTable == "" {
		log.Fatalln("data-table argument is required")
	}
	if api.Config.AuthTable == "" {
		log.Fatalln("auth-table argument is required")
	}
	if api.Config.GroupTable == "" {
		log.Fatalln("group-table argument is required")
	}

	usr, _ := user.Current()
	creds := credentials.NewSharedCredentials(filepath.Join(usr.HomeDir, ".aws/credentials"), "default")
	api.Init(&aws.Config{
		Region:      aws.String("us-east-1"),
		Credentials: creds,
	})

	// Initialize http server
	router, err := helpers.InitHTTPServer(api, "id", "/items/", []string{"CREATE", "UPDATE"})
	if err != nil {
		panic(err)
	}

	// Add get/create/update endpoints
	router.POST("/item/", create)
	router.GET("/item/:id", get)
	router.PUT("/item/:id", update)
	router.DELETE("/item/:id", delete)
	router.GET("/types/", listTypes)

	// Start the server
	log.Fatal(http.ListenAndServe(":8000", router))
}
