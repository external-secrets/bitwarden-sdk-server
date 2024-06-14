FROM alpine@sha256:77726ef6b57ddf65bb551896826ec38bc3e53f75cdde31354fbffb4f25238ebd
WORKDIR /
COPY ./bin/bitwarden-sdk-server /bitwarden-sdk-server

ENV CGO_ENABLED=1
ENV BW_SECRETS_MANAGER_STATE_PATH='/state'
ENTRYPOINT ["/bitwarden-sdk-server", "serve", "--insecure"]
