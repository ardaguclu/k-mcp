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
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
)

type DynamicConfig struct {
	CertificateAuthority string
	InsecureSkipVerify   bool
	TLSServerName        string
}

func NewDynamicConfig(certificateAuthority string, insecure bool, tlsServerName string) *DynamicConfig {
	return &DynamicConfig{
		CertificateAuthority: certificateAuthority,
		InsecureSkipVerify:   insecure,
		TLSServerName:        tlsServerName,
	}
}

func (d *DynamicConfig) LoadRestConfig(bearerToken, apiServerUrl string) (*dynamic.DynamicClient, error) {
	r := &rest.Config{
		Host:        apiServerUrl,
		BearerToken: bearerToken,
		Impersonate: rest.ImpersonationConfig{},
		TLSClientConfig: rest.TLSClientConfig{
			Insecure:   d.InsecureSkipVerify,
			ServerName: d.TLSServerName,
			CAFile:     d.CertificateAuthority,
		},
		UserAgent: "k-mcp",
	}
	dynamicClient, err := dynamic.NewForConfig(r)
	if err != nil {
		return nil, err
	}
	return dynamicClient, nil
}
