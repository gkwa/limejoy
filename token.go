package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"golang.org/x/oauth2"
)

func getToken(config *oauth2.Config) (*oauth2.Token, error) {
	token, err := tokenFromFile(tokenPath)
	if err != nil {
		token, err = getTokenFromWeb(config)
		if err != nil {
			return nil, fmt.Errorf("unable to get token: %v", err)
		}
		if err := saveToken(tokenPath, token); err != nil {
			return nil, fmt.Errorf("unable to save token: %v", err)
		}
	}
	return token, nil
}

func getTokenFromWeb(config *oauth2.Config) (*oauth2.Token, error) {
	authCode := ""
	codeChan := make(chan string)

	path, err := getPath(callbackURL)
	if err != nil {
		return nil, fmt.Errorf("error getting path: %v", err)
	}

	http.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		codeChan <- code
		fmt.Fprintf(w, "Authorization successful! You can close this window now.")
	})

	go func() {
		port, err := getPort(callbackURL)
		if err != nil {
			fmt.Printf("Error getting port: %v\n", err)
			return
		}

		err = http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
		if err != nil {
			fmt.Printf("Failed to start server: %v\n", err)
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
