#!/bin/sh

/bin/stripe listen --forward-to http://127.0.0.1:9090/api/v1/webhook/stripe --skip-verify