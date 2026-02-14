# Epoch

Epoch is a simple, distributed job scheduler built with Go and gRPC. It manages executing tasks inside Docker containers across connected worker nodes.

## Components

- **Server**: Manages job submissions, scheduling, and worker connections.
- **Worker**: Connects to the server and executes jobs using Docker.
- **Client**: Submits jobs to the server.

## Prerequisites

- Go 1.24+
- Docker running on worker nodes

## Getting Started

1. **Start the Server**

   ```sh
   go run server/server.go
   ```

2. **Start a Worker**
   Ensure Docker is running, then:

   ```sh
   go run worker/worker.go
   ```

3. **Submit a Job**
   ```sh
   go run client/client.go
   ```

## How It Works

- The **Server** listens for requests from the client and manages an in-memory job queue.
- **Workers** connect to the server via a streaming gRPC connection to receive jobs.
- **Jobs** are defined with a command, schedule (interval in seconds), and a Docker image.
- When a job is scheduled, the server dispatches it to an available worker.
- The worker pulls the specified Docker image and executes the command.

## Project Structure

- `server/`: Scheduler logic and gRPC server implementation.
- `worker/`: Worker logic and Docker client integration.
- `client/`: Client for job submission.
- `proto/`: Protocol Buffer definitions for the gRPC service.
