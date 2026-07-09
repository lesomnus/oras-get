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

## Testing

```sh
go test ./...           # unit + in-memory integration tests (no external services)
go test -tags e2e ./... # end-to-end against a real registry using the oras CLI
```

The `e2e` tests publish artifacts with the `oras` CLI and fetch them back. They use
the registry at `registry:5000` by default (override with `ORAS_GET_E2E_REGISTRY`)
and skip themselves when the CLI or registry is unavailable.
