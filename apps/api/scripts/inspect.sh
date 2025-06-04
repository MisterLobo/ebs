#!/bin/sh

atlas schema inspect --env gorm --url env://src --format '{{ sql . }}'