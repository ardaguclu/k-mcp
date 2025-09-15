# k-mcp
MCP Server to interact with Kubernetes Cluster

> [!WARNING]  
> This is currently experimental. If you are seeking something production ready, please have a look at https://github.com/containers/kubernetes-mcp-server

## About

This project is inspired by and built upon the excellent work of [kubernetes-mcp-server](https://github.com/containers/kubernetes-mcp-server). We admire their approach and this repository reflects my personal vision based on their foundation.

### Key Differences

- **Streamable HTTP Only**: Unlike other MCP servers that support multiple transport methods, this server exclusively uses Streamable HTTP
- **Token-Based Authentication**: This server only accepts a token for authentication and does not support kubeconfig files. Consequently, the MCP server does not rely on any kubeconfig configuration
- **Multi-Cluster Single Tenant**: Supports multi-cluster environments with single tenant architecture, allowing access to multiple Kubernetes clusters through token-based authentication
- **No Delete Operations**: This server does not support DELETE operations for safety reasons. *Note: Some update operations may trigger Kubernetes garbage collection to prune related resources*
- **Simplified Approach**: Focuses on a streamlined experience with fewer configuration options

## Why This Repository Exists

### Motivation

Traditional MCP servers mirror kubectl's interactive patterns with a 1:1 mapping of function calls. For example, troubleshooting a failing application requires multiple separate calls: first list pods, then get logs for specific pods, then describe those pods for additional details. 
While this step-by-step approach works well for human operators using CLI tools, it creates inefficient back-and-forth communication when working with AI agents.

AI agents excel when they can gather comprehensive context in fewer operations. Instead of multiple granular calls, they benefit from intelligent tool functions that automatically collect related datasets and return complete information in a single response.

This repository takes a fundamentally different approach - designing an MCP server specifically optimized for AI workflows rather than simply replicating kubectl's human-centric interaction patterns.

### What We Don't Need
- **kubeconfig Risk**: Fully removing kubeconfig eliminates the risk of leveraged access. Since we don't store any kubeconfig files, the MCP server doesn't expose any risk of credential leakage or store sensitive data
- **CLI Tool Dependencies**: No reliance on kubectl or any other CLI tools. The server communicates directly with Kubernetes APIs, eliminating external command dependencies and improving reliability
- **STDIO Transport**: Standard input/output transport methods are not required for our HTTP-focused architecture
- **Server-Sent Events (SSE)**: SSE support is deprecated and adds unnecessary complexity
- **DELETE Operations**: Supporting delete operations introduces potential for unexpected issues and accidental resource removal

### What We Gain
- **Zero Credential Storage**: No kubeconfig means zero risk of credential exposure and no sensitive data storage
- **JWT Token Simplicity**: Fully relying on JWT tokens provides multi-cluster support for free without complex authentication management
- **AI-Optimized Tool Calls**: Provides a few comprehensive tool calls that automatically collect sets of related data and return it in one operation, preventing back-and-forth communication ideal for generative AI workflows
- **HTTP Focus**: Streamable HTTP-only approach simplifies deployment and client integration
- **Safety First**: No delete operations prevent accidental resource removal
- **Multi-Cluster Ready**: Token-based authentication naturally supports multiple Kubernetes clusters
- **Reduced Attack Surface**: Fewer features and no credential storage mean significantly fewer potential security vulnerabilities

---

*This README was mostly generated with generative AI assistance.*
