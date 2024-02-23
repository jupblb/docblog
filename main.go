package main

import (
	"context"
	"fmt"
	"os"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	"google.golang.org/api/tasks/v1"
)

func main() {
	ctx := context.Background()

	b, err := os.ReadFile("credentials.json")
	if err != nil {
		panic(err)
	}
	config, err := google.ConfigFromJSON(b, tasks.TasksScope)
	if err != nil {
		panic(err)
	}

	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Visit the URL for the auth dialog: %v\n", authURL)
	fmt.Println("Enter the authorization code:")
	var authCode string
	if _, err := fmt.Scan(&authCode); err != nil {
		panic(err)
	}
	token, err := config.Exchange(ctx, authCode)
	if err != nil {
		panic(err)
	}

	client := config.Client(ctx, token)

	tasksService, err := tasks.NewService(
		ctx,
		// gcloud auth application-default login --scopes="https://www.googleapis.com/auth/tasks"
		// option.WithCredentialsFile(".gcloud/application_default_credentials.json"),
		option.WithHTTPClient(client),
	)
	if err != nil {
		panic(err)
	}

	tasklists, err := tasksService.Tasklists.List().Do()
	if err != nil {
		panic(err)
	}

	for _, item := range tasklists.Items {
		fmt.Println(item.Title)
	}
}
