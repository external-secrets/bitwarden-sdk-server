FROM golang:1.27 AS build
WORKDIR /workspace
COPY . .
RUN apt update && apt install unzip musl-tools ca-certificates -y
RUN update-ca-certificates
RUN make build-docker

FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=build /workspace/bin/bitwarden-sdk-server .

EXPOSE 9998
ENTRYPOINT [ "/bitwarden-sdk-server", "serve" ]
