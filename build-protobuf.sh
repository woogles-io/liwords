#!/usr/bin/env bash

# you must export CODE_DIR when invoking this script.
# i.e. CODE_DIR=/Users/cesar/code
# CODE_DIR must be a direct parent of `macondo` and `liwords` (this repo)

protoc --plugin="protoc-gen-ts=/opt/node_modules/ts-protoc-gen/bin/protoc-gen-ts" --ts_out=liwords-ui/src/gen  --js_out=import_style=commonjs,binary:liwords-ui/src/gen --proto_path=$CODE_DIR macondo/api/proto/macondo/macondo.proto


for api in "user_service" "game_service" "config_service" "tournament_service" "mod_service" "word_service" "query_service"
do
    protoc --twirp_out=rpc --go_out=rpc --proto_path=$CODE_DIR/ --proto_path=$CODE_DIR/liwords --go_opt=paths=source_relative --twirp_opt=paths=source_relative api/proto/$api/$api.proto
done

for tsapi in "game_service" "user_service" "tournament_service"
do
    protoc --plugin="protoc-gen-ts=/opt/node_modules/ts-protoc-gen/bin/protoc-gen-ts"  --js_out=import_style=commonjs,binary:liwords-ui/src/gen --ts_out=liwords-ui/src/gen --proto_path=$CODE_DIR/ --proto_path=$CODE_DIR/liwords api/proto/$tsapi/$tsapi.proto
done

protoc --go_out=rpc --proto_path=$CODE_DIR/liwords --go_opt=paths=source_relative api/proto/realtime/ipc.proto

protoc --plugin="protoc-gen-ts=/opt/node_modules/ts-protoc-gen/bin/protoc-gen-ts" --go_out=rpc --js_out=import_style=commonjs,binary:liwords-ui/src/gen --ts_out=liwords-ui/src/gen --proto_path=$CODE_DIR/ --proto_path=$CODE_DIR/liwords --go_opt=paths=source_relative api/proto/realtime/realtime.proto
