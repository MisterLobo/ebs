#!/bin/sh

API_KEY=$1
PROJECT_ID=$2
SENDER_ID=$3
APP_ID=$4
FILEPATH=public/firebase-messaging-sw.js

sed -i -e "s/API_KEY/"$API_KEY"/" $FILEPATH
sed -i -e "s/PROJECT_ID/"$PROJECT_ID"/" $FILEPATH
sed -i -e "s/SENDER_ID/"$SENDER_ID"/" $FILEPATH
sed -i -e "s/APP_ID/"$APP_ID"/" $FILEPATH