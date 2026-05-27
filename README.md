# netcheck

netcheck is a command-line utility for quick network diagnostics. It allows you to check latency (ping), download speed, DNS resolution, and retrieve information about the current network environment.

The project is implemented using Clean Architecture principles, ensuring high testability, flexibility, and code maintainability.

## Features

- TCP Ping: Check latency to popular DNS servers (Google, Cloudflare, OpenDNS, Quad9). Calculates min/avg/max and jitter.
- DNS Resolution: Measure resolution time for popular domains.
- Speed Test: Measure download speed using Cloudflare CDN.
- Network Info: Display public IP, ISP details, local interfaces, and default gateway check.

## Architecture

The project is organized into layers:

1. Domain: The core of the system. Contains main data models (PingResult, IPInfo, etc.) and repository interfaces. It does not depend on other layers.
2. UseCase: Application business logic. Orchestrates network checks. Depends only on the Domain layer.
3. Infrastructure: Implementation of interfaces for interacting with the external world (network, HTTP, OS).
4. UI/CLI: Interface adapters. Handles user commands (Handler) and formats terminal output (Presenter).

## Installation

Ensure you have Go installed on your system.

```bash
git clone https://github.com/youruser/netcheck.git
cd netcheck
go build -o netcheck ./cmd/netcheck/main.go
```

## Usage

Run all checks:
```bash
./netcheck all
# or simply
./netcheck
```

Run specific modules:
```bash
./netcheck ping   # Latency only
./netcheck dns    # DNS only
./netcheck speed  # Speed test only
./netcheck info   # Network information only
```

Help:
```bash
./netcheck help
```

## Development

The folder structure follows standard Go conventions:
- cmd/netcheck: Entry point for building the binary.
- internal/: Internal logic, hidden from import by other projects.
- internal/domain: Entities and interfaces.
- internal/usecase: Interactors (business rules).
- internal/infrastructure: Network and HTTP logic.
- internal/ui: CLI-specific code (colors, presenter).

## License

MIT
