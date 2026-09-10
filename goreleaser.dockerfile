FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY bitwarden-sdk-server /bitwarden-sdk-server
USER 65532:65532

EXPOSE 9998
ENTRYPOINT [ "/bitwarden-sdk-server", "serve" ]
