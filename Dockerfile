FROM golang:1.22 as build
WORKDIR /workspace
COPY . .
RUN mkdir state
RUN apt update && apt install unzip musl-tools ca-certificates -y
RUN update-ca-certificates
RUN make build-docker

FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=build /workspace/bin/bitwarden-sdk-server .
COPY --from=build --chown=65532:65532 /workspace/state/ ./state/

EXPOSE 9998
ENV CGO_ENABLED=1
ENV BW_SECRETS_MANAGER_STATE_PATH='/state'
ENTRYPOINT [ "/bitwarden-sdk-server", "serve" ]
