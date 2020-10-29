#! /bin/bash

curl --request POST \
  --url $OAUTH_TOKEN_ENDPOINT \
  --header 'content-type: application/x-www-form-urlencoded' \
  --data grant_type=client_credentials \
  --data "client_id=$OAUTH_CLIENT_ID" \
  --data client_secret=$OAUTH_CLIENT_SECRET \
  --data audience=$OAUTH_AUDIENCE
