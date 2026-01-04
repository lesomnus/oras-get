# ORAS-get

Retrieve *OCI* blobs from remote registries with *curl*-like commands.

## Motivation

I wanted to distribute some static files, mostly binaries built for multiple platforms, and I wanted a simple way to host and retrieve them.
OCI registries are a good fit for this purpose, as they provide a standardized way to store and retrieve artifacts.


## APIs

| Method | Path                                    | Description       |
| ------ | --------------------------------------- | ----------------- |
| `GET`  | `[/<domain>]/<path>:<tag>[/<platform>]` | Retrieve artifact |
| `GET`  | `[/<domain>]/<path>:_`                  | List tags         |
