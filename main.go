package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	admin "google.golang.org/api/admin/directory/v1"
	"google.golang.org/api/groupssettings/v1"
	"google.golang.org/api/option"
)

func main() {
	// set up flags
	customerID := flag.String("customer_id", "", "The customer ID to use.")
	keyPath := flag.String("key", "key.json", "Path to the client secret JSON file.")

	flag.Parse()

	if *customerID == "" {
		log.Fatal("customer_id must be provided")
	}

	b, err := os.ReadFile(*keyPath)
	if err != nil {
		log.Fatalf("Unable to read client secret file: %v", err)
	}

	ctx := context.Background()

	config, err := google.ConfigFromJSON(b, admin.AdminDirectoryGroupReadonlyScope, groupssettings.AppsGroupsSettingsScope)
	if err != nil {
		log.Fatalf("Unable to parse client secret file to config: %v", err)
	}

	// see if we have a valid local token
	token, err := getTokenFromFile("token.json")
	if err != nil || !token.Valid() {
		token = getTokenFromWeb(config)
		saveToken("token.json", token)
	} else {
		fmt.Println("Reusing existing token.")
	}

	client := config.Client(ctx, token)

	directoryService, err := admin.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		log.Fatalf("Unable to create Admin SDK Directory service: %v", err)
	}

	groupSettingsService, err := groupssettings.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		log.Fatalf("Unable to create Groups Settings service: %v", err)
	}

	var pageToken string
	for {
		req := directoryService.Groups.List().Customer(*customerID).MaxResults(100).PageToken(pageToken)
		resp, err := req.Do()
		if err != nil {
			log.Fatalf("Unable to retrieve groups: %v", err)
		}

		for _, group := range resp.Groups {
			groupSettings, err := groupSettingsService.Groups.Get(group.Email).Do()
			if err != nil {
				log.Printf("Unable to retrieve settings for group %s: %v", group.Email, err)
				continue
			}
			switch groupSettings.WhoCanJoin {
			case "ALL_IN_DOMAIN_CAN_JOIN":
				fmt.Printf("Group: %s | allows anyone in the domain to join\n", group.Email)
			case "ANYONE_CAN_JOIN":
				fmt.Printf("Group: %s | allows anyone anywhere to join\n", group.Email)
			}
		}

		if resp.NextPageToken == "" {
			break
		}
		pageToken = resp.NextPageToken
	}
}

// startWebServer starts a local web server and returns the authorization code via a channel
func startWebServer(codeCh chan string) *http.Server {
	srv := &http.Server{Addr: "localhost:8080"}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		fmt.Fprintf(w, "Authentication complete. You can close this window.")
		codeCh <- code
		go func() {
			time.Sleep(1 * time.Second)
			srv.Close()
		}()
	})

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Could not listen on %s: %v", srv.Addr, err)
		}
	}()

	return srv
}

// saveToken saves a token to a file path.
func saveToken(path string, token *oauth2.Token) {
	fmt.Printf("Saving credential file to: %s\n", path)
	f, err := os.Create(path)
	if err != nil {
		log.Fatalf("Unable to cache oauth token: %v", err)
	}
	defer f.Close()
	json.NewEncoder(f).Encode(token)
}

// getTokenFromFile retrieves a token from a local file.
func getTokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	tok := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(tok)
	return tok, err
}

// getTokenFromWeb triggers the OAuth flow to get a new token
func getTokenFromWeb(config *oauth2.Config) *oauth2.Token {
	codeCh := make(chan string)
	server := startWebServer(codeCh)
	defer server.Close()

	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Go to the following link in your browser and complete the authentication: \n%v\n", authURL)

	code := <-codeCh
	tok, err := config.Exchange(context.TODO(), code)
	if err != nil {
		log.Fatalf("Unable to retrieve token from web: %v", err)
	}
	return tok
}
