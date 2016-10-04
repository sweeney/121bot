package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/garyburd/redigo/redis"
	"github.com/nlopes/slack"
	"net/http"
	"os"
)

type SlackSlashResponse struct {
	ResponseType string `json:"response_type"`
	Text         string `json:"text"`
}

func main() {

	port := os.Getenv("PORT")
	if port == "" {
		port = "5000"
	}

	http.HandleFunc("/", rootHandler)
	http.HandleFunc("/1:1", oneToOneHandler)
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

func oneToOneHandler(w http.ResponseWriter, r *http.Request) {
	/*
	  token=xyzxyzxyz&team_id=xyzxyzxyz&team_domain=ravelin&channel_id=xyzxyzxyz&channel_name=zztest&user_id=xyzxyzxyz&user_name=sweeney&command=%2F1%3A1&text=now&response_url=xyzxyzxyz
	*/

	teamID := r.FormValue("team_id")
	self := r.FormValue("user_name")

	if teamID == "" {
		http.Error(w, "Missing team_id", http.StatusBadRequest)
		return
	}

	friend, isOffline, err := findAFriend(teamID, self)

	if err != nil {
		http.Error(w, fmt.Sprintf("Error: %s", err.Error()), http.StatusInternalServerError)
		return
	}

	var resp SlackSlashResponse
	resp.ResponseType = "in_channel"

	if isOffline {
		resp.Text = fmt.Sprintf("There's no one around right now, but why not have a 1:1 with @%s when they're back?", friend)
	} else {
		resp.Text = fmt.Sprintf("Why not have a 1:1 with @%s?", friend)
	}

	j, jsonErr := json.Marshal(resp)

	if jsonErr != nil {
		http.Error(w, fmt.Sprintf("Error: %s", jsonErr.Error()), http.StatusInternalServerError)
	}

	fmt.Fprint(w, string(j))

	return
}

func findAFriend(teamID string, self string) (string, bool, error) {

	token, err := getTeamToken(teamID)
	if err != nil {
		return "", false, err
	}

	allUsers, userErr := slack.New(token).GetUsers()
	if userErr != nil {
		return "", false, userErr
	}

	validUsersOnline := make(map[string]string)
	validUsersAll := make(map[string]string)

	for _, user := range allUsers {
		if validUser(user, self, false) {
			validUsersAll[user.ID] = user.Name
		}
		if validUser(user, self, true) {
			validUsersOnline[user.ID] = user.Name
		}
	}

	// Try and see if there's anyone around at the moment
	for _, name := range validUsersOnline {
		return name, false, nil
	}

	// Fall back on all valid users even if they're offline
	for _, name := range validUsersAll {
		return name, true, nil
	}

	return "", false, errors.New("There's no one in your team to talk to!")

}

func validUser(u slack.User, self string, checkActive bool) bool {

	if checkActive && u.Presence != "active" {
		return false
	}

	if u.IsBot || u.IsRestricted || u.IsUltraRestricted || u.Deleted || u.Name == self || u.Name == "slackbot" {
		return false
	}

	return true

}

func getTeamToken(teamID string) (string, error) {

	config, err := getTeamConfig(teamID)
	if err != nil {
		return "", err
	}

	if token, ok := config["token"]; ok {
		return token, nil
	} else {
		return "", errors.New("Couldn't find token for team!")
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
