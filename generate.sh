#!/bin/bash

go run main.go

docker run --rm \
  --user $(id -u):$(id -g) \
  -v $(PWD)/..:/local openapitools/openapi-generator-cli:v7.1.0 generate \
  -i /local/james-openapi-schema/api.yaml \
  -g go \
  --git-user-id searchlight \
  --git-repo-id james-go-client \
  -o /local/james-go-client
