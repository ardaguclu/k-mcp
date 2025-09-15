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
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"k8s.io/cli-runtime/pkg/genericiooptions"
)

var (
	kmcpExample = `
	# Show the help
	k-mcp -h

	# Run MCP Server with default values
	k-mcp

	# Run MCP Server with custom values
	k-mcp --port=8080
`
)

const DefaultPort = "8080"

// KMCPOptions provides information required to run
// MCP Server
type KMCPOptions struct {
	Port string

	genericiooptions.IOStreams
}

// NewKMCPOptions provides an instance of KMCPOptions with default values
func NewKMCPOptions(streams genericiooptions.IOStreams) *KMCPOptions {
	return &KMCPOptions{
		IOStreams: streams,
		Port:      DefaultPort,
	}
}

// NewCmdKMCP provides a cobra command wrapping KMCPOptions
func NewCmdKMCP(streams genericiooptions.IOStreams) *cobra.Command {
	o := NewKMCPOptions(streams)

	cmd := &cobra.Command{
		Use:     "k-mcp [options]",
		Short:   "MCP Server to interact with Kubernetes Cluster",
		Example: kmcpExample,
		Annotations: map[string]string{
			cobra.CommandDisplayNameAnnotation: "k-mcp",
		},
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

	cmd.AddCommand(NewCmdVersion(streams))

	return cmd
}

// Complete sets all information required to run the MCP server
func (o *KMCPOptions) Complete(cmd *cobra.Command) error {
	return nil
}

// Validate ensures that all required arguments and flag values are provided
func (o *KMCPOptions) Validate() error {
	_, err := strconv.Atoi(o.Port)
	if err != nil {
		return fmt.Errorf("invalid port number %s err: %w", o.Port, err)
	}

	return nil
}

// Run runs the MCP Server
func (o *KMCPOptions) Run() error {
	return nil
}
