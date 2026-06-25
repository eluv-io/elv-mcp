# Eluvio MCP Server — Public Documentation

This directory contains the **public, user‑facing documentation** for the Eluvio MCP Server.  
It is intended for:

- Developers integrating the MCP server into LLM applications  
- Engineers who want to understand the system architecture  
- Contributors extending the server with new tasks  
- Users who need a clear reference for available tools  

The documentation is structured to be **modular, explicit, and easy to navigate**.

---

# Documentation Index

## 1. [`index.md`](index.md) — Landing Page
A high‑level introduction to the Eluvio MCP Server.

Covers:
- What the server does  
- Supported capabilities  
- How the documentation is organized  
- Where to start depending on your goal  

---

## 2. [`architecture.md`](architecture.md) — System Architecture
Explains how the server is built internally.

Includes:
- Task vs Worker model  
- Async task manager  
- MCP request flow  
- Fabric API integration  
- Registry and schema generation  

Useful for:
- Contributors  
- Engineers extending the server  
- Anyone wanting a deeper understanding of the system  

---

## 3. [`integration_guide.md`](integration_guide.md) — Integrating With LLMs
A practical guide for connecting the MCP server to:

- ChatGPT  
- Claude Desktop  
- LibreChat  
- Custom MCP clients  

Includes:
- Authentication setup  
- Example tool calls  
- Async task handling  
- File upload behavior  

This is the starting point for **application developers**.

---

## 4. [`tools_reference.md`](tools_reference.md) — Complete Tool Catalog
The authoritative reference for all MCP tools.

For each tool:
- Description  
- Required parameters  
- Optional parameters  
- When to use it  
- When *not* to use it  
- Example calls  

Covers:
- Fabric search tools  
- Tagger workflows  
- TagStore operations  
- Async task polling  

This is the primary document for **LLM agent developers**.

---

## 5. [`async_tasks.md`](async_tasks.md) — Async Task System
Explains how long‑running operations work.

Includes:
- Async lifecycle  
- Starting async tasks  
- Polling with `task_status`  
- Progress reporting  
- Cancellation  
- Best practices for LLMs  

Useful for:
- Anyone implementing workflows that involve tagging  
- Developers building interactive UIs or agents  

---

## 6. [`developing_tasks.md`](developing_tasks.md) — Contributor Guide
How to add new tasks to the MCP server.

Covers:
- Task vs Worker separation  
- Schema design  
- Writing LLM‑safe descriptions  
- Async support  
- Testing guidelines  
- Best practices  

This is the main reference for **contributors**.

---

## 7. [`authentication.md`](authentication.md) — OAuth2 & Security Model
Documents how authentication works.

Includes:
- Bearer token requirements  
- OAuth2 discovery  
- Token validation  
- Localhost protection  
- DNS rebinding safeguards  
- Error responses  

Essential for:
- Integrators  
- Security reviewers  
- Anyone deploying the server  

---

# How to Use This Documentation

### If you are integrating the MCP server:
Start with **[`integration_guide.md`](integration_guide.md)**, then refer to **[`tools_reference.md`](tools_reference.md)**.

### If you are building an LLM agent:
Use **[`tools_reference.md`](tools_reference.md)** and **[`async_tasks.md`](async_tasks.md)**.

### If you are extending the server:
Read **[`architecture.md`](architecture.md)** and **[`developing_tasks.md`](developing_tasks.md)**.

### If you are configuring authentication:
See **[`authentication.md`](authentication.md)**.

---

# Contributing

If you want to propose improvements, add new tools, or refine documentation:

1. Open an issue  
2. Submit a pull request  
3. Follow the conventions in `developing_tasks.md`  

---

# License

This documentation is part of the Eluvio MCP Server project.  
See the repository root for licensing information.
