package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/mitchellh/go-homedir"

	"github.com/briandowns/spinner"
	gphotos "github.com/gphotosuploader/google-photos-api-client-go/v3"
	"github.com/gphotosuploader/google-photos-api-client-go/v3/media_items"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

const (
	manifestPath = "./manifest.json"
	tokenPath = "./token.json"
	credsPath = "~/Downloads/client_secret_20836135302-f2fj886fcj8ggfr8bjf52l4jfuknokg1.apps.googleusercontent.com.json"
)

func main() {
	if len(os.Args) > 1 && (os.Args[1] == "-version" || os.Args[1] == "version" || os.Args[1] == "-v") {
		buildInfo := GetBuildInfo()
		fmt.Println(buildInfo)
		os.Exit(0)
	}

	ctx := context.Background()
	credsPath, err := homedir.Expand(credsPath)
	if err != nil {
		log.Fatalf("Error expanding path: %v", err)
	}

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

	token, err := tokenFromFile(tokenPath)
	if err != nil {
		token, err = getTokenFromWeb(config)
		if err != nil {
			fmt.Printf("Unable to get token: %v", err)
			return
		}
		if err := saveToken(tokenPath, token); err != nil {
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
	startTime := time.Now()
	s.Suffix = " fetching list of Google photos..."

	var allMediaItems []media_items.MediaItem
	updateSpinner := func(s *spinner.Spinner) {
		elapsed := time.Since(startTime)
		s.Suffix = fmt.Sprintf(" %s fetching list of Google photos... (Items: %s)", formatDuration(elapsed), humanize.Comma(int64(len(allMediaItems))))
	}

	s.PreUpdate = updateSpinner
	s.Start()

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
	totalDuration := time.Since(startTime)
	fmt.Printf("Fetched %s media items in %s\n", humanize.Comma(int64(len(allMediaItems))), formatDuration(totalDuration))

	s.Suffix = " Writing manifest..."
	s.Start()

	file, err := os.Create(manifestPath)
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
	fmt.Printf("%s created successfully\n", manifestPath)
}

func formatDuration(d time.Duration) string {
	d = d.Round(time.Second)
	h := d / time.Hour
	d -= h * time.Hour
	m := d / time.Minute
	d -= m * time.Minute
	s := d / time.Second

	if h > 0 {
		return fmt.Sprintf("%dh%dm%ds", h, m, s)
	}
	if m > 0 {
		return fmt.Sprintf("%dm%ds", m, s)
	}
	return fmt.Sprintf("%ds", s)
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

	encoder := json.NewEncoder(f)
	encoder.SetIndent("", "  ")
	return encoder.Encode(token)
}
