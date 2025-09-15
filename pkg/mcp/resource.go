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
	"github.com/modelcontextprotocol/go-sdk/mcp"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
)

func FindResource(resourceName string, discoveryClient discovery.CachedDiscoveryInterface, session *mcp.ServerSession) (v1.GroupVersionResource, error) {
	// TODO: Search in kinds with matching resource name, if exact match is not found, search for partial matches to ask for elicitation
	// APi version not needed.
}
