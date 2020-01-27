package main

import (
	"context"
	"fmt"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/option"
)

func main() {
	ctx := context.Background()
	client, err := firestore.NewClient(ctx, "simple-api-265401", option.WithCredentialsFile("../gcp.json"))
	if err != nil {
		panic(err)
	}

	data := client.Collection("data")
	docs, err := data.Documents(ctx).GetAll()
	if err != nil {
		panic(err)
	}
	for _, doc := range docs {
		fmt.Println(doc.Ref.ID)
		fmt.Println(doc.Data())
	}
}
