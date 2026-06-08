# BBC MCP

MCP (Model Context Protocol) Go examples using `mark3labs/mcp-go`.

## Examples

### example1 — Full-featured MCP demo

- **Server** (`example/example1/server`): Exposes a `calculate` tool, a `docs://readme` resource, and a `greeting` prompt via SSE on `:9000`.
- **Client** (`example/example1/client`): Connects to the server and exercises all capabilities.

### example2 — Minimal echo server

- **Server** (`example/example2/server`): Exposes an `echo` tool via SSE on `:8080`.
- **Client** (`example/example2/client`): Connects and calls the echo tool.

## Running

```bash
# Start example1 server
go run ./example/example1/server/

# In another terminal, run example1 client
go run ./example/example1/client/
```
