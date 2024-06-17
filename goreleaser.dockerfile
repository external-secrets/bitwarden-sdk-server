FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY bitwarden-sdk-server /bitwarden-sdk-server
USER 65532:65532

EXPOSE 9998
ENV CGO_ENABLED=1
ENV BW_SECRETS_MANAGER_STATE_PATH='/state'
ENTRYPOINT [ "/bitwarden-sdk-server", "serve" ]
