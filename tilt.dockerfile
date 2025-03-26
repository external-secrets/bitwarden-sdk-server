FROM ubuntu@sha256:72297848456d5d37d1262630108ab308d3e9ec7ed1c3286a32fe09856619a782
WORKDIR /
COPY ./bin/bitwarden-sdk-server /bitwarden-sdk-server

ENV CGO_ENABLED=1
ENV BW_SECRETS_MANAGER_STATE_PATH='/state'
ENTRYPOINT ["/bitwarden-sdk-server", "serve"]
