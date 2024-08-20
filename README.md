# bitwarden-sdk-server

This repository contains a simple REST wrapper for the Bitwarden Rust SDK.

## Purpose

The main purpose of this API is to accommodate the needs for [External Secrets Operator](https://external-secrets.io) to
talk to Bitwarden Secrets Manager.

The API is slim and follows basic REST principles. The following endpoints are supported with sample requests:


### GetSecret

`/rest/api/1/secret`

Method `GET`.

```json
{
  "id": "f5847eef-2f89-43bc-885a-b18a01178e3e"
}
```

Response:
```json
{
  "creationDate": "2024-04-04",
  "id": "f5847eef-2f89-43bc-885a-b18a01178e3e",
  "key": "test",
  "note": "note",
  "organizationId": "f5847eef-2f89-43bc-885a-b18a01178e3e",
  "revisionDate": "2024-04-04",
  "value": "value"
}
```

### GetSecretsByIds

`/rest/api/1/secrets-by-ids`

Method `GET`.

```json
{
  "ids": [
    "f5847eef-2f89-43bc-885a-b18a01178e3e", "0cab75c4-ba26-4996-a8bf-517095857ce3"
  ]
}
```

Response:
```json
{
  "data": [
    {
      "creationDate": "2024-04-04",
      "id": "f5847eef-2f89-43bc-885a-b18a01178e3e",
      "key": "test",
      "note": "note",
      "organizationId": "f5847eef-2f89-43bc-885a-b18a01178e3e",
      "revisionDate": "2024-04-04",
      "value": "value"
    },
    {
      "creationDate": "2024-04-05",
      "id": "0cab75c4-ba26-4996-a8bf-517095857ce3",
      "key": "test2",
      "note": "note2",
      "organizationId": "f5847eef-2f89-43bc-885a-b18a01178e3e",
      "revisionDate": "2024-04-05",
      "value": "value2"
    }
  ]
}
```

### ListSecrets

`/rest/api/1/secrets`

Method `GET`.

```json
{
  "organizationId": "f5847eef-2f89-43bc-885a-b18a01178e3e"
}
```

Response:
```json
{
  "data":[
    {
      "id": "1ba2f0c9-d73d-48bf-84a5-290ce5012258",
      "organizationId": "f5847eef-2f89-43bc-885a-b18a01178e3e",
      "key": "this-is-the-name"
    }
  ]
}
```

### UpdateSecret

`rest/api/1/secret`

Method `PUT`.

```json
{
  "id": "1ba2f0c9-d73d-48bf-84a5-290ce5012258",
  "key": "name",
  "note": "new-note",
  "organizationId": "f5847eef-2f89-43bc-885a-b18a01178e3e",
  "value": "new-value"
}
```

Response:

```json
{
  "creationDate": "2024-04-04",
  "id": "1ba2f0c9-d73d-48bf-84a5-290ce5012258",
  "key": "test",
  "note": "note",
  "organizationId": "f5847eef-2f89-43bc-885a-b18a01178e3e",
  "revisionDate": "2024-04-04",
  "value": "value"
}
```

### CreateSecret

`rest/api/1/secret`

Method `POST`.

```json
{
  "key": "name",
  "note": "note",
  "organizationId": "f5847eef-2f89-43bc-885a-b18a01178e3e",
  "value": "value"
}
```

Response:

```json
{
  "creationDate": "2024-04-04",
  "id": "f5847eef-2f89-43bc-885a-b18a01178e3e",
  "key": "name",
  "note": "note",
  "organizationId": "f5847eef-2f89-43bc-885a-b18a01178e3e",
  "revisionDate": "2024-04-04",
  "value": "value"
}
```

## Authentication

The router is using a middleware called `Warden` that will create an authenticated client for all the requests.
This client is created through the use of Headers. The following headers can be provided for each call:

```
Warden-Access-Token: <token> // mandatory
Warden-State-Path: <state-path>
Warden-Api-Url: <url>
Warden-Identity-Url: <url>
```

A sample call could look something like this:

```
curl --insecure -d '{"key": "test2", "value": "secret","note": "shit", "organizationId": "ac2b00ac-2ef7-4d86-8cbd-b18a011760cb", "projectIds":[
"f5847eef-2f89-43bc-885a-b18a01178e3e"]}' https://chart-bitwarden-sdk-server.default.svc.cluster.local:9998/rest/api/1/secret --header 'Warden-Acce
ss-Token:<token>' -X POST
```

## Install

The server is a dependency to external-secrets' helm chart, therefor it can be installed together with ESO like this:

```
helm install external-secrets \
   external-secrets/external-secrets \
    -n external-secrets \
    --create-namespace \
    --set bitwarden-sdk-server.enabled=true
```

Or, it can also be installed in a standalone way using helm from this repository.

The server **MUST** run using HTTPS. A recommended way to generate a certificate is to use cert-manager.
The certificate can be defined in a Kubernetes secret called `bitwarden-tls-certs`. This can be overwritten in the helm
chart values file.

The certificate will then be required when using external-secrets' Bitwarden provider.

## Certificates

There are many ways to generate secrets for an HTTP server. One of which could be through cert-manager.

That process can be found under the `hack` folder. But using an existing certificate is also possible through helm
values. These are mounted inside the container and used further by the client with keys defined by the following
command line arguments:

```go
	flag.StringVar(&rootArgs.server.KeyFile, "key-file", "/certs/key.pem", "--key-file /certs/key.pem")
	flag.StringVar(&rootArgs.server.CertFile, "cert-file", "/certs/cert.pem", "--cert-file /certs/cert.pem")
```

The certificate mount target and values are defined under `image` section in the values file as such:

```yaml
image:
  repository: ghcr.io/external-secrets/bitwarden-sdk-server
  pullPolicy: IfNotPresent
  # Overrides the image tag whose default is the chart appVersion.
  tag: ""
  tls:
    enabled: true
    volumeMounts:
      - mountPath: "/certs"
        name: "bitwarden-tls-certs"
    volumes:
      - name: "bitwarden-tls-certs"
        secret:
          secretName: "bitwarden-tls-certs"
          items:
            - key: "tls.crt"
              path: "cert.pem"
            - key: "tls.key"
              path: "key.pem"
            - key: "ca.crt"
              path: "ca.pem"
```

To use cert-manager the `hack` folder sets up the following certificate issuer:

```yaml
apiVersion: cert-manager.io/v1
kind: ClusterIssuer
metadata:
  name: bitwarden-bootstrap-issuer
spec:
  selfSigned: {}
---
apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  name: bitwarden-bootstrap-certificate
  namespace: cert-manager
spec:
  # this is discouraged but required by ios
  commonName: cert-manager-bitwarden-tls
  isCA: true
  secretName: bitwarden-tls-certs
  subject:
    organizations:
      - external-secrets.io
  dnsNames:
    - external-secrets-bitwarden-sdk-server.default.svc.cluster.local
    - bitwarden-sdk-server.default.svc.cluster.local
    - localhost
  ipAddresses:
    - 127.0.0.1
    - ::1
  privateKey:
    algorithm: RSA
    encoding: PKCS8
    size: 2048
  issuerRef:
    name: bitwarden-bootstrap-issuer
    kind: ClusterIssuer
    group: cert-manager.io
---
apiVersion: cert-manager.io/v1
kind: ClusterIssuer
metadata:
  name: bitwarden-certificate-issuer
spec:
  ca:
    secretName: bitwarden-tls-certs
```

The important bits are the `dnsNames`. The first one is with the external-secrets helm release name, and the second one
is a plain install. But also, external-secrets pins the release name of bitwarden, so that should work too. This will
create a self-signed certificate for us to use internally. This certificate will later be provided to external-secrets
so it can talk to the service.

Next, we create a Certificate for bitwarden with the following request:

```yaml
apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  name: bitwarden-tls-certs
  namespace: default
spec:
  secretName: bitwarden-tls-certs
  dnsNames:
    - bitwarden-sdk-server.default.svc.cluster.local
    - external-secrets-bitwarden-sdk-server.default.svc.cluster.local
    - localhost
  ipAddresses:
    - 127.0.0.1
    - ::1
  privateKey:
    algorithm: RSA
    encoding: PKCS8
    size: 2048
  issuerRef:
    name: bitwarden-certificate-issuer
    kind: ClusterIssuer
    group: cert-manager.io
```

This is provided to bitwarden to initialize an HTTPS server.

### External-secrets

On external-secrets side, there are two options to provide the certificate.

One is through `caBundle` which accepts the plain root certificate as a base64 encoded value.

Second is through `caProvider` that uses either a secret or a configmap and looks for the right key.

**_WARNING_**: DO NOT provide the same secret as the server. For more detail read [cert-manager Trust Post](https://cert-manager.io/docs/trust/).

### Insecure

For testing purposes, or if you trust your network that much, an `--insecure` flag has been provided that runs this
server as plain HTTP.

## Testing

Run `make prime-test-cluster` to launch a cluster and generate a certificate for the service. One done, simply run tilt
to create the service. Note OSX users must install https://github.com/FiloSottile/homebrew-musl-cross in order to
build the CGO library.

## External-secrets documentation

Usage on the external-secrets side is documented under [Bitwarden Secrets Manager Provider](https://external-secrets.io/latest/provider/bitwarden-secrets-manager/).

## License

[![FOSSA Status](https://app.fossa.com/api/projects/git%2Bgithub.com%2Fexternal-secrets%2Fbitwarden-sdk-server.svg?type=large&issueType=license)](https://app.fossa.com/projects/git%2Bgithub.com%2Fexternal-secrets%2Fbitwarden-sdk-server?ref=badge_large&issueType=license)
