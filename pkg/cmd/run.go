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

package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"

	"github.com/ardaguclu/k-mcp/pkg/mcp"
	"github.com/spf13/cobra"

	"k8s.io/cli-runtime/pkg/genericiooptions"
)

var (
	runExample = `
	# Run MCP Server with default values
	k-mcp run

	# Run MCP Server with custom values
	k-mcp run --port=8080 --log-level=debug

	# Run MCP Server with TLS configuration
	k-mcp run --certificate-authority=/path/to/ca.crt --tls-server-name=my-server
`
)

const (
	DefaultPort     = "8080"
	DefaultAudience = "k-mcp"
)

// RunOptions provides information required to run
// MCP Server
type RunOptions struct {
	Port                    string
	LogLevel                string
	Audience                string
	TLSInsecure             bool
	TLSCertificateAuthority string
	TLSServerName           string

	Server        *mcp.Server
	DynamicConfig *mcp.DynamicConfig

	genericiooptions.IOStreams
}

// NewRunOptions provides an instance of RunOptions with default values
func NewRunOptions(streams genericiooptions.IOStreams) *RunOptions {
	return &RunOptions{
		IOStreams: streams,
		Port:      DefaultPort,
		Audience:  DefaultAudience,
	}
}

// NewCmdRun provides a cobra command wrapping RunOptions
func NewCmdRun(streams genericiooptions.IOStreams) *cobra.Command {
	o := NewRunOptions(streams)

	cmd := &cobra.Command{
		Use:     "run [options]",
		Short:   "Start the MCP server",
		Long:    "Start the MCP server to provide Kubernetes access via Model Context Protocol",
		Example: runExample,
		RunE: func(c *cobra.Command, args []string) error {
			if err := o.Complete(c); err != nil {
				return err
			}
			if err := o.Validate(); err != nil {
				return err
			}
			if err := o.Run(); err != nil {
				return err
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&o.Port, "port", o.Port, "Start a streamable HTTP on the specified port. Default is 8080")
	cmd.Flags().StringVar(&o.LogLevel, "log-level", "info", "Log level (debug, info, warn, error)")
	cmd.Flags().StringVar(&o.Audience, "audience", o.Audience, "JWT token audience for validation. Default is k-mcp")
	cmd.Flags().BoolVar(&o.TLSInsecure, "insecure", false, "Skip TLS certificate verification when connecting to Kubernetes API server")
	cmd.Flags().StringVar(&o.TLSCertificateAuthority, "certificate-authority", "", "Path to a cert authority file for the certificate authority in TLS")
	cmd.Flags().StringVar(&o.TLSServerName, "tls-server-name", o.TLSServerName, "The name of the server to use for TLS")

	return cmd
}

// Complete sets all information required to run the MCP server
func (o *RunOptions) Complete(cmd *cobra.Command) error {
	_, err := strconv.Atoi(o.Port)
	if err != nil {
		return fmt.Errorf("invalid port number %s err: %w", o.Port, err)
	}

	var level slog.Level
	switch strings.ToLower(o.LogLevel) {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
	})
	logger := slog.New(handler)
	slog.SetDefault(logger)

	o.Server = mcp.NewServer(o.Port, o.Audience)

	if o.TLSCertificateAuthority != "" {
		_, err = os.ReadFile(o.TLSCertificateAuthority)
		if err != nil {
			return fmt.Errorf("failed to read CA certificate from %s: %w", o.TLSCertificateAuthority, err)
		}
	}

	if o.TLSInsecure {
		slog.Warn("Using insecure TLS client config. This is not recommended for production.")
	}

	o.DynamicConfig = mcp.NewDynamicConfig(o.TLSCertificateAuthority, o.TLSInsecure, o.TLSServerName)

	return nil
}

// Validate ensures that all required arguments and flag values are provided
func (o *RunOptions) Validate() error {
	validLevels := []string{"debug", "info", "warn", "error"}
	for _, valid := range validLevels {
		if strings.ToLower(o.LogLevel) == valid {
			return nil
		}
	}
	return fmt.Errorf("invalid log level %s, must be one of: %s", o.LogLevel, strings.Join(validLevels, ", "))
}

// Run runs the MCP Server
func (o *RunOptions) Run() error {
	ctx := context.Background()

	if err := o.Server.Run(ctx, o.DynamicConfig); err != nil {
		return err
	}
	return nil
}