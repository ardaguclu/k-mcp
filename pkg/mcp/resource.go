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
	"fmt"
	"strconv"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
)

func FindResource(resourceName string, discoveryClient discovery.CachedDiscoveryInterface, session *mcp.ServerSession) (schema.GroupVersionResource, bool, error) {
	_, gk := schema.ParseKindArg(resourceName)

	resources, err := discoveryClient.ServerPreferredResources()
	if err != nil {
		return schema.GroupVersionResource{}, false, fmt.Errorf("failed to get server resources: %w", err)
	}

	type resourceMatch struct {
		gvr        schema.GroupVersionResource
		namespaced bool
	}

	var exactMatches []resourceMatch
	var partialMatches []resourceMatch

	for _, resourceList := range resources {
		gv, err := schema.ParseGroupVersion(resourceList.GroupVersion)
		if err != nil {
			continue
		}

		for _, resource := range resourceList.APIResources {
			currentMatch := resourceMatch{
				gvr: schema.GroupVersionResource{
					Group:    gv.Group,
					Version:  gv.Version,
					Resource: resource.Name,
				},
				namespaced: resource.Namespaced,
			}

			if resource.Kind == gk.Kind && gv.Group == gk.Group {
				exactMatches = append(exactMatches, currentMatch)
			}

			if strings.Contains(strings.ToLower(resource.Kind), strings.ToLower(gk.Kind)) ||
				strings.Contains(strings.ToLower(resource.Name), strings.ToLower(resourceName)) {
				partialMatches = append(partialMatches, currentMatch)
			}
		}
	}

	if len(exactMatches) == 1 {
		return exactMatches[0].gvr, exactMatches[0].namespaced, nil
	}

	if len(exactMatches) > 1 {
		return exactMatches[0].gvr, exactMatches[0].namespaced, nil
	}

	if len(partialMatches) == 0 {
		return schema.GroupVersionResource{}, false, fmt.Errorf("resource %q not found", resourceName)
	}

	if len(partialMatches) == 1 {
		return partialMatches[0].gvr, partialMatches[0].namespaced, nil
	}

	if session == nil {
		var options []string
		for _, match := range partialMatches {
			options = append(options, fmt.Sprintf("%s.%s.%s", match.gvr.Resource, match.gvr.Version, match.gvr.Group))
		}
		return schema.GroupVersionResource{}, false, fmt.Errorf("resource %q not found, did you mean one of these: %s", resourceName, strings.Join(options, ", "))
	}

	var options []string
	for i, match := range partialMatches {
		options = append(options, fmt.Sprintf("%d. %s.%s.%s", i+1, match.gvr.Resource, match.gvr.Version, match.gvr.Group))
	}

	optionsText := "Did you mean one of these?\n" + strings.Join(options, "\n")

	elicitResult, err := session.Elicit(context.Background(), &mcp.ElicitParams{
		Message: fmt.Sprintf("Resource '%s' not found. %s", resourceName, optionsText),
	})
	if err != nil {
		return schema.GroupVersionResource{}, false, fmt.Errorf("failed to elicit user choice: %w", err)
	}

	if elicitResult.Action != "accept" {
		return schema.GroupVersionResource{}, false, fmt.Errorf("user cancelled resource selection")
	}

	choiceStr, ok := elicitResult.Content["choice"].(string)
	if !ok {
		return schema.GroupVersionResource{}, false, fmt.Errorf("invalid choice format")
	}

	choice, err := strconv.Atoi(choiceStr)
	if err != nil || choice < 1 || choice > len(partialMatches) {
		return schema.GroupVersionResource{}, false, fmt.Errorf("invalid choice: %s", choiceStr)
	}

	return partialMatches[choice-1].gvr, partialMatches[choice-1].namespaced, nil
}
