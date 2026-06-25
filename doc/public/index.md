# Eluvio MCP Server — Public Documentation

Welcome to the public documentation for the **Eluvio MCP Server**, a Model Context Protocol (MCP) service that exposes Eluvio Fabric search, tagging, and tag‑store operations to LLM applications.

This documentation is designed for:

- Developers integrating the MCP server into ChatGPT, Claude, LibreChat, or custom LLM agents  
- Engineers who want to understand the server architecture  
- Contributors who want to add new tasks or extend functionality  
- Users who need a clear reference for all available tools  

---

## What This Server Provides

The Eluvio MCP Server exposes a set of **structured, safe, deterministic tools** that allow LLMs to:

### Search Fabric
- Search for video clips  
- Search for images (text or image‑based)  
- Refresh expiring clip URLs  

### Run Tagger Workflows
- Run multi‑model tagging (`tag_content`)  
- Run high‑level workflows (`tag_chapters`, `tag_characters`)  
- Inspect tagging status  
- Stop tagging jobs  
- List available Tagger models  

### Manage TagStore Tracks
- Create TagStore tracks  
- Delete TagStore tracks  

### Use Asynchronous Tasks
- Start long‑running operations  
- Poll for completion via `task_status`  

All tools follow strict rules to ensure predictable behavior when used by LLMs.

---

## Documentation Structure

This public documentation is organized into the following files:

### 1. **architecture.md**
A high‑level overview of the MCP server architecture, including:
- Task registry  
- Worker model  
- Async task manager  
- HTTP and MCP flow  

### 2. **integration_guide.md**
How to integrate this MCP server with:
- ChatGPT  
- Claude Desktop  
- LibreChat  
- Custom MCP clients  

Includes authentication, connection setup, and example tool calls.

### 3. **tools_reference.md**
A complete catalog of all available MCP tools:
- Parameters  
- Behavior  
- When to use / not use  
- Example calls  

This is the primary reference for LLM developers.

### 4. **async_tasks.md**
Explains the asynchronous task system:
- Task lifecycle  
- How to start async work  
- How to poll with `task_status`  
- Result formats  

### 5. **developing_tasks.md**
A contributor guide for adding new tasks:
- Task vs worker separation  
- Input schema design  
- File upload support  
- Async task integration  
- Testing guidelines  

### 6. **authentication.md**
Describes:
- OAuth2 Bearer token requirements  
- Discovery endpoint  
- Localhost protection  
- DNS rebinding safeguards  

---

## Who This Documentation Is For

This documentation is intended for:

- **LLM application developers**  
  Integrating the MCP server into agents, assistants, or chat interfaces.

- **Backend engineers**  
  Extending the server with new tasks or workers.

- **Contributors**  
  Maintaining or improving the server.

- **Technical users**  
  Who need a clear understanding of what each tool does.

---

## How to Use This Documentation

If you are:

###  Integrating the MCP server  
Start with **integration_guide.md**.

### Calling tools from an LLM  
Use **tools_reference.md**.

### Understanding the system  
Read **architecture.md**.

### Adding new tasks  
Go to **developing_tasks.md**.

### Working with async operations  
See **async_tasks.md**.

### Configuring authentication  
Check **authentication.md**.

---

## Feedback & Contributions

If you want to propose improvements, add new tools, or refine the documentation, please open an issue or submit a pull request in the repository.

---

This concludes the public landing page.  
Proceed to the next file when ready.
