#!/bin/bash
# This is used by docker to build and run stopwatch
# Remove the godep restore line to speed up the time it takes to start the container
# By default it restores all godeps which can take a while
godep restore
go build -o stopwatch *.go
export VAULT_ADDR="http://vault:8200"
export VAULT_TOKEN='b2395c3d-7649-857e-d405-e39aab6b3d2f'
export REDIS_HOST_PORT=redis:6379
export REDIS_PASSWORD=placeholder
# Be sure to put your sendgrid api token here
export SENDGRID_API_TOKEN='REPLACE_ME'
# Replace the below with the email that should be listed in From when sending emails
export EMAIL_FROM_ADDRESS='REPLACE_ME'
export STOPWATCH_URL='http://localhost:4000'
export VAULT_KEY_1='7cf27f7a3ae96ca443e10c0f988de94f70c4a3b786a077252068c1355deba90e01'
export VAULT_KEY_2='7636b428557be640c359f191aa465b59150448c9f9267c3010dd83afcb1ff16302'
export VAULT_KEY_3='ea673554de03b87799fbf2ca9196e4c1398f955921b524f7f525355ec4480b4e03'
/opt/stopwatch/stopwatch
