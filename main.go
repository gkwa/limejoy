package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/briandowns/spinner"
	gphotos "github.com/gphotosuploader/google-photos-api-client-go/v3"
	"github.com/gphotosuploader/google-photos-api-client-go/v3/media_items"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

const tokenFile = "token.json"

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

	config.RedirectURL = "http://localhost:8080/auth/google/callback"

	token, err := tokenFromFile(tokenFile)
	if err != nil {
		token, err = getTokenFromWeb(config)
		if err != nil {
			fmt.Printf("Unable to get token: %v", err)
			return
		}
		if err := saveToken(tokenFile, token); err != nil {
			fmt.Printf("Unable to save token: %v", err)
			return
		}
	}

	client := config.Client(ctx, token)
	gphotosClient, err := gphotos.NewClient(client)
	if err != nil {
		fmt.Printf("error creating gphotos client: %v", err)
		return
	}

	s := spinner.New(spinner.CharSets[9], 100*time.Millisecond)
	s.Suffix = " fetching list of Google photos..."
	s.Start()

	var allMediaItems []media_items.MediaItem
	var nextPageToken string
	for {
		options := &media_items.PaginatedListOptions{
			Limit:     100,
			PageToken: nextPageToken,
		}
		mediaItems, token, err := gphotosClient.MediaItems.PaginatedList(ctx, options)
		if err != nil {
			s.Stop()
			fmt.Printf("error listing media items: %v", err)
			return
		}
		allMediaItems = append(allMediaItems, mediaItems...)
		if token == "" {
			break
		}
		nextPageToken = token
	}

	s.Stop()
	fmt.Printf("Fetched %d media items\n", len(allMediaItems))

	s.Suffix = " Writing manifest..."
	s.Start()

	file, err := os.Create("manifest.json")
	if err != nil {
		s.Stop()
		fmt.Printf("error creating manifest.json: %v", err)
		return
	}
	defer file.Close()

	enc := json.NewEncoder(file)
	enc.SetIndent("", "  ")
	if err := enc.Encode(allMediaItems); err != nil {
		s.Stop()
		fmt.Printf("error encoding media items to JSON: %v", err)
		return
	}

	s.Stop()
	fmt.Println("Manifest created successfully")
}

func getTokenFromWeb(config *oauth2.Config) (*oauth2.Token, error) {
	authCode := ""
	codeChan := make(chan string)

	http.HandleFunc("/auth/google/callback", func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		codeChan <- code
		fmt.Fprintf(w, "Authorization successful! You can close this window now.")
	})

	go func() {
		err := http.ListenAndServe(":8080", nil)
		if err != nil {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Visit the URL for the auth dialog: %v\n", authURL)

	authCode = <-codeChan

	token, err := config.Exchange(context.Background(), authCode)
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve token from web: %v", err)
	}
	return token, nil
}

func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	tok := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(tok)
	return tok, err
}

func saveToken(path string, token *oauth2.Token) error {
	fmt.Printf("Saving credential file to: %s\n", path)
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("unable to cache oauth token: %v", err)
	}
	defer f.Close()
	return json.NewEncoder(f).Encode(token)
}
