# liwords

### How to develop locally

1. Download Docker for your operating system
2. Download the latest stable version of Node.js for your operating system
3. `cd` to this directory
4. Run the following command in one of your terminal tabs, to run the backend:

`docker-compose up`

5. In another terminal tab, `cd liwords-ui`, `npm install`, then `npm run start`
6. This should open a browser window once the front end is done compiling. If not, you can see it at `http://localhost:3000`
7. On further runs you will _not_ need to do `npm install` again, unless new packages have been added to the front-end.

### macondo

`liwords` has a dependency on https://github.com/domino14/macondo

`macondo` provides the logic for the actual crossword board game. `liwords` adds
the web app logic to allow two players to play against each other, or against
a computer, etc.

### protoc

If you change any of the `.proto` files (in this repo or in the Macondo repo) you will need to run the `protoc` compiler to regenerate the appropriate code.

To do so, run in this directory:

`inv build-protobuf`

You must have the Python `invoke` program installed (`pip install invoke`)

See the `tasks.py` file to see how this function works.

(note, you'll have to change the proto_path to match your folder layout. Make sure that `liwords` and `macondo` are both inside the supplied `proto_path`)
