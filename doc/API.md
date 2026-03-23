# API Reference

## HTTP Endpoints

| Endpoint | Method | Handler | Auth |
|----------|--------|---------|------|
| `/` | `POST` | `StreamableHTTPHandler` | Selective (see MCP methods below) |
| `/` | `GET` | `StreamableHTTPHandler` (SSE stream) | None |
| `/` | `DELETE` | `StreamableHTTPHandler` (session teardown) | None |
| `/.well-known/oauth-protected-resource` | `GET` | `auth.ProtectedResourceMetadataHandler` | None (public discovery) |

## MCP Methods (via `POST /`)

| JSON-RPC Method | Handler | Auth |
|-----------------|---------|------|
| `initialize` | MCP SDK built-in | None (open) |
| `notifications/initialized` | MCP SDK built-in | None (open) |
| `tools/list` | MCP SDK built-in (returns tool schemas) | OAuth Bearer required |
| `tools/call` -> `search_clips` | `SearchClips()` | OAuth Bearer required |
| `tools/call` -> `refresh_clips` | `RefreshToken()` | None (open) |

## Middleware Stack (on `/`)

```
loggingMiddleware -> recoverMiddleware -> selectiveAuthMiddleware -> StreamableHTTPHandler
```

The `selectiveAuthMiddleware` peeks at the JSON-RPC `method` (and `params.name`) fields in POST bodies.
The following pass through unauthenticated:
- `initialize`
- `notifications/initialized`
- `tools/call` with `params.name == "refresh_clips"`

All other methods require a valid `Authorization: Bearer <token>` header.

On 401 failure, the response includes:

```
WWW-Authenticate: Bearer resource_metadata=<RESOURCE_URL>/.well-known/oauth-protected-resource
```

This directs the client to the discovery endpoint to find the OAuth authorization server.

## Localhost Protection

When `ResourceURL` points to a non-localhost hostname (e.g. an ngrok URL), the SDK's DNS rebinding
protection is automatically disabled to allow proxied requests. When `ResourceURL` is `localhost`,
`127.0.0.1`, or `::1`, the protection remains active.
