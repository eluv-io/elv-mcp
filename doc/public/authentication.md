# Authentication  
OAuth2 Authorization Code Flow (No PKCE) for the Eluvio MCP Server

The Eluvio MCP Server uses **OAuth2 Authorization Code Flow (no PKCE)** for authenticating all incoming MCP requests.  
The server does **not** perform OAuth2 token exchange.  
Instead:

- The **MCP client** performs the OAuth2 login and token exchange  
- The **MCP server** validates the resulting access token  
- The server maps the OAuth `sub` claim to a tenant defined in `config.yaml`  

This document describes the authentication model, token validation, tenant mapping, and security protections.

---

# 1. Overview

The MCP server exposes an HTTP endpoint at:

```
http://<host>:<port>/
```


The `<port>` is defined in `config.yaml`:

``` json
server:
  port: 8181
  oauth_issuer: https://auth.example.com
  resource_url: https://mcp.example.com
```

### Key points:

- The **MCP client** performs OAuth2 Authorization Code Flow  
- The **MCP server** validates tokens using the configured issuer  
- The server extracts the `sub` claim and maps it to a tenant  
- Each tenant has its own Fabric credentials  

---

# 2. OAuth2 Responsibilities

## 2.1 What the MCP Client Does

Every MCP client must:

1. Redirect the user to the OAuth2 provider  
2. Receive the authorization code at its own redirect URI  
3. Exchange the code for tokens  
4. Store and refresh tokens  
5. Send authenticated MCP requests with:

```
Authorization: Bearer <access_token>
```

The MCP server does **not** assist with these steps.

---

## 2.2 What the MCP Server Does

The server:

- Validates the access token  
- Verifies the issuer matches `oauth_issuer`  
- Fetches JWKS keys from the issuer  
- Verifies token signature  
- Verifies expiration  
- Extracts the `sub` claim  
- Maps `sub` → tenant  
- Uses the tenant’s Fabric credentials  
- Executes the requested MCP tool  

The server does **not**:

- Perform OAuth2 token exchange  
- Store client secrets  
- Handle redirect URIs  
- Issue tokens  
- Refresh tokens  

---

# 3. Token Validation

Token validation is performed using the configured issuer:

``` json
server:
  oauth_issuer: https://auth.example.com
```

The server performs:

### ✔ Signature verification  
Using JWKS keys from the issuer.

### ✔ Issuer validation  
The token’s `iss` claim must match `oauth_issuer`.

### ✔ Expiration validation  
Expired tokens are rejected.

### ✔ Audience/resource validation  
If the token includes an `aud` claim, it must match `resource_url`.

### ✔ Subject extraction  
The `sub` claim identifies the authenticated user.

---

# 4. Tenant Mapping

After validating the token, the server extracts the `sub` claim and maps it to a tenant defined in `config.yaml`:

``` json
tenants:
  - id: example
    users:
      - "00000000-0000-0000-0000-000000000000"
    fabric:
      private_key: "XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX"
      qlibid_index: "ilibXXXXXXXXXXXXXXXXXXXXXXXXXXXX"
      qid_index: "iq__XXXXXXXXXXXXXXXXXXXXXXXXXXXX"
```

### Mapping rules:

- Each user’s `sub` must appear exactly once  
- A user belongs to exactly one tenant  
- The tenant determines which Fabric credentials the server uses  

If the `sub` is not found:

- The request is rejected with an authentication error  

---

# 5. Fabric Credential Handling

Each tenant provides:

- A private key  
- QLib index ID  
- Q index ID  
- Optional search collection ID  

These are loaded at startup and stored in memory.

The server:

- Signs Fabric requests using the tenant’s private key  
- Caches Fabric tokens per tenant  
- Protects cached tokens with a mutex  

This is internal to the server and transparent to MCP clients.

---

# 6. Error Responses

### Missing Authorization Header
``` json
{
  "error": "unauthorized",
  "message": "Missing Authorization header"
}
```

### Invalid Token
``` json
{
  "error": "unauthorized",
  "message": "Invalid or expired access token"
}
```

### Unknown Tenant
``` json
{
  "error": "forbidden",
  "message": "User is not associated with any configured tenant"
}
```

### Invalid Issuer
``` json
{
  "error": "unauthorized",
  "message": "Token issuer does not match server.oauth_issuer"
}
```

---

# 7. Security Protections

### Localhost & Interface Restrictions
The server is intended to run locally or behind a controlled network boundary.

### DNS Rebinding Protection
The server validates host headers and interface bindings to prevent rebinding attacks.

### No Token Storage
The server does not store or persist access tokens.

### No OAuth2 Secrets
The server does not require or store:

- client_id  
- client_secret  
- redirect_uri  

These belong exclusively to the MCP client.

---

# 8. Summary

- MCP clients perform OAuth2 Authorization Code Flow  
- The MCP server validates tokens using `oauth_issuer`  
- The server extracts `sub` and maps it to a tenant  
- Tenants define Fabric credentials  
- The server listens on the port defined in `config.yaml`  
- The server does not perform token exchange or refresh  
- All authentication is stateless and based on bearer tokens  

