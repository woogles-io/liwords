Migration scripts.

1. `cd` to the directory of the script and

`GOOS=linux GOARCH=amd64 go build`

2. `scp` the executable to the ECS host machine

3. `docker cp my-script mycontainer:/my-script`

4. `docker exec -it mycontainer bash`

5. Run the script `./my-script` in the container.
