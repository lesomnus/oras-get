# ORAS-get

Retrieve *OCI* blobs from remote registries with *curl*-like commands.

## Motivation

I wanted to distribute some static files, mostly binaries built for multiple platforms, and I wanted a simple way to host and retrieve them.
OCI registries are a good fit for this purpose, as they provide a standardized way to store and retrieve artifacts.


## APIs

| Method | Path                             | Description                                  |
| ------ | -------------------------------- | -------------------------------------------- |
| `GET`  | `[/<domain>]/<repo>:<tag>`       | Retrieve the file of a single-file artifact  |
| `GET`  | `[/<domain>]/<repo>:<tag>/<file>`| Retrieve a specific file from the artifact   |
| `GET`  | `[/<domain>]/<repo>:<tag>/`      | List the files in the artifact               |
| `GET`  | `[/<domain>]/<repo>:_`           | List tags                                    |

A `<file>` is matched against a layer's `org.opencontainers.image.title` annotation
(the file name recorded by `oras push`). A single-file artifact needs no `<file>`;
a multi-file artifact returns `400` unless a `<file>` is given.

The optional `?platform=<os>[/<arch>[/<variant>]]` query selects a manifest out of
an image index, e.g. `GET /ghcr.io/me/tools:v1/bin/tool?platform=linux/amd64`.

```sh
# single-file artifact
curl -O http://localhost:5001/ghcr.io/me/tool:v1

# pick one file out of a multi-file artifact
curl -O http://localhost:5001/ghcr.io/me/bundle:v1/bin/tool

# list what's inside
curl http://localhost:5001/ghcr.io/me/bundle:v1/

# a specific platform from a multi-arch index
curl -O "http://localhost:5001/ghcr.io/me/bundle:v1/bin/tool?platform=linux/arm64"
```

## Caching

`GET` and `HEAD` are both supported. A served file carries an `ETag` equal to
its OCI layer digest (a strong, content-addressed validator) plus
`Cache-Control: no-cache`, so clients cache the bytes and revalidate cheaply:

```sh
# HEAD fetches only the metadata (Content-Type, Content-Length, ETag) — no body
curl -I http://localhost:5001/ghcr.io/me/bundle:v1/bin/tool

# a conditional request returns 304 Not Modified when the file is unchanged,
# without re-downloading the blob from the upstream registry
curl -H 'If-None-Match: "sha256:..."' http://localhost:5001/ghcr.io/me/bundle:v1/bin/tool
```

Because the ETag is the layer digest, it stays stable across tag re-pushes as
long as the file's bytes do not change.

## Testing

```sh
go test ./...           # unit + in-memory integration tests (no external services)
go test -tags e2e ./... # end-to-end against a real registry using the oras CLI
```

The `e2e` tests publish artifacts with the `oras` CLI and fetch them back. They use
the registry at `registry:5000` by default (override with `ORAS_GET_E2E_REGISTRY`)
and skip themselves when the CLI or registry is unavailable.
