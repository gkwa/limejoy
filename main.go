package main

import (
	"context"
	"fmt"
	"os"

	gphotos "github.com/gphotosuploader/google-photos-api-client-go/v3"
	"github.com/mitchellh/go-homedir"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

const (
	callbackURL  = "http://localhost:8080/auth/google/callback"
	credsPath    = "~/Downloads/client_secret_20836135302-f2fj886fcj8ggfr8bjf52l4jfuknokg1.apps.googleusercontent.com.json"
	manifestPath = "./manifest.json"
	tokenPath    = "./token.json"
)

func main() {
	if checkVersion() {
		return
	}

	ctx := context.Background()
	config, err := getGoogleConfig()
	if err != nil {
		fmt.Printf("Error getting Google config: %v\n", err)
		return
	}

	token, err := getToken(config)
	if err != nil {
		fmt.Printf("Error getting token: %v\n", err)
		return
	}

	client := config.Client(ctx, token)
	gphotosClient, err := gphotos.NewClient(client)
	if err != nil {
		fmt.Printf("Error creating gphotos client: %v\n", err)
		return
	}

	allMediaItems, err := fetchMediaItems(ctx, gphotosClient)
	if err != nil {
		fmt.Printf("Error fetching media items: %v\n", err)
		return
	}

	err = writeManifest(allMediaItems)
	if err != nil {
		fmt.Printf("Error writing manifest: %v\n", err)
		return
	}

	fmt.Printf("%s created successfully\n", manifestPath)
}

func getGoogleConfig() (*oauth2.Config, error) {
	credsPath, err := homedir.Expand(credsPath)
	if err != nil {
		return nil, fmt.Errorf("error expanding path: %v", err)
	}

	b, err := os.ReadFile(credsPath)
	if err != nil {
		return nil, fmt.Errorf("error reading %s: %v", credsPath, err)
	}

	config, err := google.ConfigFromJSON(b, gphotos.PhotoslibraryReadonlyScope)
	if err != nil {
		return nil, fmt.Errorf("error creating config from JSON: %v", err)
	}

	config.RedirectURL = callbackURL
	return config, nil
}
