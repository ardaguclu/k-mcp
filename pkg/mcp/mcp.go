/*
Copyright 2025 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/auth"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/ardaguclu/k-mcp/pkg/version"
)

type Server struct {
	Port     string
	Audience string
}

func NewServer(port string, audience string) *Server {
	return &Server{
		Port:     port,
		Audience: audience,
	}
}

// JWTClaims represents the claims in our JWT tokens.
// In a real application, you would include additional claims like issuer, audience, etc.
type JWTClaims struct {
	Scopes []string `json:"scopes"`
	jwt.RegisteredClaims
}

func (s *Server) Run(ctx context.Context, dynamicConfig *DynamicConfig) error {
	mux := http.NewServeMux()

	verifyToken := func(ctx context.Context, tokenString string, _ *http.Request) (*auth.TokenInfo, error) {
		parser := jwt.NewParser()
		token, _, err := parser.ParseUnverified(tokenString, &JWTClaims{})
		if err != nil {
			return nil, fmt.Errorf("%w: failed to parse token: %v", auth.ErrInvalidToken, err)
		}

		if !token.Valid {
			return nil, fmt.Errorf("%w: invalid token", auth.ErrInvalidToken)
		}

		claims, ok := token.Claims.(*JWTClaims)
		if !ok {
			return nil, fmt.Errorf("%w: invalid token claims", auth.ErrInvalidToken)
		}

		if claims.ExpiresAt == nil {
			return nil, fmt.Errorf("%w: invalid token expired", auth.ErrInvalidToken)
		}

		if claims.ExpiresAt.Before(time.Now()) {
			return nil, fmt.Errorf("%w: token has expired", auth.ErrInvalidToken)
		}

		if claims.NotBefore != nil && claims.NotBefore.After(time.Now()) {
			return nil, fmt.Errorf("%w: token not yet valid", auth.ErrInvalidToken)
		}

		if claims.Audience == nil {
			return nil, fmt.Errorf("%w: invalid token audience", auth.ErrInvalidToken)
		}

		found := false
		apiServers := make([]string, 0)
		for _, aud := range claims.Audience {
			if aud == s.Audience {
				found = true
			} else {
				apiServers = append(apiServers, aud)
			}
		}
		if !found {
			return nil, fmt.Errorf("%w: token audience does not match %s", auth.ErrInvalidToken, s.Audience)
		}

		if len(apiServers) == 0 {
			return nil, fmt.Errorf("%w: apiserver url not found in audience %s", auth.ErrInvalidToken, s.Audience)
		}

		return &auth.TokenInfo{
			Scopes:     claims.Scopes,
			Expiration: claims.ExpiresAt.Time,
			Extra: map[string]any{
				"audience":     apiServers,
				"bearer_token": tokenString,
			},
		}, nil
	}

	loggingMiddleware := func(next mcp.MethodHandler) mcp.MethodHandler {
		return func(
			ctx context.Context,
			method string,
			req mcp.Request,
		) (mcp.Result, error) {
			slog.Debug("MCP method started",
				"method", method,
				"session_id", req.GetSession().ID(),
				"has_params", req.GetParams() != nil,
			)
			// Log more for tool calls.
			if ctr, ok := req.(*mcp.CallToolRequest); ok {
				slog.Debug("Calling tool",
					"name", ctr.Params.Name,
					"args", ctr.Params.Arguments)
			}

			start := time.Now()
			result, err := next(ctx, method, req)
			duration := time.Since(start)
			if err != nil {
				slog.Error("MCP method failed",
					"method", method,
					"session_id", req.GetSession().ID(),
					"duration_ms", duration.Milliseconds(),
					"err", err,
				)
			} else {
				slog.Debug("MCP method completed",
					"method", method,
					"session_id", req.GetSession().ID(),
					"duration_ms", duration.Milliseconds(),
					"has_result", result != nil,
				)
				// Log more for tool results.
				if ctr, ok := result.(*mcp.CallToolResult); ok {
					slog.Debug("tool result",
						"isError", ctr.IsError,
						"structuredContent", ctr.StructuredContent)
				}
			}
			return result, err
		}
	}

	server := mcp.NewServer(&mcp.Implementation{
		Name:    "k-mcp",
		Version: version.Get().Version,
	}, nil)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "resource_list",
		Description: "List Kubernetes resources of a specific type. This can be pods, deployments.v1.apps, etc. Kind.version.group format",
	}, func(_ context.Context, request *mcp.CallToolRequest, input ResourceListInput) (*mcp.CallToolResult, any, error) {
		apiServerUrls := request.Extra.TokenInfo.Extra["audience"].([]string)
		bearerToken := request.Extra.TokenInfo.Extra["bearer_token"].(string)
		var result []map[string]interface{}
		for _, u := range apiServerUrls {
			dynamicClient, discoveryClient, err := dynamicConfig.LoadRestConfig(bearerToken, u)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to load dynamic client: %w", err)
			}
			gvr, _, err := FindResource(input.Resource, discoveryClient, request.Session)
			if err != nil {
				return nil, nil, fmt.Errorf("given resource %s not found %w", input.Resource, err)
			}

			var resources *unstructured.UnstructuredList
			namespace := input.Namespace
			listOptions := v1.ListOptions{}
			if input.LabelSelector != "" {
				listOptions.LabelSelector = input.LabelSelector
			}

			if namespace != "" {
				resources, err = dynamicClient.Resource(gvr).Namespace(namespace).List(context.Background(), listOptions)
			} else {
				resources, err = dynamicClient.Resource(gvr).List(context.Background(), listOptions)
			}
			if err != nil {
				return nil, nil, fmt.Errorf("failed to list resources: %w", err)
			}

			for _, item := range resources.Items {
				result = append(result, item.Object)
			}
		}

		message := fmt.Sprintf("Found %d %s resources", len(result), input.Resource)
		if input.LabelSelector != "" {
			message += fmt.Sprintf(" with label selector '%s'", input.LabelSelector)
		}
		if input.Namespace != "" {
			message += fmt.Sprintf(" in namespace '%s'", input.Namespace)
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{
					Text: message,
				},
			},
		}, result, nil
	})
	mcp.AddTool(server, &mcp.Tool{
		Name:        "resource_get",
		Description: "Get detailed information about a specific Kubernetes resource. This can be pods, deployments.v1.apps, etc. Kind.version.group format",
	}, func(_ context.Context, request *mcp.CallToolRequest, input ResourceGetInput) (*mcp.CallToolResult, any, error) {
		apiServerUrls := request.Extra.TokenInfo.Extra["audience"].([]string)
		bearerToken := request.Extra.TokenInfo.Extra["bearer_token"].(string)
		var result []map[string]interface{}
		for _, u := range apiServerUrls {
			dynamicClient, discoveryClient, err := dynamicConfig.LoadRestConfig(bearerToken, u)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to load dynamic client: %w", err)
			}
			gvr, isNamespaced, err := FindResource(input.Resource, discoveryClient, request.Session)
			if err != nil {
				return nil, nil, fmt.Errorf("given resource %s not found %w", input.Resource, err)
			}

			if isNamespaced && input.Namespace == "" {
				defaultValue := json.RawMessage(`"default"`)
				elicitResult, err := request.Session.Elicit(context.Background(), &mcp.ElicitParams{
					Message: fmt.Sprintf("Namespace is required for namespaced resource %s. Please specify a namespace:", input.Resource),
					RequestedSchema: &jsonschema.Schema{
						Type: "object",
						Properties: map[string]*jsonschema.Schema{
							"namespace": {
								Type:        "string",
								Description: "The namespace for the resource",
								Default:     defaultValue,
							},
						},
						Required: []string{"namespace"},
					},
				})
				if err != nil {
					return nil, nil, fmt.Errorf("failed to elicit namespace: %w", err)
				}

				if elicitResult.Action != "accept" {
					return nil, nil, fmt.Errorf("user cancelled namespace selection")
				}

				namespace, ok := elicitResult.Content["namespace"].(string)
				if !ok || namespace == "" {
					namespace = "default"
				}
				input.Namespace = namespace
			}

			namespace := input.Namespace
			var resource *unstructured.Unstructured
			if namespace != "" {
				resource, err = dynamicClient.Resource(gvr).Namespace(namespace).Get(context.Background(), input.Name, v1.GetOptions{})
			} else {
				resource, err = dynamicClient.Resource(gvr).Get(context.Background(), input.Name, v1.GetOptions{})
			}
			if err != nil {
				return nil, nil, fmt.Errorf("failed to get resource: %w", err)
			}
			result = append(result, resource.Object)
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{
					Text: fmt.Sprintf("Retrieved %s/%s", input.Resource, input.Name),
				},
			},
		}, result, nil
	})
	server.AddReceivingMiddleware(loggingMiddleware)
	handler := mcp.NewStreamableHTTPHandler(func(req *http.Request) *mcp.Server {
		return server
	}, &mcp.StreamableHTTPOptions{
		Stateless: false,
	})
	handlerWithLogging := loggingHandler(handler)
	handlerWithJWT := auth.RequireBearerToken(verifyToken, nil)(handlerWithLogging)

	mux.Handle("/mcp", handlerWithJWT)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		//nolint:errcheck
		json.NewEncoder(w).Encode(map[string]string{
			"status": "healthy",
			"time":   time.Now().Format(time.RFC3339),
		})
	})

	httpServer := &http.Server{
		Addr:    ":" + s.Port,
		Handler: mux,
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGHUP, syscall.SIGTERM)

	serverErr := make(chan error, 1)
	go func() {
		slog.InfoContext(ctx, "Streaming streameable HTTP server", "port", s.Port)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	select {
	case sig := <-sigChan:
		slog.InfoContext(ctx, "received signal", "signal", sig)
		cancel()
	case <-ctx.Done():
		slog.InfoContext(ctx, "Context cancelled, initiating graceful shutdown")
	case err := <-serverErr:
		slog.ErrorContext(ctx, "Error from server", "error", err)
		return err
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	slog.InfoContext(shutdownCtx, "Shutting down HTTP server gracefully...")
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		slog.ErrorContext(shutdownCtx, "HTTP server shutdown error", "error", err)
		return err
	}

	slog.InfoContext(shutdownCtx, "HTTP server shutdown complete")
	return nil
}

type ResourceListInput struct {
	Resource      string `json:"resource" jsonschema:"required,description=The Kubernetes resource type (e.g. pods services deployments)"`
	Namespace     string `json:"namespace,omitempty" jsonschema:"description=The namespace to list resources from (optional defaults to all namespaces)"`
	LabelSelector string `json:"labelSelector,omitempty" jsonschema:"description=Label selector to filter resources (e.g. app=myapp,version=v1.0)"`
}

type ResourceGetInput struct {
	Resource  string `json:"resource" jsonschema:"required,description=The Kubernetes resource type (e.g. pod service deployment)"`
	Name      string `json:"name" jsonschema:"required,description=The name of the resource"`
	Namespace string `json:"namespace,omitempty" jsonschema:"description=The namespace of the resource (required for namespaced resources)"`
}
