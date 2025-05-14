#!/bin/bash

apt-get update
apt-get install -y curl tar

CWD=$(pwd)

curl -O https://dl.google.com/dl/cloudsdk/channels/rapid/downloads/google-cloud-cli-linux-x86_64.tar.gz

tar -xf "$CWD/google-cloud-cli-linux-x86_64.tar.gz"

"$CWD/google-cloud-sdk/bin/gcloud" --version
"$CWD/google-cloud-sdk/bin/gcloud" run deploy $SERVICE_NAME \
  --image "$AR_HOSTNAME/$PROJECT_ID/$AR_REPO/$SERVICE_NAME:$COMMIT_SHA" \
  --region $DEPLOY_REGION \
  --platform $PLATFORM
