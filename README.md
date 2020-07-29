# liwords

### Components

- liwords (this repo) is an API server
  - liwords-ui (inside this repo) is a TypeScript front-end, built using `create-react-app`
- liwords-socket is a socket server. It handles all the real-time communication. It resides at https://github.com/domino14/liwords-socket
- NATS for pubsub / req-response functionality between liwords, liwords-socket, and the user.
- PostgreSQL

### How to develop locally

1. Download Docker for your operating system
2. Download the latest stable version of Node.js for your operating system
3. Clone the `liwords-socket` repository from `https://github.com/domino14/liwords-socket`, and place it at the same level as this repo. For example, if your code resides at `/home/developer/code`, you should have two repos, at `/home/developer/code/liwords` (this repo) and `/home/developer/code/liwords-socket`.
4. `cd` to this directory
5. Run the following command in one of your terminal tabs, to run the backend, frontend, and databases.

`docker-compose up`

6. Edit your `hosts` file, typically `/etc/hosts`, by adding this line:

```
127.0.0.1	liwords.localhost
```

7. Access the dashboard at http://liwords.localhost
8. If you wish to add a new front-end package, you can run `npm install` LOCALLY (in your host OS) in the `liwords-ui` directory. This adds the package to `package.json`. Then you can do `docker-compose build frontend` to rebuild the frontend and install the package in the internal node_modules directory.

#### Tips

You can do `docker-compose up app` and `docker-compose up frontend` in two different terminal windows to bring these up separately. This may be desirable, for example, when making backend changes and not wanting to restart the frontend compilation everytime something changes.

### macondo

`liwords` has a dependency on https://github.com/domino14/macondo

`macondo` provides the logic for the actual crossword board game. `liwords` adds
the web app logic to allow two players to play against each other, or against
a computer, etc.

### socket

The app requires `liwords-socket` as a socket server. See the instructions above for how to run it alongside this api server.

### protoc

If you change any of the `.proto` files (in this repo or in the Macondo repo) you will need to run the `protoc` compiler to regenerate the appropriate code.

To do so, run in this directory:

`inv build-protobuf`

You must have the Python `invoke` program installed (`pip install invoke`)

See the `tasks.py` file to see how this function works.

(note, you'll have to change the proto_path to match your folder layout. Make sure that `liwords` and `macondo` are both inside the supplied `proto_path`)

### Attributions

This app uses these sounds from freesound:

S: single dog bark 3 by crazymonke9 -- https://freesound.org/s/418105/
