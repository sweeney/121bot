package main

import (
	"fmt"
	"github.com/nlopes/slack"
	"net/http"
	"os"
)

func main() {

	port := os.Getenv("PORT")
	if port == "" {
		port = "5000"
	}

	http.HandleFunc("/", rootHandler)
	//http.HandleFunc("/1:1", OneToOneHandler)
	http.HandleFunc("/oauth/", oauthHandler)

	http.ListenAndServe(":"+port, nil)

}

func rootHandler(w http.ResponseWriter, r *http.Request) {

	clientID := os.Getenv("SLACK_CLIENT_ID")

	if clientID == "" {
		http.Error(w, "Missing clientID in config", http.StatusInternalServerError)
	}

	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, "121bot: <a href=\"https://slack.com/oauth/authorize?scope=commands+users:read&client_id=%s\">Add to slack</a>\n", clientID)
	return

}

// oauthHandler manages the OAuth 2.0 handshake and exchanges a code for an access token
func oauthHandler(w http.ResponseWriter, r *http.Request) {

	clientID := os.Getenv("SLACK_CLIENT_ID")
	clientSecret := os.Getenv("SLACK_CLIENT_SECRET")
	code := r.FormValue("code")

	if code != "" {

		authResponse, err := slack.GetOAuthResponse(clientID, clientSecret, code, "", false)

		if err == nil {

			fmt.Fprintf(w, "Team: %s(%s), Access token: %s, scope: %s", authResponse.TeamName, authResponse.TeamID, authResponse.AccessToken, authResponse.Scope)

		} else {
			http.Error(w, fmt.Sprintf("Error: %s", err.Error()), http.StatusBadRequest)
		}

	} else {
		http.Error(w, "Missing code", http.StatusBadRequest)
	}

}
