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

You have two options for developing locally.

1. Using the entire Docker stack is the most straightforward option, but, unless you are on Linux, Docker has to spin up virtual machines for your code. Stopping and starting containers repeatedly, especially the frontend code container, is significantly slower than running these natively; rebuilding containers, etc is also quite slow.

2. The other option is to use Docker for the long-running services (postgres, Redis, NATS), and run your program executables locally. It is a bit more complex to set up initially, but may work better if you are developing on Mac OS (or Windows?).

#### Using the full stack on Docker:

1. Download Docker for your operating system
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

11. To register a bot, run the script in `scripts/utilities/register-bot.sh`. You can run it like this:

`./scripts/utilities/register-bot.sh BotUsername`, replacing BotUsername with your desired bot username.

**Tips**

You can do `docker-compose up app` and `docker-compose up frontend` in two different terminal windows to bring these up separately. This may be desirable, for example, when making backend changes and not wanting to restart the frontend compilation everytime something changes.

#### Hybrid stack on Docker


1. Download Docker for your operating system
2. Download the latest stable version of Node.js for your operating system and install it
3. Download and install Go from golang.org
4. Copy the `local_skeleton.env` file in this directory to `local.env`, and modify the copy to match your local paths. (See all the variables ending in _PATH).
5. Open up a few tabs or panels in your terminal so you can bring up the services separately. In each tab, you can do `source local.env`, or alternatively you can put this command in your profile to do it automatically.
6. Bring up the `dc-local-services.yml` file with `docker-compose -f dc-local-services.yml up` in one tab.
7. You can bring up the other services in your other tabs:
- For the api server, do `go run cmd/liwords-api/*.go`
- For the socket server, do `go run cmd/socketsrv/main.go` in the `liwords-socket` repo.
- For the frontend, do `npm start` in the `liwords-ui` directory.
- For the bot, do `go run cmd/bot/*.go` in the `macondo` directory.

8. Go to `http://localhost:3000` to see Woogles.
9. You can register a user by clicking on `SIGN UP` at the top right.

To have two players play each other you must have one browser window in incognito mode, or use another browser.

10. To register a bot, register a user the regular way. Then run this following script, replacing the `$1` with the bot username you just registered.

`docker-compose exec db psql -U postgres liwords -c "UPDATE users SET internal_bot='t' WHERE username = '$1';"`



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

`docker-compose run --rm goutils ./build-protobuf.sh`

### sqlc

We use `sqlc` for generating Go code from our `.sql` files. If you create new `.sql` files in `db/migrations` or `db/queries` you can rerun sqlc as follows:

`docker-compose run --rm goutils sqlc generate`

### Attributions

#### Sounds

This app uses these sounds from freesound:

S: single dog bark 3 by crazymonke9 -- https://freesound.org/s/418105/

#### Code

Part of the front-end timer code borrows from https://github.com/ornicar/lila's code (AGPL licensed, like this app).

Wolges-wasm is Copyright (C) 2020-2023 Andy Kurnia and released under the MIT license. It can be found at https://github.com/andy-k/wolges-wasm/.

### Images

Country flags created by https://hampusborgos.github.io/
