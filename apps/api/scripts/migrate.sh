#!/bin/sh

export DB_PASSWORD=password
atlas schema apply -c file://atlas.hcl --env gorm