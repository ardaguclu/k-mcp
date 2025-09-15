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
	"testing"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	cmdtesting "k8s.io/kubectl/pkg/cmd/testing"
)

func TestFindResource(t *testing.T) {
	tests := []struct {
		name           string
		resourceName   string
		setupDiscovery func() *cmdtesting.FakeCachedDiscoveryClient
		expectedGVR    schema.GroupVersionResource
		expectedError  string
	}{
		{
			name:         "exact match - pod",
			resourceName: "Pod",
			setupDiscovery: func() *cmdtesting.FakeCachedDiscoveryClient {
				dc := cmdtesting.NewFakeCachedDiscoveryClient()
				dc.PreferredResources = []*v1.APIResourceList{
					{
						GroupVersion: "v1",
						APIResources: []v1.APIResource{
							{Name: "pods", Kind: "Pod", Namespaced: true},
							{Name: "services", Kind: "Service", Namespaced: true},
							{Name: "nodes", Kind: "Node", Namespaced: false},
						},
					},
				}
				return dc
			},
			expectedGVR: schema.GroupVersionResource{
				Group:    "",
				Version:  "v1",
				Resource: "pods",
			},
		},
		{
			name:         "exact match - deployment with group",
			resourceName: "Deployment.apps",
			setupDiscovery: func() *cmdtesting.FakeCachedDiscoveryClient {
				dc := cmdtesting.NewFakeCachedDiscoveryClient()
				dc.PreferredResources = []*v1.APIResourceList{
					{
						GroupVersion: "apps/v1",
						APIResources: []v1.APIResource{
							{Name: "deployments", Kind: "Deployment", Namespaced: true},
							{Name: "replicasets", Kind: "ReplicaSet", Namespaced: true},
						},
					},
				}
				return dc
			},
			expectedGVR: schema.GroupVersionResource{
				Group:    "apps",
				Version:  "v1",
				Resource: "deployments",
			},
		},
		{
			name:         "exact match - deployment with version and group",
			resourceName: "Deployment.v1.apps",
			setupDiscovery: func() *cmdtesting.FakeCachedDiscoveryClient {
				dc := cmdtesting.NewFakeCachedDiscoveryClient()
				dc.PreferredResources = []*v1.APIResourceList{
					{
						GroupVersion: "apps/v1",
						APIResources: []v1.APIResource{
							{Name: "deployments", Kind: "Deployment", Namespaced: true},
							{Name: "replicasets", Kind: "ReplicaSet", Namespaced: true},
						},
					},
				}
				return dc
			},
			expectedGVR: schema.GroupVersionResource{
				Group:    "apps",
				Version:  "v1",
				Resource: "deployments",
			},
		},
		{
			name:         "resource by name - pods",
			resourceName: "pods",
			setupDiscovery: func() *cmdtesting.FakeCachedDiscoveryClient {
				dc := cmdtesting.NewFakeCachedDiscoveryClient()
				dc.PreferredResources = []*v1.APIResourceList{
					{
						GroupVersion: "v1",
						APIResources: []v1.APIResource{
							{Name: "pods", Kind: "Pod", Namespaced: true},
							{Name: "services", Kind: "Service", Namespaced: true},
						},
					},
				}
				return dc
			},
			expectedGVR: schema.GroupVersionResource{
				Group:    "",
				Version:  "v1",
				Resource: "pods",
			},
		},
		{
			name:         "resource not found",
			resourceName: "nonexistent",
			setupDiscovery: func() *cmdtesting.FakeCachedDiscoveryClient {
				dc := cmdtesting.NewFakeCachedDiscoveryClient()
				dc.PreferredResources = []*v1.APIResourceList{
					{
						GroupVersion: "v1",
						APIResources: []v1.APIResource{
							{Name: "pods", Kind: "Pod", Namespaced: true},
						},
					},
				}
				return dc
			},
			expectedError: "resource \"nonexistent\" not found",
		},
		{
			name:         "single partial match - auto select",
			resourceName: "node",
			setupDiscovery: func() *cmdtesting.FakeCachedDiscoveryClient {
				dc := cmdtesting.NewFakeCachedDiscoveryClient()
				dc.PreferredResources = []*v1.APIResourceList{
					{
						GroupVersion: "v1",
						APIResources: []v1.APIResource{
							{Name: "nodes", Kind: "Node", Namespaced: false},
							{Name: "pods", Kind: "Pod", Namespaced: true},
						},
					},
				}
				return dc
			},
			expectedGVR: schema.GroupVersionResource{
				Group:    "",
				Version:  "v1",
				Resource: "nodes",
			},
		},
		{
			name:         "multiple partial matches - nil session",
			resourceName: "po",
			setupDiscovery: func() *cmdtesting.FakeCachedDiscoveryClient {
				dc := cmdtesting.NewFakeCachedDiscoveryClient()
				dc.PreferredResources = []*v1.APIResourceList{
					{
						GroupVersion: "v1",
						APIResources: []v1.APIResource{
							{Name: "pods", Kind: "Pod", Namespaced: true},
							{Name: "podtemplates", Kind: "PodTemplate", Namespaced: true},
						},
					},
				}
				return dc
			},
			expectedError: "resource \"po\" not found, did you mean one of these: pods.v1., podtemplates.v1.",
		},
		{
			name:         "exact match - ingress with networking.k8s.io group",
			resourceName: "Ingress.networking.k8s.io",
			setupDiscovery: func() *cmdtesting.FakeCachedDiscoveryClient {
				dc := cmdtesting.NewFakeCachedDiscoveryClient()
				dc.PreferredResources = []*v1.APIResourceList{
					{
						GroupVersion: "networking.k8s.io/v1",
						APIResources: []v1.APIResource{
							{Name: "ingresses", Kind: "Ingress", Namespaced: true},
							{Name: "networkpolicies", Kind: "NetworkPolicy", Namespaced: true},
						},
					},
				}
				return dc
			},
			expectedGVR: schema.GroupVersionResource{
				Group:    "networking.k8s.io",
				Version:  "v1",
				Resource: "ingresses",
			},
		},
		{
			name:         "exact match - networkpolicy with version and group",
			resourceName: "NetworkPolicy.v1.networking.k8s.io",
			setupDiscovery: func() *cmdtesting.FakeCachedDiscoveryClient {
				dc := cmdtesting.NewFakeCachedDiscoveryClient()
				dc.PreferredResources = []*v1.APIResourceList{
					{
						GroupVersion: "networking.k8s.io/v1",
						APIResources: []v1.APIResource{
							{Name: "ingresses", Kind: "Ingress", Namespaced: true},
							{Name: "networkpolicies", Kind: "NetworkPolicy", Namespaced: true},
						},
					},
				}
				return dc
			},
			expectedGVR: schema.GroupVersionResource{
				Group:    "networking.k8s.io",
				Version:  "v1",
				Resource: "networkpolicies",
			},
		},
		{
			name:         "exact match - persistent volume claim",
			resourceName: "PersistentVolumeClaim",
			setupDiscovery: func() *cmdtesting.FakeCachedDiscoveryClient {
				dc := cmdtesting.NewFakeCachedDiscoveryClient()
				dc.PreferredResources = []*v1.APIResourceList{
					{
						GroupVersion: "v1",
						APIResources: []v1.APIResource{
							{Name: "persistentvolumeclaims", Kind: "PersistentVolumeClaim", Namespaced: true},
							{Name: "persistentvolumes", Kind: "PersistentVolume", Namespaced: false},
						},
					},
				}
				return dc
			},
			expectedGVR: schema.GroupVersionResource{
				Group:    "",
				Version:  "v1",
				Resource: "persistentvolumeclaims",
			},
		},
		{
			name:         "partial match - custom resource definition by resource name",
			resourceName: "customresource",
			setupDiscovery: func() *cmdtesting.FakeCachedDiscoveryClient {
				dc := cmdtesting.NewFakeCachedDiscoveryClient()
				dc.PreferredResources = []*v1.APIResourceList{
					{
						GroupVersion: "apiextensions.k8s.io/v1",
						APIResources: []v1.APIResource{
							{Name: "customresourcedefinitions", Kind: "CustomResourceDefinition", Namespaced: false},
						},
					},
					{
						GroupVersion: "v1",
						APIResources: []v1.APIResource{
							{Name: "pods", Kind: "Pod", Namespaced: true},
						},
					},
				}
				return dc
			},
			expectedGVR: schema.GroupVersionResource{
				Group:    "apiextensions.k8s.io",
				Version:  "v1",
				Resource: "customresourcedefinitions",
			},
		},
		{
			name:         "multiple complex partial matches - networking resources",
			resourceName: "net",
			setupDiscovery: func() *cmdtesting.FakeCachedDiscoveryClient {
				dc := cmdtesting.NewFakeCachedDiscoveryClient()
				dc.PreferredResources = []*v1.APIResourceList{
					{
						GroupVersion: "networking.k8s.io/v1",
						APIResources: []v1.APIResource{
							{Name: "networkpolicies", Kind: "NetworkPolicy", Namespaced: true},
							{Name: "networkattachmentdefinitions", Kind: "NetworkAttachmentDefinition", Namespaced: true},
						},
					},
				}
				return dc
			},
			expectedError: "resource \"net\" not found, did you mean one of these: networkpolicies.v1.networking.k8s.io, networkattachmentdefinitions.v1.networking.k8s.io",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			discoveryClient := tt.setupDiscovery()

			gvr, err := FindResource(tt.resourceName, discoveryClient, nil)

			if tt.expectedError != "" {
				if err == nil {
					t.Errorf("expected error containing %q, got nil", tt.expectedError)
					return
				}
				if err.Error() == "" || err.Error()[:len(tt.expectedError)] != tt.expectedError {
					t.Errorf("expected error containing %q, got %q", tt.expectedError, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if gvr != tt.expectedGVR {
				t.Errorf("expected GVR %+v, got %+v", tt.expectedGVR, gvr)
			}
		})
	}
}

func TestFindResource_ExactMatchPriority(t *testing.T) {
	// Test that exact matches are prioritized over partial matches with RBAC resources
	dc := cmdtesting.NewFakeCachedDiscoveryClient()
	dc.PreferredResources = []*v1.APIResourceList{
		{
			GroupVersion: "rbac.authorization.k8s.io/v1",
			APIResources: []v1.APIResource{
				{Name: "roles", Kind: "Role", Namespaced: true},
				{Name: "rolebindings", Kind: "RoleBinding", Namespaced: true},
				{Name: "clusterroles", Kind: "ClusterRole", Namespaced: false},
				{Name: "clusterrolebindings", Kind: "ClusterRoleBinding", Namespaced: false},
			},
		},
	}

	// Search for "Role.rbac.authorization.k8s.io" should return exact match "roles", not partial match with "RoleBinding"
	gvr, err := FindResource("Role.rbac.authorization.k8s.io", dc, nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
		return
	}

	expected := schema.GroupVersionResource{
		Group:    "rbac.authorization.k8s.io",
		Version:  "v1",
		Resource: "roles",
	}

	if gvr != expected {
		t.Errorf("expected exact match %+v, got %+v", expected, gvr)
	}
}

func TestFindResource_MultipleExactMatches(t *testing.T) {
	// Test that when multiple exact matches exist, the first one is returned
	dc := cmdtesting.NewFakeCachedDiscoveryClient()
	dc.PreferredResources = []*v1.APIResourceList{
		{
			GroupVersion: "v1",
			APIResources: []v1.APIResource{
				{Name: "pods", Kind: "Pod", Namespaced: true},
			},
		},
		{
			GroupVersion: "v2",
			APIResources: []v1.APIResource{
				{Name: "pods", Kind: "Pod", Namespaced: true},
			},
		},
	}

	gvr, err := FindResource("Pod", dc, nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
		return
	}

	expected := schema.GroupVersionResource{
		Group:    "",
		Version:  "v1",
		Resource: "pods",
	}

	if gvr != expected {
		t.Errorf("expected first match %+v, got %+v", expected, gvr)
	}
}
