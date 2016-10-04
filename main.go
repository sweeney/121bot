package main

import (
	"errors"
	"fmt"
	"github.com/garyburd/redigo/redis"
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

			config, configError := makeConfig(authResponse)
			if configError != nil {
				http.Error(w, fmt.Sprintf("Error: %s", configError.Error()), http.StatusInternalServerError)
			}

			storageError := storeTeamConfig(config, authResponse.TeamID)
			if storageError != nil {
				http.Error(w, fmt.Sprintf("Error: %s", storageError.Error()), http.StatusInternalServerError)
			}

			fmt.Fprintf(w, "Great success! Stored all creds for %s.\n", authResponse.TeamName)
			return

		} else {
			http.Error(w, fmt.Sprintf("Error: %s", err.Error()), http.StatusBadRequest)
		}

	} else {
		http.Error(w, "Missing code", http.StatusBadRequest)
	}

}

func makeConfig(response *slack.OAuthResponse) (map[string]string, error) {
	config := make(map[string]string)

	if response.TeamName == "" {
		return nil, errors.New("Missing TeamName in response")
	}
	config["name"] = response.TeamName

	if response.TeamID == "" {
		return nil, errors.New("Missing TeamID in response")
	}
	config["ID"] = response.TeamID

	if response.AccessToken == "" {
		return nil, errors.New("Missing AccessToken in response")
	}
	config["token"] = response.AccessToken

	if response.Scope == "" {
		return nil, errors.New("Missing Scope in response")
	}
	config["scope"] = response.Scope

	return config, nil
}

func getTeamConfig(teamID string) (map[string]string, error) {

	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		return nil, errors.New("REDIS_URL not configured")
	}

	c, err := redis.DialURL(redisURL)
	if err != nil {
		return nil, err
	}
	defer c.Close()

	return redis.StringMap(c.Do("HGETALL", "team:"+teamID))
}

func storeTeamConfig(config map[string]string, teamID string) error {

	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		return errors.New("REDIS_URL not configured")
	}

	c, err := redis.DialURL(redisURL)
	if err != nil {
		return err
	}
	defer c.Close()

	var args []interface{}
	args = append(args, "team:"+teamID)

	for f, v := range config {
		args = append(args, f, v)
	}

	_, setErr := c.Do("HMSET", args...)
	return setErr

}
