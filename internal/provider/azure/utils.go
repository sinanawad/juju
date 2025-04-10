// Copyright 2015 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package azure

import (
	"context"

	azurecloud "github.com/Azure/azure-sdk-for-go/sdk/azcore/cloud"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"
)

// collectAPIVersions returns a map of the latest API version for each
// possible resource type. This is needed to use the Azure Resource
// Management API, because the API version requested must match the
// type of the resource being manipulated through the API, rather than
// the API version specified statically in the resource client code.
func (env *azureEnviron) collectAPIVersions(ctx context.Context, client *armresources.ProvidersClient) (map[string]string, error) {
	result := make(map[string]string)

	res := client.NewListPager(nil)
	for res.More() {
		p, err := res.NextPage(ctx)
		if err != nil {
			return map[string]string{}, env.HandleCredentialError(ctx, err)
		}

		providers := p.ProviderListResult
		for _, provider := range providers.Value {
			for _, resourceType := range provider.ResourceTypes {
				key := toValue(provider.Namespace) + "/" + toValue(resourceType.ResourceType)
				if len(resourceType.APIVersions) == 0 {
					continue
				}
				// The versions are newest-first.
				result[key] = toValue(resourceType.APIVersions[0])
			}
		}
	}
	return result, nil
}

func azureCloud(cloudName, apiEndpoint, identityEndpoint string) azurecloud.Configuration {
	// Use well known cloud definitions from the SDk if possible.
	switch cloudName {
	case "azure":
		return azurecloud.AzurePublic
	case "azure-china":
		return azurecloud.AzureChina
	case "azure-gov":
		return azurecloud.AzureGovernment
	}
	return azurecloud.Configuration{
		ActiveDirectoryAuthorityHost: identityEndpoint,
		Services: map[azurecloud.ServiceName]azurecloud.ServiceConfiguration{
			azurecloud.ResourceManager: {
				Audience: "https://management.core.windows.net/",
				Endpoint: apiEndpoint,
			},
		},
	}
}

func toValue[T any](v *T) T {
	if v == nil {
		return *new(T)
	}
	return *v
}

func toMap(in map[string]*string) map[string]string {
	result := make(map[string]string)
	for k, v := range in {
		result[k] = toValue(v)
	}
	return result
}

func toMapPtr(in map[string]string) map[string]*string {
	result := make(map[string]*string)
	for k, v := range in {
		result[k] = to.Ptr(v)
	}
	return result
}
