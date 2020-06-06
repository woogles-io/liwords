import os
from invoke import task

code_dir = os.getenv("CODE_DIR", "/Users/cesar/code")


@task
def build_protobuf(c):
    c.run(
        "protoc "
        '--plugin="protoc-gen-ts=liwords-ui/node_modules/.bin/protoc-gen-ts" '
        "--ts_out=liwords-ui/src/gen "
        "--js_out=import_style=commonjs,binary:liwords-ui/src/gen "
        f"--proto_path={code_dir} macondo/api/proto/macondo/macondo.proto"
    )
    c.run(
        "protoc "
        '--plugin="protoc-gen-ts=liwords-ui/node_modules/.bin/protoc-gen-ts" '
        "--twirp_out=rpc --go_out=rpc "
        "--js_out=import_style=commonjs,binary:liwords-ui/src/gen "
        f"--ts_out=liwords-ui/src/gen --proto_path={code_dir}/ "
        f"--proto_path={code_dir}/crosswords "
        "--go_opt=paths=source_relative "
        "--twirp_opt=paths=source_relative api/proto/game_service.proto"
    )

    # Prepend line to disable eslint to generated files. It doesn't work
    # if I put them in the .eslintignore file for some reason.
    for gen_filename in (
        "liwords-ui/src/gen/macondo/api/proto/macondo/macondo_pb.js",
        "liwords-ui/src/gen/api/proto/game_service_pb.js",
    ):
        tmp = c.run("mktemp").stdout.strip()
        c.run(r'printf "/* eslint-disable */\n" > ' + tmp)
        c.run(f"cat {gen_filename} >> " + tmp)
        c.run(f"mv {tmp} {gen_filename}")


@task
def deploy(c):
    with c.cd("liwords-ui"):
        c.run("yarn build")
        c.run("rsync -avz --del build/ ubuntu@xword.club:~/liwords-ui-build")
    with c.cd("cmd/server"):
        c.run("GOOS=linux GOARCH=amd64 go build -o liwords-linux-amd64")
        c.run("scp liwords-linux-amd64 ubuntu@xword.club:.")
