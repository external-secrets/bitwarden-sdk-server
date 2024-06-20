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

## Testing

Run `make prime-test-cluster` to launch a cluster and generate a certificate for the service. One done, simply run tilt
to create the service. Note OSX users must install https://github.com/FiloSottile/homebrew-musl-cross in order to
build the CGO library.

## License

[![FOSSA Status](https://app.fossa.com/api/projects/git%2Bgithub.com%2Fexternal-secrets%2Fbitwarden-sdk-server.svg?type=large&issueType=license)](https://app.fossa.com/projects/git%2Bgithub.com%2Fexternal-secrets%2Fbitwarden-sdk-server?ref=badge_large&issueType=license)
