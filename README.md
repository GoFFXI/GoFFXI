# FFXI Login Server

A modern Go implementation of a multi-role TCP server for Final Fantasy XI, featuring three distinct server roles (auth, data, view) that can be run from a single binary.

## Features

- **Single Binary, Multiple Roles**: One compiled binary that can function as auth, data, or view server based on runtime flags
- **TCP Connection Handling**: Efficient TCP server implementation with connection queuing and concurrent processing
- **Channel-Based Architecture**: Connections are processed in the order received using Go channels
- **Graceful Shutdown**: Proper signal handling for clean server shutdown
- **Structured Logging**: Built-in logger with multiple log levels and structured output

## Architecture

The server is designed with three distinct roles:

- **Auth Server** (port 54230): Handles authentication and login processes
- **Data Server** (port 54231): Manages game data and character information
- **View Server** (port 54001): Handles presentation/lobby and server selection

Each server:

- Accepts TCP connections on its designated port
- Queues connections in a channel for ordered processing
- Handles each connection in a separate goroutine
- Maintains connection state until client disconnects or timeout

## Installation

### Prerequisites

- Go 1.20 or higher
- Make (optional, for using Makefile commands)

### Building from Source

```bash
# Clone the repository
git clone https://github.com/GoFFXI/GoFFXI.git
cd GoFFXI

# Build the binary
go build -o bin/goffxi

# Or use make
make build
```

### Cross-Platform Builds

```bash
# Build for all platforms
make build-all

# Build for specific platforms
make build-linux
make build-darwin
make build-windows
```

## Usage

### Running a Single Server Role

```bash
# Run auth server
./bin/goffxi --role=auth

# Run data server
./bin/goffxi --role=data

# Run view server
./bin/goffxi --role=view
```

### Using Make Commands

```bash
# Run specific servers
make run-auth
make run-data
make run-view
```

### Running All Servers (Development)

Open three terminal windows and run each server:

```bash
# Terminal 1
./bin/goffxi --role=auth

# Terminal 2
./bin/goffxi --role=data

# Terminal 3
./bin/goffxi --role=view
```
