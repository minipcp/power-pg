# power-pg
Power PG is a middleware for PostgreSQL written in Go. Primarily, it proxies PostgreSQL requests, but it allows you to intercept and replace statements then forward the answer back to the client.

The proxy part is a simplified copy from https://github.com/lumanetworks/go-tcp-proxy project.
