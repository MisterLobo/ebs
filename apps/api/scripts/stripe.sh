#!/bin/sh

/bin/stripe listen --forward-to http://127.0.0.1:9090/webhook/stripe --skip-verify