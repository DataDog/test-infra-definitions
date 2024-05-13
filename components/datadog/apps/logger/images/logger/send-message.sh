#!/usr/bin/env sh
curl -v \
--header 'Content-Type: application/json' \
--data "{ \"data\": [ {\"message\": \"$1\"} ]}" \
localhost:3333/
