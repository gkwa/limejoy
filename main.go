package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	gphotos "github.com/gphotosuploader/google-photos-api-client-go/v3"
	"github.com/gphotosuploader/google-photos-api-client-go/v3/media_items"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

func main() {
	if len(os.Args) > 1 && (os.Args[1] == "-version" || os.Args[1] == "version" || os.Args[1] == "-v") {
		buildInfo := GetBuildInfo()
		fmt.Println(buildInfo)
		os.Exit(0)
	}

	ctx := context.Background()

	credsPath := "/Users/mtm/Downloads/client_secret_20836135302-f2fj886fcj8ggfr8bjf52l4jfuknokg1.apps.googleusercontent.com.json"

	b, err := os.ReadFile(credsPath)
	if err != nil {
		fmt.Printf("error reading credentials.json: %v", err)
		return
	}

	config, err := google.ConfigFromJSON(b, gphotos.PhotoslibraryReadonlyScope)
	if err != nil {
		fmt.Printf("error creating config from JSON: %v", err)
		return
	}

	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Visit the URL for the auth dialog: %v\n", authURL)

	fmt.Fprint(os.Stdout, "scan\n")

	var authCode string
	if _, err := fmt.Scan(&authCode); err != nil {
		fmt.Printf("error reading authorization code: %v", err)
		return
	}

	token, err := config.Exchange(ctx, authCode)
	if err != nil {
		fmt.Printf("error exchanging authorization code for token: %v", err)
		return
	}

	client := config.Client(ctx, token)
	gphotosClient, err := gphotos.NewClient(client)
	if err != nil {
		fmt.Printf("error creating gphotos client: %v", err)
		return
	}

	var allMediaItems []media_items.MediaItem
	var nextPageToken string

	for {
		options := &media_items.PaginatedListOptions{
			Limit:     100,
			PageToken: nextPageToken,
		}

		mediaItems, token, err := gphotosClient.MediaItems.PaginatedList(ctx, options)
		if err != nil {
			fmt.Printf("error listing media items: %v", err)
			return
		}

		allMediaItems = append(allMediaItems, mediaItems...)

		if token == "" {
			break
		}
		nextPageToken = token
	}

	file, err := os.Create("manifest.json")
	if err != nil {
		fmt.Printf("error creating manifest.json: %v", err)
		return
	}
	defer file.Close()

	enc := json.NewEncoder(file)
	enc.SetIndent("", "  ")
	if err := enc.Encode(allMediaItems); err != nil {
		fmt.Printf("error encoding media items to JSON: %v", err)
		return
	}

	fmt.Println("Manifest created successfully")
}
