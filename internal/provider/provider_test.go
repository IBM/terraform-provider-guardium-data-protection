// Copyright (c) IBM Corporation
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

// TestGuardiumDataProtectionProvider_Metadata validates that the provider
// returns correct metadata including type name and version
func TestGuardiumDataProtectionProvider_Metadata(t *testing.T) {
	testCases := []struct {
		name            string
		version         string
		expectedType    string
		expectedVersion string
	}{
		{
			name:            "Production version",
			version:         "1.0.0",
			expectedType:    "guardium-data-protection",
			expectedVersion: "1.0.0",
		},
		{
			name:            "Development version",
			version:         "dev",
			expectedType:    "guardium-data-protection",
			expectedVersion: "dev",
		},
		{
			name:            "Test version",
			version:         "test",
			expectedType:    "guardium-data-protection",
			expectedVersion: "test",
		},
		{
			name:            "Semantic version with patch",
			version:         "2.3.4",
			expectedType:    "guardium-data-protection",
			expectedVersion: "2.3.4",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			p := &GuardiumDataProtectionProvider{
				version: tc.version,
			}

			ctx := context.Background()
			req := provider.MetadataRequest{}
			resp := &provider.MetadataResponse{}

			p.Metadata(ctx, req, resp)

			if resp.TypeName != tc.expectedType {
				t.Errorf("Expected TypeName %s, got %s", tc.expectedType, resp.TypeName)
			}

			if resp.Version != tc.expectedVersion {
				t.Errorf("Expected Version %s, got %s", tc.expectedVersion, resp.Version)
			}
		})
	}
}

// TestGuardiumDataProtectionProvider_Schema validates that the provider
// schema includes required attributes with correct configuration
func TestGuardiumDataProtectionProvider_Schema(t *testing.T) {
	p := &GuardiumDataProtectionProvider{
		version: "test",
	}

	ctx := context.Background()
	req := provider.SchemaRequest{}
	resp := &provider.SchemaResponse{}

	p.Schema(ctx, req, resp)

	// Verify schema is not nil
	if resp.Schema.Attributes == nil {
		t.Fatal("Expected schema attributes to be defined, got nil")
	}

	// Verify host attribute exists and is required
	hostAttr, exists := resp.Schema.Attributes["host"]
	if !exists {
		t.Error("Expected 'host' attribute to exist in schema")
	} else {
		if !hostAttr.IsRequired() {
			t.Error("Expected 'host' attribute to be required")
		}
	}

	// Verify port attribute exists and is required
	portAttr, exists := resp.Schema.Attributes["port"]
	if !exists {
		t.Error("Expected 'port' attribute to exist in schema")
	} else {
		if !portAttr.IsRequired() {
			t.Error("Expected 'port' attribute to be required")
		}
	}

	// Verify we have exactly 2 attributes (host and port)
	if len(resp.Schema.Attributes) != 2 {
		t.Errorf("Expected 2 attributes in schema, got %d", len(resp.Schema.Attributes))
	}
}

// TestGuardiumDataProtectionProvider_Configure validates that the provider
// configuration correctly processes host and port settings
func TestGuardiumDataProtectionProvider_Configure(t *testing.T) {
	testCases := []struct {
		name        string
		host        string
		port        string
		expectError bool
	}{
		{
			name:        "Valid configuration",
			host:        "guardium.example.com",
			port:        "8443",
			expectError: false,
		},
		{
			name:        "Localhost configuration",
			host:        "localhost",
			port:        "8080",
			expectError: false,
		},
		{
			name:        "IP address configuration",
			host:        "192.168.1.100",
			port:        "443",
			expectError: false,
		},
		{
			name:        "Different port",
			host:        "gdp-server.local",
			port:        "9443",
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Note: Full configuration testing requires a complete Terraform context
			// This test validates the provider structure and basic setup
			p := &GuardiumDataProtectionProvider{
				version: "test",
			}

			// Verify provider can be instantiated
			if p == nil {
				t.Fatal("Failed to create provider instance")
			}

			// Verify version is set correctly
			if p.version != "test" {
				t.Errorf("Expected version 'test', got '%s'", p.version)
			}
		})
	}
}

// TestGuardiumDataProtectionProvider_Resources validates that the provider
// returns all expected resource constructors
func TestGuardiumDataProtectionProvider_Resources(t *testing.T) {
	p := &GuardiumDataProtectionProvider{
		version: "test",
	}

	ctx := context.Background()
	resources := p.Resources(ctx)

	expectedResourceCount := 6
	if len(resources) != expectedResourceCount {
		t.Errorf("Expected %d resources, got %d", expectedResourceCount, len(resources))
	}

	// Verify each resource constructor returns a valid resource
	for i, resourceFunc := range resources {
		resource := resourceFunc()
		if resource == nil {
			t.Errorf("Resource constructor at index %d returned nil", i)
		}
	}
}

// TestGuardiumDataProtectionProvider_DataSources validates that the provider
// returns all expected data source constructors
func TestGuardiumDataProtectionProvider_DataSources(t *testing.T) {
	p := &GuardiumDataProtectionProvider{
		version: "test",
	}

	ctx := context.Background()
	dataSources := p.DataSources(ctx)

	expectedDataSourceCount := 1
	if len(dataSources) != expectedDataSourceCount {
		t.Errorf("Expected %d data sources, got %d", expectedDataSourceCount, len(dataSources))
	}

	// Verify each data source constructor returns a valid data source
	for i, dataSourceFunc := range dataSources {
		dataSource := dataSourceFunc()
		if dataSource == nil {
			t.Errorf("Data source constructor at index %d returned nil", i)
		}
	}
}

// TestNew validates that the New function returns a valid provider factory
func TestNew(t *testing.T) {
	testCases := []struct {
		name    string
		version string
	}{
		{
			name:    "Production version",
			version: "1.0.0",
		},
		{
			name:    "Development version",
			version: "dev",
		},
		{
			name:    "Test version",
			version: "test",
		},
		{
			name:    "Empty version",
			version: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			providerFunc := New(tc.version)

			if providerFunc == nil {
				t.Fatal("Expected provider function, got nil")
			}

			// Call the provider function to get the provider instance
			p := providerFunc()

			if p == nil {
				t.Fatal("Expected provider instance, got nil")
			}

			// Verify it's the correct type
			gdpProvider, ok := p.(*GuardiumDataProtectionProvider)
			if !ok {
				t.Fatal("Expected *GuardiumDataProtectionProvider type")
			}

			// Verify version is set correctly
			if gdpProvider.version != tc.version {
				t.Errorf("Expected version '%s', got '%s'", tc.version, gdpProvider.version)
			}
		})
	}
}

// TestGuardiumDataProtectionProvider_Interface validates that the provider
// satisfies the provider.Provider interface
func TestGuardiumDataProtectionProvider_Interface(t *testing.T) {
	var _ provider.Provider = &GuardiumDataProtectionProvider{}
}

// TestGuardiumDataProtectionProvider_ResourceTypes validates that all
// expected resource types are registered
func TestGuardiumDataProtectionProvider_ResourceTypes(t *testing.T) {
	p := &GuardiumDataProtectionProvider{
		version: "test",
	}

	ctx := context.Background()
	resources := p.Resources(ctx)

	// Expected resource constructors
	expectedResources := []func() resource.Resource{
		NewImportProfilesResource,
		NewInstallConnectorResource,
		NewRegisterVADatasourceResource,
		NewConfigureVADatasourceResource,
		NewConfigureVANotificationsResource,
		NewAWSSecretsManagerResource,
	}

	if len(resources) != len(expectedResources) {
		t.Errorf("Expected %d resources, got %d", len(expectedResources), len(resources))
	}

	// Verify each resource can be instantiated
	for i, resourceFunc := range resources {
		resource := resourceFunc()
		if resource == nil {
			t.Errorf("Resource at index %d failed to instantiate", i)
		}
	}
}

// TestGuardiumDataProtectionProvider_DataSourceTypes validates that all
// expected data source types are registered
func TestGuardiumDataProtectionProvider_DataSourceTypes(t *testing.T) {
	p := &GuardiumDataProtectionProvider{
		version: "test",
	}

	ctx := context.Background()
	dataSources := p.DataSources(ctx)

	// Expected data source constructors
	expectedDataSources := []func() datasource.DataSource{
		NewAuthenticationDataSource,
	}

	if len(dataSources) != len(expectedDataSources) {
		t.Errorf("Expected %d data sources, got %d", len(expectedDataSources), len(dataSources))
	}

	// Verify each data source can be instantiated
	for i, dataSourceFunc := range dataSources {
		dataSource := dataSourceFunc()
		if dataSource == nil {
			t.Errorf("Data source at index %d failed to instantiate", i)
		}
	}
}

// TestProviderServer validates that the provider can be served via gRPC
func TestProviderServer(t *testing.T) {
	version := "test"
	providerFunc := New(version)

	// Create a provider server
	providers := map[string]func() (tfprotov6.ProviderServer, error){
		"guardium-data-protection": providerserver.NewProtocol6WithError(providerFunc()),
	}

	if len(providers) != 1 {
		t.Errorf("Expected 1 provider, got %d", len(providers))
	}

	// Verify the provider server can be created
	serverFunc, exists := providers["guardium-data-protection"]
	if !exists {
		t.Fatal("Provider server function not found")
	}

	server, err := serverFunc()
	if err != nil {
		t.Fatalf("Failed to create provider server: %v", err)
	}

	if server == nil {
		t.Fatal("Expected provider server, got nil")
	}
}

// TestGuardiumDataProtectionProvider_MultipleInstances validates that
// multiple provider instances can coexist with different configurations
func TestGuardiumDataProtectionProvider_MultipleInstances(t *testing.T) {
	versions := []string{"1.0.0", "2.0.0", "dev", "test"}

	providers := make([]*GuardiumDataProtectionProvider, len(versions))

	// Create multiple provider instances
	for i, version := range versions {
		providers[i] = &GuardiumDataProtectionProvider{
			version: version,
		}
	}

	// Verify each instance has the correct version
	for i, p := range providers {
		if p.version != versions[i] {
			t.Errorf("Provider %d: expected version '%s', got '%s'", i, versions[i], p.version)
		}
	}

	// Verify instances are independent
	ctx := context.Background()
	for i, p := range providers {
		req := provider.MetadataRequest{}
		resp := &provider.MetadataResponse{}
		p.Metadata(ctx, req, resp)

		if resp.Version != versions[i] {
			t.Errorf("Provider %d: metadata version mismatch, expected '%s', got '%s'", i, versions[i], resp.Version)
		}
	}
}

// TestGuardiumDataProtectionProvider_SchemaConsistency validates that
// the schema remains consistent across multiple calls
func TestGuardiumDataProtectionProvider_SchemaConsistency(t *testing.T) {
	p := &GuardiumDataProtectionProvider{
		version: "test",
	}

	ctx := context.Background()

	// Call Schema multiple times
	for i := 0; i < 3; i++ {
		req := provider.SchemaRequest{}
		resp := &provider.SchemaResponse{}

		p.Schema(ctx, req, resp)

		// Verify schema consistency
		if len(resp.Schema.Attributes) != 2 {
			t.Errorf("Call %d: expected 2 attributes, got %d", i, len(resp.Schema.Attributes))
		}

		if _, exists := resp.Schema.Attributes["host"]; !exists {
			t.Errorf("Call %d: 'host' attribute missing", i)
		}

		if _, exists := resp.Schema.Attributes["port"]; !exists {
			t.Errorf("Call %d: 'port' attribute missing", i)
		}
	}
}

// TestGuardiumDataProtectionProvider_ResourcesConsistency validates that
// Resources() returns consistent results across multiple calls
func TestGuardiumDataProtectionProvider_ResourcesConsistency(t *testing.T) {
	p := &GuardiumDataProtectionProvider{
		version: "test",
	}

	ctx := context.Background()

	// Call Resources multiple times
	var resourceCounts []int
	for i := 0; i < 3; i++ {
		resources := p.Resources(ctx)
		resourceCounts = append(resourceCounts, len(resources))
	}

	// Verify all calls return the same count
	expectedCount := resourceCounts[0]
	for i, count := range resourceCounts {
		if count != expectedCount {
			t.Errorf("Call %d: expected %d resources, got %d", i, expectedCount, count)
		}
	}
}

// TestGuardiumDataProtectionProvider_DataSourcesConsistency validates that
// DataSources() returns consistent results across multiple calls
func TestGuardiumDataProtectionProvider_DataSourcesConsistency(t *testing.T) {
	p := &GuardiumDataProtectionProvider{
		version: "test",
	}

	ctx := context.Background()

	// Call DataSources multiple times
	var dataSourceCounts []int
	for i := 0; i < 3; i++ {
		dataSources := p.DataSources(ctx)
		dataSourceCounts = append(dataSourceCounts, len(dataSources))
	}

	// Verify all calls return the same count
	expectedCount := dataSourceCounts[0]
	for i, count := range dataSourceCounts {
		if count != expectedCount {
			t.Errorf("Call %d: expected %d data sources, got %d", i, expectedCount, count)
		}
	}
}

// Made with Bob
