package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/briandowns/spinner"
	gphotos "github.com/gphotosuploader/google-photos-api-client-go/v3"
	"github.com/gphotosuploader/google-photos-api-client-go/v3/media_items"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

func generateMediaItemsManifest(ctx context.Context, gphotosClient *gphotos.Client) ([]media_items.MediaItem, error) {
	prettyPrinter := message.NewPrinter(language.English)
	s := spinner.New(spinner.CharSets[9], 100*time.Millisecond)
	startTime := time.Now()
	s.Suffix = " fetching list of Google photos..."

	var allMediaItems []media_items.MediaItem
	updateSpinner := func(s *spinner.Spinner) {
		elapsed := time.Since(startTime)
		t := prettyPrinter.Sprintf("%d", len(allMediaItems))
		s.Suffix = fmt.Sprintf(" %s fetching list of Google photos... (Items: %s)", formatDuration(elapsed), t)
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
			return nil, fmt.Errorf("error listing media items: %v", err)
		}
		allMediaItems = append(allMediaItems, mediaItems...)
		if token == "" {
			break
		}
		nextPageToken = token
	}

	s.Stop()
	totalDuration := time.Since(startTime)
	t := prettyPrinter.Sprintf("%d", len(allMediaItems))
	fmt.Printf("Fetched %s media items in %s\n", t, formatDuration(totalDuration))

	return allMediaItems, nil
}

func writeManifest(allMediaItems []media_items.MediaItem) error {
	s := spinner.New(spinner.CharSets[9], 100*time.Millisecond)
	s.Suffix = " Writing manifest..."
	s.Start()

	file, err := os.Create(manifestPath)
	if err != nil {
		s.Stop()
		return fmt.Errorf("error creating manifest.json: %v", err)
	}
	defer file.Close()

	enc := json.NewEncoder(file)
	enc.SetIndent("", "  ")
	if err := enc.Encode(allMediaItems); err != nil {
		s.Stop()
		return fmt.Errorf("error encoding media items to JSON: %v", err)
	}

	s.Stop()
	return nil
}
