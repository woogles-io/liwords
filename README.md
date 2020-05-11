# liwords

### macondo

`liwords` has a dependency on https://github.com/domino14/macondo

`macondo` provides the logic for the actual crossword board game. `liwords` adds
the web app logic to allow two players to play against each other, or against
a computer, etc.

### protoc

To generate pb files, run in this directory:

`protoc --twirp_out=rpc --go_out=rpc --proto_path=/Users/cesar/code/ --proto_path=/Users/cesar/code/crosswords --go_opt=paths=source_relative --twirp_opt=paths=source_relative api/proto/game_service.proto`

(note, you'll have to change the proto_path to match your folder layout. Make sure that `crosswords` and `macondo` are both inside the supplied `proto_path`)
