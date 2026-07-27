# liwords

### License

This source code is AGPL-licensed. You can modify the source for this app, or for apps that communicate with this app through a network, but must make available any of your related code under the same license.

### Components

- liwords (this repo) is an API server, written in Go.
  - liwords-ui (inside this repo) is a TypeScript front-end. It is built with `rsbuild`.
  - services/socketsrv (inside this repo) is a socket server, written in Go. It handles all the real-time communication.
- NATS for pubsub / req-response functionality between liwords, socketsrv, and the user.
- PostgreSQL

### How to develop locally

You have two options for developing locally.

1. Using the entire Docker stack is the most straightforward option, but, unless you are on Linux, Docker has to spin up virtual machines for your code. Stopping and starting containers repeatedly, especially the frontend code container, is significantly slower than running these natively; rebuilding containers, etc is also quite slow.

2. The other option is to use Docker for the long-running services (postgres, Redis, NATS), and run your program executables locally. It is a bit more complex to set up initially, but may work better if you are developing on Mac OS (or Windows?).

<details>
<summary>Using the full stack on Docker</summary>

#### Using the full stack on Docker:

1. Download Docker for your operating system
2. Clone the `macondo` repository from `https://github.com/domino14/macondo`, and place it at the same level as this repo.
3. `cd` to this directory

4. Run the following command in one of your terminal tabs, to run the backend, frontend, and databases.

`docker compose up`

5. Edit your `hosts` file, typically `/etc/hosts`, by adding this line:

```
127.0.0.1	liwords.localhost
```

(If you are on Windows and you want to use Chrome, you cannot use `.localhost`. Use `liwords.local` in your `C:\Windows\System32\drivers\etc\hosts`.)

6. Access the app at http://liwords.localhost
7. If you wish to add a new front-end package, you need to run `npm i` INSIDE the Docker container. You can do this like: `docker compose exec frontend npm i` when the docker compose is up.
8. You can register a user by going to http://liwords.localhost/ and clicking on `SIGN UP` at the top right.

To have two players play each other you must have one browser window in incognito mode, or use another browser.

9. To register a bot, run the script in `scripts/utilities/register-bot.sh`. You can run it like this:

`./scripts/utilities/register-bot.sh BotUsername`, replacing BotUsername with your desired bot username.

**Tips**

You can do `docker compose up app` and `docker compose up frontend` in two different terminal windows to bring these up separately. This may be desirable, for example, when making backend changes and not wanting to restart the frontend compilation everytime something changes.

</details>

<details>

<summary>Using a hybrid stack on Docker</summary>

In this setup, Docker runs only the long-running services (postgres, Redis, NATS, via `dc-local-services.yml`), and the Go servers and frontend run natively on your machine. The frontend dev server (port 3000) proxies `/api` and `/ws`.

1. Download Docker for your operating system
2. Download the latest stable version of Node.js for your operating system and install it
3. Download and install Go from golang.org
4. Clone the `macondo` repository from `https://github.com/domino14/macondo`, and place it at the same level as this repo.
5. Copy the `local_skeleton.env` file in this directory to `local.env`, and modify the copy to match your local paths. (See all the variables ending in \_PATH).
6. Run `./scripts/dev-hybrid.sh`. It brings up the Docker services, waits for postgres, and then runs the API server, socket server, and frontend together in one terminal, with prefixed logs. Add `--bot` to also run the macondo bot. Ctrl-C stops the local processes (the Docker services are left running; stop them with `docker compose -f dc-local-services.yml stop`).
7. Go to `http://localhost:3000` to see Woogles.
8. You can register a user by clicking on `SIGN UP` at the top right.

To have two players play each other you must have one browser window in incognito mode, or use another browser.

9. To register a bot, register a user the regular way. Then run the following, replacing the `$1` with the bot username you just registered.

`docker compose -f dc-local-services.yml exec db psql -U postgres liwords -c "UPDATE users SET internal_bot='t' WHERE username = '$1';"`

**Running the services by hand**

If you'd rather run the services in separate terminal tabs instead of using the script, do `source local.env` in each tab, bring up `docker compose -f dc-local-services.yml up` in one of them, and then:

- For the api server, do `go run cmd/liwords-api/*.go`
- For the socket server, do `go run cmd/socketsrv/main.go`
- For the frontend, do `npm start` in the `liwords-ui` directory.
- For the bot, do `go run cmd/bot/*.go` in the `macondo` directory.

**Notes and troubleshooting**

- Both `docker-compose.yml` and `dc-local-services.yml` run under the same Docker compose project name, so they share the same postgres data volume. Keep the `db` service definitions (especially the pinned image version) in sync between the two files — a newer postgres major version will refuse to start on a data volume initialized by an older one. For the same reason, don't run both stacks at the same time (they also both publish NATS on port 4222).
- _ERROR: relation "users" does not exist_: migrations haven't run. Make sure `RUN_MIGRATIONS=1` is in your `local.env` (see `local_skeleton.env`) and restart the API server.
- _"Please verify your email address" on login_: the user was registered while the API server was running without `SKIP_EMAIL_VERIFICATION=1`. Add it and restart, then mark the existing user verified: `docker compose -f dc-local-services.yml exec db psql -U postgres liwords -c "UPDATE users SET verified='t' WHERE username='<name>';"`

</details>

### macondo

`liwords` has a dependency on https://github.com/domino14/macondo

`macondo` provides the logic for the actual crossword board game. `liwords` adds
the web app logic to allow two players to play against each other, or against
a computer, etc.

`macondo` also provides a bot.

### protoc

If you change any of the `.proto` files (in this repo or in the Macondo repo) you will need to run the `protoc` compiler to regenerate the appropriate code.

To do so, run in this directory:

`go generate`

### sqlc

We use `sqlc` for generating Go code from our `.sql` files. If you create new `.sql` files in `db/migrations` or `db/queries` you can rerun sqlc as follows:

`go generate`

### Attributions

#### Sounds

This app uses these sounds from freesound:

S: single dog bark 3 by crazymonke9 -- https://freesound.org/s/418105/

#### Code

Part of the front-end timer code borrows from https://github.com/ornicar/lila's code (AGPL licensed, like this app).

Wolges-wasm is Copyright (C) 2020-2026 Andy Kurnia and released under the MIT license. It can be found at https://github.com/andy-k/wolges-wasm/.

### Images

Country flags created by https://hampusborgos.github.io/
