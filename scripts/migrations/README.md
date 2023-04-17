Migration scripts.

1. `cd` to the directory of the script and

`CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-extldflags=-static"`

2. `scp` the executable to the ECS host machine

3. `docker cp my-script mycontainer:/my-script`

4. `docker exec mycontainer sh`

5. Run the script `./my-script` in the container.
