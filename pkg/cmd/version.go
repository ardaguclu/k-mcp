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
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"k8s.io/cli-runtime/pkg/genericiooptions"

	"github.com/ardaguclu/k-mcp/pkg/version"
)

type VersionOptions struct {
	Output string

	genericiooptions.IOStreams
}

func NewVersionOptions(streams genericiooptions.IOStreams) *VersionOptions {
	return &VersionOptions{
		IOStreams: streams,
	}
}

func NewCmdVersion(streams genericiooptions.IOStreams) *cobra.Command {
	o := NewVersionOptions(streams)

	cmd := &cobra.Command{
		Use:   "version",
		Short: "Print the version information",
		RunE: func(c *cobra.Command, args []string) error {
			return o.Run()
		},
	}

	cmd.Flags().StringVarP(&o.Output, "output", "o", "", "Output format. One of: (json)")

	return cmd
}

func (o *VersionOptions) Run() error {
	versionInfo := version.Get()

	switch o.Output {
	case "json":
		data, err := json.MarshalIndent(versionInfo, "", "  ")
		if err != nil {
			return err
		}
		fmt.Fprintln(o.Out, string(data))
	default:
		fmt.Fprintln(o.Out, versionInfo.String())
	}

	return nil
}