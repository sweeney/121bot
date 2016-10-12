# 1:1 bot

- Adds the slash command `/1:1` to your Slack team
- Suggests a random team member for you to have a 1:1 with

## Usage

Can run on Heroku. Uses `govendor`. Needs Redis and the following env variables:

```
- SLACK_CLIENT_ID
- SLACK_CLIENT_SECRET
- REDIS_URL
- PORT
```
