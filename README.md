# k-mcp
MCP Server to interact with Kubernetes Cluster

## About

This project is inspired by and built upon the excellent work of [kubernetes-mcp-server](https://github.com/containers/kubernetes-mcp-server).
We admire their approach; however, this repository seeks to provide an alternative solution to eliminate certain pain points

### Key Differences

- **Streamable HTTP Only**: Unlike other MCP servers that support multiple transport methods, this server exclusively uses Streamable HTTP
- **Token-Based Authentication**: This server only accepts a token for authentication and does not support kubeconfig files. Consequently, the MCP server does not rely on any kubeconfig configuration
- **Multi-Cluster Single Tenant**: Supports multi-cluster environments with single tenant architecture, allowing access to multiple Kubernetes clusters through token-based authentication
- **No Delete Operations**: This server does not support DELETE operations for safety reasons. *Note: Some update operations may trigger Kubernetes garbage collection to prune related resources*
- **Simplified Approach**: Focuses on a streamlined experience with fewer configuration options

## Why This Repository Exists

### Motivation

Current MCP servers mirror kubectl's interactive patterns with a 1:1 mapping of function calls. For example, troubleshooting a failing application requires multiple separate calls: first list pods, then get logs for specific pods, then describe those pods for additional details. 
While this step-by-step approach works well for human operators using CLI tools, it creates inefficient back-and-forth communication when working with AI agents.

AI agents excel when they can gather comprehensive context in fewer operations. Instead of multiple granular calls, they benefit from intelligent tool functions that automatically collect related datasets and return complete information in a single response.

This repository takes a fundamentally different approach - designing an MCP server specifically optimized for AI workflows rather than simply replicating kubectl's human-centric interaction patterns.

### What We Don't Need
- **kubeconfig Risk**: Fully removing kubeconfig eliminates the risk of leveraged access. Since we don't store any kubeconfig files, the MCP server doesn't expose any risk of credential leakage or store sensitive data
- **CLI Tool Dependencies**: No reliance on kubectl or any other CLI tools. The server communicates directly with Kubernetes APIs, eliminating external command dependencies and improving reliability
- **STDIO Transport**: Standard input/output transport methods are not required for our HTTP-focused architecture
- **Server-Sent Events (SSE)**: SSE support is deprecated and adds unnecessary complexity
- **DELETE Operations**: Supporting delete operations introduces potential for unexpected issues and accidental resource removal

## Available Tools

This MCP server provides three core tools for interacting with Kubernetes clusters:

### resource_list
Lists Kubernetes resources of a specific type. Supports filtering by namespace and label selectors.
- **Parameters**: resource type (required), namespace (optional), label selector (optional)
- **Example**: List all pods in the default namespace with specific labels
- **Read-only operation** with no side effects

### resource_get
Retrieves detailed information about a specific Kubernetes resource.
- **Parameters**: resource type (required), resource name (required), namespace (optional for namespaced resources)
- **Example**: Get detailed information about a specific deployment
- **Read-only operation** with no side effects

### resource_apply
Applies Kubernetes resources using server-side apply. Supports both single resources and multiple resources separated by `---`.
- **Parameters**: resource YAML (required)
- **Features**: Dry-run validation, user confirmation prompts, multi-document YAML support
- **Destructive operation** that can modify cluster state

All tools support multiple API servers through JWT token-based authentication and provide comprehensive error handling with user-friendly messages. The server uses Kubernetes discovery APIs to dynamically access all live resources in the cluster, eliminating the need for pre-configured resource definitions.

## Security Restrictions

To improve security posture, this MCP server opinionatedly restricts access to certain sensitive Kubernetes resources:

### Restricted Resources
- **Secrets** (`secrets.v1`): Contains sensitive data like passwords, tokens, and certificates
- **Service Accounts** (`serviceaccounts.v1`): Manages authentication tokens and cluster access credentials
- **All RBAC Resources** (`*.rbac.authorization.k8s.io`): Includes roles, rolebindings, clusterroles, and clusterrolebindings that control cluster permissions

These resources are completely filtered out during discovery and will not appear in resource listings or be accessible through any MCP tools. Attempts to access them will result in "resource not found" errors.

## Setup and Usage

### Prerequisites

- [Kind](https://kind.sigs.k8s.io/docs/user/quick-start/#installation) for local Kubernetes development
- `kubectl` CLI tool

### Step-by-Step Setup

#### 1. Create a Kind Cluster

```bash
kind create cluster --name k-mcp-demo
kubectl cluster-info --context kind-k-mcp-demo
```

#### 2. Export Cluster Certificate Authority

Kind clusters generate their own certificate authority. Extract it to a file for the MCP server:

```bash
kubectl config view --raw -o jsonpath='{.clusters[0].cluster.certificate-authority-data}' | base64 -d > ca.cert
```

#### 3. Create a Service Account

```bash
kubectl create serviceaccount k-mcp-sa
```

#### 4. Grant Necessary Permissions

Create appropriate RBAC permissions for the service account:

```bash
# Create ClusterRole with necessary permissions
kubectl create clusterrole k-mcp-role \
  --verb=get,list,watch,create,update,patch,apply \
  --resource=*

# Bind the ClusterRole to the service account
kubectl create clusterrolebinding k-mcp-binding \
  --clusterrole=k-mcp-role \
  --serviceaccount=default:k-mcp-sa
```

**Note**: Adjust the permissions based on your security requirements. For production use, consider using more restrictive permissions.

#### 5. Generate Service Account Token

The k-mcp server requires JWT tokens with specific audience configuration. The token **must** include three audiences in this order:
1. **First audience**: Your Kubernetes API server URL
2. **Second audience**: The MCP server audience (default: "k-mcp", or custom value set via `--audience` flag)
3. **Third audience**: The default audience (can be checked via `kubectl create token default`)

##### Get Your API Server URL

```bash
kubectl config view --minify -o jsonpath='{.clusters[0].cluster.server}'
```

##### Check Default Audience

```bash
kubectl create token default
```

This will show you the default audience format used by your cluster.

##### Create Token with Correct Audiences

```bash
# Replace <API_SERVER_URL> with your actual API server URL
# Replace <MCP_AUDIENCE> with your MCP server audience (default: "k-mcp")
# Replace <DEFAULT_AUDIENCE> with your cluster's default audience

kubectl create token k-mcp-sa \
  --audience=<API_SERVER_URL> \
  --audience=<MCP_AUDIENCE> \
  --audience=<DEFAULT_AUDIENCE> \
  --duration=24h
```

**Examples**:

```bash
kubectl create token k-mcp-sa \
  --audience=https://127.0.0.1:6443 \
  --audience=k-mcp \
  --audience=default \
  --duration=24h

#### 6. Start the MCP Server

```bash
# Using default settings (port 8080, audience "k-mcp") with CA certificate
./k-mcp --certificate-authority ca.cert

# Using custom settings with CA certificate
./k-mcp --port 9090 --audience my-custom-mcp --certificate-authority ca.cert
```

#### 7. Configure Your MCP Client

Use the generated token to authenticate with the MCP server. Configure your MCP client (such as Claude Desktop) by adding the server configuration to your `mcp.json` file:

```json
{
  "mcpServers": {
    "k-mcp": {
      "url": "http://localhost:8080/mcp",
      "headers": {
        "Authorization": "Bearer <YOUR_SERVICE_ACCOUNT_TOKEN>"
      }
    }
  }
}
```

Replace `<YOUR_SERVICE_ACCOUNT_TOKEN>` with the token generated in step 5.

**Important**:
- If you specified a custom audience via the `--audience` flag when starting the server, ensure your token includes that exact audience as the second audience parameter.
- This MCP server supports multiple Kubernetes clusters as long as the JWT issuer is the same and the audiences are correctly aligned in the token.

### Token Requirements Summary

- **Expiration**: Token must not be expired
- **Not Before**: Token must be currently valid (if nbf claim is present)
- **Audiences**: Must contain exactly three audiences:
  1. Kubernetes API server URL (used for cluster communication)
  2. MCP server audience (used for server authentication)
  3. Default audience (cluster default)

### Security Considerations

- Service account tokens have limited lifetime - regenerate as needed
- Use least privilege principle when assigning RBAC permissions
- Consider using namespace-scoped roles instead of cluster roles when possible
- Regularly rotate service account tokens
- Store tokens securely and avoid logging them

---

*This README was mostly generated with generative AI assistance.*
