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
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/modelcontextprotocol/go-sdk/auth"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/ardaguclu/k-mcp/pkg/version"
)

type Server struct {
	Port string
}

func NewServer(port string) *Server {
	return &Server{
		Port: port,
	}
}

func (s *Server) Run(ctx context.Context) error {
	mux := http.NewServeMux()

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
	server.AddReceivingMiddleware(loggingMiddleware)
	handler := mcp.NewStreamableHTTPHandler(func(req *http.Request) *mcp.Server {
		return server
	}, &mcp.StreamableHTTPOptions{
		Stateless: false,
	})
	handlerWithLogging := loggingHandler(handler)
	handlerWithJWT := auth.RequireBearerToken(verifyJWT, nil)(handlerWithLogging)

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
