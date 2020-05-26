# liwords

### macondo

`liwords` has a dependency on https://github.com/domino14/macondo

`macondo` provides the logic for the actual crossword board game. `liwords` adds
the web app logic to allow two players to play against each other, or against
a computer, etc.

### protoc

To generate pb files, run in this directory:

`protoc --plugin="protoc-gen-ts=liwords-ui/node_modules/.bin/protoc-gen-ts" --twirp_out=rpc --go_out=rpc --js_out=import_style=commonjs,binary:liwords-ui/src/gen --ts_out=liwords-ui/src/gen --proto_path=/Users/cesar/code/ --proto_path=/Users/cesar/code/crosswords --go_opt=paths=source_relative --twirp_opt=paths=source_relative api/proto/game_service.proto`

`protoc --plugin="protoc-gen-ts=liwords-ui/node_modules/.bin/protoc-gen-ts" --ts_out=liwords-ui/src/gen --js_out=import_style=commonjs,binary:liwords-ui/src/gen --proto_path=/Users/cesar/code macondo/api/proto/macondo/macondo.proto`

(note, you'll have to change the proto_path to match your folder layout. Make sure that `crosswords` and `macondo` are both inside the supplied `proto_path`)

** IMPORTANT **
Finally, in order to get this to compile, you have to manually add this line: `/* eslint-disable */` to the top of any generated \*\_pb.js files

At some point we should make this automatic.
