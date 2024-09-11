package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"

	"github.com/go-resty/resty/v2"
	"github.com/joho/godotenv"
)

// OAuthToken represents the response structure for the Twitch OAuth token
type OAuthToken struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}

// Game represents the structure of the IGDB game data
type Game struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	ReleaseDate int64  `json:"first_release_date"`
	Summary     string `json:"summary"`
}

// FetchOAuthToken retrieves an OAuth token from Twitch
func FetchOAuthToken() (string, error) {
	client := resty.New()

	// Get credentials from environment variables
	clientID := os.Getenv("IGDB_CLIENT_ID")
	clientSecret := os.Getenv("IGDB_CLIENT_SECRET")

	if clientID == "" || clientSecret == "" {
		return "", fmt.Errorf("IGDB Client ID or Client Secret is not set in environment variables")
	}

	// Define the OAuth token URL and the parameters
	oauthURL := "https://id.twitch.tv/oauth2/token"
	params := url.Values{}
	params.Add("client_id", clientID)
	params.Add("client_secret", clientSecret)
	params.Add("grant_type", "client_credentials")

	// Make the POST request to get the token
	resp, err := client.R().
		SetHeader("Content-Type", "application/x-www-form-urlencoded").
		SetBody(params.Encode()).
		Post(oauthURL)

	if err != nil {
		return "", err
	}

	// Parse the response
	var token OAuthToken
	if err := json.Unmarshal(resp.Body(), &token); err != nil {
		return "", err
	}

	// Return the access token
	return token.AccessToken, nil
}

// FetchGames calls the IGDB API and returns a list of games
func FetchGames(query, accessToken string) ([]Game, error) {
	client := resty.New()

	// Get credentials from environment variables
	clientID := os.Getenv("IGDB_CLIENT_ID")

	if clientID == "" || accessToken == "" {
		return nil, fmt.Errorf("IGDB Client ID or Access Token is not set")
	}

	// Make a POST request to IGDB API with query
	resp, err := client.R().
		SetHeader("Client-ID", clientID).
		SetHeader("Authorization", fmt.Sprintf("Bearer %s", accessToken)).
		SetBody(fmt.Sprintf(`search "%s"; fields name, first_release_date, summary;`, query)).
		Post("https://api.igdb.com/v4/games")

	if err != nil {
		return nil, err
	}

	// Unmarshal response into a slice of Game
	var games []Game
	if err := json.Unmarshal(resp.Body(), &games); err != nil {
		return nil, err
	}

	return games, nil
}

// handleGameSearch handles requests to search for games
func handleGameSearch(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		http.Error(w, "Query parameter 'q' is required", http.StatusBadRequest)
		return
	}

	// Get OAuth token
	accessToken, err := FetchOAuthToken()
	if err != nil {
		http.Error(w, "Error fetching OAuth token: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Fetch games from IGDB API
	games, err := FetchGames(query, accessToken)
	if err != nil {
		http.Error(w, "Error fetching games: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(games); err != nil {
		http.Error(w, "Error encoding response: "+err.Error(), http.StatusInternalServerError)
	}
}

func main() {
	// Load the .env file
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	// Get port from environment variables, or set default
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Define route
	http.HandleFunc("/games/search", handleGameSearch)

	// Start server
	log.Printf("Server started on port %s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal("ListenAndServe:", err)
	}
}
