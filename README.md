# k-mcp
MCP Server to interact with Kubernetes Cluster

> [!WARNING]  
> This is currently experimental. If you are seeking something production ready, please have a look at https://github.com/containers/kubernetes-mcp-server

## About

This project is inspired by and built upon the excellent work of [kubernetes-mcp-server](https://github.com/containers/kubernetes-mcp-server). We admire their approach and this repository reflects my personal vision based on their foundation.

### Key Differences

- **Streamable HTTP Only**: Unlike other MCP servers that support multiple transport methods, this server exclusively uses Streamable HTTP
- **Token-Based Authentication**: This server only accepts a token for authentication and does not support kubeconfig files
- **Simplified Approach**: Focuses on a streamlined experience with fewer configuration options
