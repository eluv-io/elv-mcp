# Integration Guide  
Connecting OAuth2‑Enabled MCP Clients to the Eluvio MCP Server

This guide explains how to integrate LLM applications with the **Eluvio MCP Server** using **OAuth2 Authorization Code Flow (no PKCE)**.  
The MCP server does **not** perform OAuth2 token exchange.  
Instead:

- The **MCP client** performs the OAuth2 login and token exchange  
- The **MCP server** validates tokens using the configured OAuth issuer  
- The server maps the OAuth `sub` claim to a tenant defined in `config.yaml`  

This document describes how to configure MCP clients, how authentication works, and how to connect to the server.

---

# 1. Overview

The Eluvio MCP Server exposes a Model Context Protocol endpoint at:
```
http://<host>:<port>/
```

The `<port>` is defined in `config.yaml`:

``` yaml
server:
  port: 8181
```

All MCP clients must authenticate using **OAuth2 Authorization Code Flow**.  
The server validates tokens using the configured issuer:

``` yaml
server:
  oauth_issuer: https://auth.example.com
  resource_url: https://mcp.example.com
```

The server does **not** require:

- client_id  
- client_secret  
- redirect_uri  

These belong to the **MCP client**, not the server.

---

# 2. Server Configuration Summary

The server is configured entirely through `config.yaml`.  
Relevant fields for integration:

``` yaml
server:
  port: 8181
  oauth_issuer: https://auth.example.com
  resource_url: https://mcp.example.com
```

### What these fields mean

- **port** — The TCP port the MCP server listens on  
- **oauth_issuer** — The OAuth2 provider that issues tokens (e.g., Ory)  
- **resource_url** — The public URL representing this MCP server as a protected resource  

The server uses `oauth_issuer` to:

- Fetch JWKS keys  
- Validate signatures  
- Validate issuer claim  
- Extract the `sub` claim  

The `sub` claim is mapped to a tenant in:

``` yaml
tenants:
  - id: example
    users:
      - "00000000-0000-0000-0000-000000000000"
```

---

# 3. OAuth2 Authentication Model

## 3.1 What the MCP Client Must Do

Every MCP client must:

1. Redirect the user to the OAuth2 provider  
2. Receive the authorization code at its own redirect URI  
3. Exchange the code for tokens  
4. Store and refresh tokens  
5. Send authenticated MCP requests with:

```
Authorization: Bearer <access_token>
```

## 3.2 What the MCP Server Does

The server:

- Validates the access token  
- Extracts the `sub` claim  
- Maps `sub` → tenant  
- Uses the tenant’s Fabric credentials  
- Executes the requested MCP tool  

The server does **not**:

- Perform OAuth2 token exchange  
- Store client secrets  
- Handle redirect URIs  
- Issue tokens  

---

# 4. Platform‑Specific MCP Client Configuration

Each MCP client uses its own configuration format.  
Below are correct examples using **OAuth2 Authorization Code Flow**.

---

# 4.1 LibreChat (YAML)

LibreChat handles OAuth2 login and token exchange internally.

``` yaml
mcpServers:
  eluvio:
    title: "Eluvio MCP"
    description: "Eluvio MCP Server"
    type: streamable-http
    url: "https://mcp.example.com"
    timeout: 120000
    initTimeout: 15000
    startup: false
    serverInstructions: true
    oauth:
      authorization_url: "https://auth.example.com/oauth2/auth"
      token_url: "https://auth.example.com/oauth2/token"
      redirect_uri: "http://localhost:3080/api/mcp/eluvio/oauth/callback"
      scope: "openid offline_access"
      client_id: "00000000-0000-0000-0000-000000000000"
      client_secret: "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
```

LibreChat responsibilities:

- Opens browser for login  
- Handles redirect  
- Exchanges code for tokens  
- Refreshes tokens  
- Sends authenticated MCP requests  

---

# 4.2 ChatGPT (OpenAI MCP)

ChatGPT uses a JSON‑like configuration block.

```
{
  "url": "https://mcp.example.com",
  "oauth": {
    "authorization_url": "https://auth.example.com/oauth2/auth",
    "token_url": "https://auth.example.com/oauth2/token",
    "client_id": "00000000-0000-0000-0000-000000000000",
    "client_secret": "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
    "redirect_uri": "https://chat.openai.com/mcp/oauth/callback",
    "scope": "openid offline_access"
  }
}
```

ChatGPT responsibilities:

- Performs OAuth2 login  
- Stores tokens  
- Refreshes tokens  
- Sends authenticated MCP requests  

---

# 4.3 Claude Desktop

Claude Desktop uses a JSON config file:

`~/Library/Application Support/Claude/claude_desktop_config.json`

``` json
{
  "mcpServers": {
    "eluvio": {
      "url": "https://mcp.example.com",
      "type": "streamable-http",
      "oauth": {
        "authorization_url": "https://auth.example.com/oauth2/auth",
        "token_url": "https://auth.example.com/oauth2/token",
        "client_id": "00000000-0000-0000-0000-000000000000",
        "client_secret": "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
        "redirect_uri": "claude://mcp/eluvio/oauth/callback",
        "scope": "openid offline_access"
      }
    }
  }
}
```

Claude responsibilities:

- Opens browser for login  
- Handles custom URI redirect (`claude://`)  
- Exchanges code for tokens  
- Refreshes tokens  

---

# 4.4 Custom MCP Clients

Custom clients must implement:

### Step 1 — Redirect user to authorization URL

```
GET https://auth.example.com/oauth2/auth
  ?client_id=...
  &redirect_uri=...
  &response_type=code
  &scope=openid offline_access
```

### Step 2 — Receive authorization code at redirect URI

Your application must host a callback endpoint.

### Step 3 — Exchange code for tokens

```
POST https://auth.example.com/oauth2/token
  grant_type=authorization_code
  code=<auth_code>
  redirect_uri=<redirect_uri>
  client_id=<client_id>
  client_secret=<client_secret>
```

### Step 4 — Call MCP server with access token

```
Authorization: Bearer <access_token>
```

### Step 5 — Refresh tokens

```
grant_type=refresh_token
refresh_token=<refresh_token>
```

---

# 5. Calling Tools

Once authenticated, tool calls follow the MCP protocol.

Example:

``` json
{
  "method": "tools/call",
  "params": {
    "name": "search_clips",
    "arguments": {
      "query": "sunset beach",
      "limit": 5
    }
  }
}
```

---

# 6. Troubleshooting

### Invalid token  
Check that the MCP client is performing OAuth2 correctly.

### Unknown tenant  
Ensure the user’s `sub` claim appears in `config.yaml` under `tenants:`.

### Wrong redirect URI  
OAuth2 providers require exact string matches.

### Token exchange failure  
Verify:
- authorization_url  
- token_url  
- client_id  
- client_secret  

---

# Summary

- MCP clients perform OAuth2 Authorization Code Flow  
- The MCP server validates tokens using `oauth_issuer`  
- The server maps `sub` → tenant  
- The server listens on the port defined in `config.yaml`  
- Each platform has its own OAuth2 configuration format  
- Once authenticated, clients can call any MCP tool  

