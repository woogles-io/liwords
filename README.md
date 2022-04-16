# liwords

### License

This source code is AGPL-licensed. You can modify the source for this app, or for apps that communicate with this app through a network, but must make available any of your related code under the same license.

### Components

- liwords (this repo) is an API server, written in Go.
  - liwords-ui (inside this repo) is a TypeScript front-end, built using `create-react-app`
- liwords-socket is a socket server, written in Go. It handles all the real-time communication. It resides at https://github.com/domino14/liwords-socket.
- NATS for pubsub / req-response functionality between liwords, liwords-socket, and the user.
- PostgreSQL

### How to develop locally

1. Download Docker for your operating system (download the Docker preview for M1 Macs, if the full stable version isn't out yet).
2. Download the latest stable version of Node.js for your operating system
3. Clone the `liwords-socket` repository from `https://github.com/domino14/liwords-socket`, and place it at the same level as this repo. For example, if your code resides at `/home/developer/code`, you should have two repos, at `/home/developer/code/liwords` (this repo) and `/home/developer/code/liwords-socket`.
4. Clone the `macondo` repository from `https://github.com/domino14/macondo`, and place it at the same level as this repo.
5. `cd` to this directory

6. Run the following command in one of your terminal tabs, to run the backend, frontend, and databases.

`docker-compose up`

7. Edit your `hosts` file, typically `/etc/hosts`, by adding this line:

```
127.0.0.1	liwords.localhost
```

(If you are on Windows and you want to use Chrome, you cannot use `.localhost`. Use `liwords.local` in your `C:\Windows\System32\drivers\etc\hosts`.)

8. Access the app at http://liwords.localhost
9. If you wish to add a new front-end package, you can run `npm install` LOCALLY (in your host OS) in the `liwords-ui` directory. This adds the package to `package.json`. Then you can do `docker-compose build frontend` to rebuild the frontend and install the package in the internal node_modules directory.
10. You can register a user by going to http://liwords.localhost/ and clicking on `SIGN UP` at the top right.

To have two players play each other you must have one browser window in incognito mode, or use another browser.

11. To register a bot, register a user the regular way. Then change their `internal_bot` flag in the database (`users` table) to true, and restart the server. You need to register at least one bot in order for bot games to work.

#### Tips

You can do `docker-compose up app` and `docker-compose up frontend` in two different terminal windows to bring these up separately. This may be desirable, for example, when making backend changes and not wanting to restart the frontend compilation everytime something changes.

### macondo

`liwords` has a dependency on https://github.com/domino14/macondo

`macondo` provides the logic for the actual crossword board game. `liwords` adds
the web app logic to allow two players to play against each other, or against
a computer, etc.

`macondo` also provides a bot.

### socket

The app requires `liwords-socket` as a socket server. See the instructions above for how to run it alongside this api server.

### protoc

If you change any of the `.proto` files (in this repo or in the Macondo repo) you will need to run the `protoc` compiler to regenerate the appropriate code.

To do so, run in this directory:

`docker-compose run --rm pb_compiler ./build-protobuf.sh`

### Attributions

#### Sounds

This app uses these sounds from freesound:

S: single dog bark 3 by crazymonke9 -- https://freesound.org/s/418105/

#### Code

Part of the front-end timer code borrows from https://github.com/ornicar/lila's code (AGPL licensed, like this app).

Wolges-wasm is Copyright (C) 2020-2021 Andy Kurnia and released under the MIT license. It can be found at https://github.com/andy-k/wolges-wasm/.

### Images

Country flags created by https://hampusborgos.github.io/
