# Epoch

A distributed job scheduler built with Go and gRPC. Jobs are defined with a Docker image, a command, and a repeat interval. The server dispatches scheduled jobs to connected worker nodes, which execute them inside Docker containers.

## Architecture

| Component  | Role                                                                                                                      |
| ---------- | ------------------------------------------------------------------------------------------------------------------------- |
| **Server** | Accepts job submissions, maintains a persistent job queue (BadgerDB), and streams jobs to workers over mTLS-secured gRPC. |
| **Worker** | Connects to the server, receives jobs, and runs them via the local Docker daemon.                                         |
| **Client** | CLI tool for submitting jobs to the server.                                                                               |

## Prerequisites

- Go 1.24+
- Docker (required on worker nodes)
- OpenSSL (for certificate generation)

## Setup

**1. Generate TLS certificates**

```sh
cd certs && bash gen.sh
```

**2. Run with Docker Compose**

```sh
docker compose up --build
```

Or run each component manually:

```sh
go run server/server.go
go run worker/worker.go --target localhost:50051 --worker-id worker-1
go run client/client.go
```

## How It Works

1. The client submits a job (image, command, interval) to the server over gRPC.
2. The server stores the job in BadgerDB and dispatches it on schedule to an available worker.
3. The worker pulls the Docker image and executes the command, streaming results back.
4. All communication between components is secured with mutual TLS.

## Project Layout

```
server/   – Scheduler and gRPC server
worker/   – Worker and Docker client integration
client/   – Job submission CLI
proto/    – Protobuf service definitions
certs/    – TLS certificate generation scripts
```
