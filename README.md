# crosswords

### protoc

To generate pb files, run in this directory:

`protoc --twirp_out=rpc --go_out=rpc --proto_path=/Users/cesar/code/ --proto_path=/Users/cesar/code/crosswords --go_opt=paths=source_relative --twirp_opt=paths=source_relative api/proto/game_service.proto`

(note, you'll have to change the proto_path to match your folder layout. Make sure that `crosswords` and `macondo` are both inside the supplied `proto_path`)
