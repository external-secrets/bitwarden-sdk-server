# -*- mode: Python -*-

kubectl_cmd = "kubectl"

# verify kubectl command exists
if str(local("command -v " + kubectl_cmd + " || true", quiet = True)) == "":
    fail("Required command '" + kubectl_cmd + "' not found in PATH")

install = helm('deploy/charts/bitwarden-sdk-server')

# Apply the updated yaml to the cluster.
k8s_yaml(install, allow_duplicates = True)

load('ext://restart_process', 'docker_build_with_restart')

# enable hot reloading by doing the following:
# - locally build the whole project
# - create a docker imagine using tilt's hot-swap wrapper
# - push that container to the local tilt registry
local_resource(
    'external-secret-binary',
    "CC=x86_64-linux-musl-gcc GOOS=linux GOARCH=amd64 CGO_LDFLAGS='-lm' CGO_ENABLED=1 go build -ldflags '-linkmode external -extldflags -static' -o bin/bitwarden-sdk-server main.go",
    deps = [
        "main.go",
        "go.mod",
        "go.sum",
        "cmd",
        "pkg",
    ],
)


# Build the docker image for our controller. We use a specific Dockerfile
# since tilt can't run on a scratch container.
# `only` here is important, otherwise, the container will get updated
# on _any_ file change. We only want to monitor the binary.
# If debugging is enabled, we switch to a different docker file using
# the delve port.
entrypoint = ['/bitwarden-sdk-server', 'serve']
dockerfile = 'tilt.dockerfile'
docker_build_with_restart(
    'ghcr.io/external-secrets/bitwarden-sdk-server',
    '.',
    dockerfile = dockerfile,
    entrypoint = entrypoint,
    only=[
      './bin',
    ],
    live_update = [
        sync('./bin/bitwarden-sdk-server', '/bitwarden-sdk-server'),
    ],
)
