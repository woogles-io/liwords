#!/usr/bin/env bash

# you must export CODE_DIR when invoking this script.
# i.e. CODE_DIR=/Users/cesar/code
# CODE_DIR must be a direct parent of `macondo` and `liwords` (this repo)

export PATH="/opt/node_modules/.bin":$PATH

protoc --es_out=liwords-ui/src/gen  --proto_path=$CODE_DIR macondo/api/proto/macondo/macondo.proto


for api in "config_service" "game_service" "mod_service" "puzzle_service" "tournament_service"  "user_service" "word_service"
do
    protoc --twirp_out=rpc --go_out=rpc --proto_path=$CODE_DIR/ --proto_path=$CODE_DIR/liwords --go_opt=paths=source_relative --twirp_opt=paths=source_relative  api/proto/$api/$api.proto
done

for esapi in "config_service" "game_service" "mod_service" "puzzle_service" "tournament_service"  "user_service" "word_service"
do
    protoc  --es_out=liwords-ui/src/gen --proto_path=$CODE_DIR/ --proto_path=$CODE_DIR/liwords --connect-web_out=liwords-ui/src/gen api/proto/$esapi/$esapi.proto
done

for ipcapi in "chat" "errors" "ipc" "omgseeks" "omgwords" "presence" "tournament" "users"
do
    protoc  --es_out=liwords-ui/src/gen --proto_path=$CODE_DIR/ --proto_path=$CODE_DIR/liwords  --go_out=rpc --go_opt=paths=source_relative api/proto/ipc/$ipcapi.proto
done
