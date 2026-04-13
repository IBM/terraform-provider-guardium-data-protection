// Copyright (c) IBM Corporation
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

func TestAWSVPCCleanupResource_New(t *testing.T) {
	r := NewAWSVPCCleanupResource()
	if r == nil {
		t.Fatal("NewAWSVPCCleanupResource() returned nil")
	}

	if _, ok := r.(*AWSVPCCleanupResource); !ok {
		t.Fatalf("NewAWSVPCCleanupResource() type = %T, want *AWSVPCCleanupResource", r)
	}
}

func TestAWSVPCCleanupResource_Metadata(t *testing.T) {
	r := NewAWSVPCCleanupResource()
	req := resource.MetadataRequest{
		ProviderTypeName: "guardium-data-protection",
	}
	resp := &resource.MetadataResponse{}

	r.Metadata(context.Background(), req, resp)

	expected := "guardium-data-protection_aws_vpc_cleanup"
	if resp.TypeName != expected {
		t.Fatalf("Metadata() TypeName = %q, want %q", resp.TypeName, expected)
	}
}

func TestAWSVPCCleanupResource_Schema(t *testing.T) {
	r := NewAWSVPCCleanupResource()
	req := resource.SchemaRequest{}
	resp := &resource.SchemaResponse{}

	r.Schema(context.Background(), req, resp)

	if resp.Schema.Attributes == nil {
		t.Fatal("Schema() expected attributes to be defined")
	}

	requiredAttrs := []string{
		"id",
		"vpc_id",
		"region",
		"profile",
		"access_key_id",
		"secret_access_key",
	}

	for _, attrName := range requiredAttrs {
		if _, exists := resp.Schema.Attributes[attrName]; !exists {
			t.Fatalf("Schema() missing attribute %q", attrName)
		}
	}

	idAttr, ok := resp.Schema.Attributes["id"].(schema.StringAttribute)
	if !ok {
		t.Fatal("id attribute is not a StringAttribute")
	}
	if !idAttr.Computed {
		t.Error("id should be computed")
	}
	if len(idAttr.PlanModifiers) == 0 {
		t.Error("id should include plan modifiers")
	}

	vpcIDAttr, ok := resp.Schema.Attributes["vpc_id"].(schema.StringAttribute)
	if !ok {
		t.Fatal("vpc_id attribute is not a StringAttribute")
	}
	if !vpcIDAttr.Required {
		t.Error("vpc_id should be required")
	}
	if len(vpcIDAttr.PlanModifiers) == 0 {
		t.Error("vpc_id should include replace plan modifiers")
	}

	regionAttr, ok := resp.Schema.Attributes["region"].(schema.StringAttribute)
	if !ok {
		t.Fatal("region attribute is not a StringAttribute")
	}
	if !regionAttr.Required {
		t.Error("region should be required")
	}

	profileAttr, ok := resp.Schema.Attributes["profile"].(schema.StringAttribute)
	if !ok {
		t.Fatal("profile attribute is not a StringAttribute")
	}
	if !profileAttr.Optional {
		t.Error("profile should be optional")
	}

	accessKeyAttr, ok := resp.Schema.Attributes["access_key_id"].(schema.StringAttribute)
	if !ok {
		t.Fatal("access_key_id attribute is not a StringAttribute")
	}
	if !accessKeyAttr.Optional {
		t.Error("access_key_id should be optional")
	}
	if !accessKeyAttr.Sensitive {
		t.Error("access_key_id should be sensitive")
	}

	secretKeyAttr, ok := resp.Schema.Attributes["secret_access_key"].(schema.StringAttribute)
	if !ok {
		t.Fatal("secret_access_key attribute is not a StringAttribute")
	}
	if !secretKeyAttr.Optional {
		t.Error("secret_access_key should be optional")
	}
	if !secretKeyAttr.Sensitive {
		t.Error("secret_access_key should be sensitive")
	}
}

func TestBuildVPCCleanupAWSConfig(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name            string
		region          string
		profile         string
		accessKeyID     string
		secretAccessKey string
	}{
		{
			name:   "region only",
			region: "us-east-1",
		},
		{
			name:            "region with static credentials",
			region:          "eu-central-1",
			accessKeyID:     "AKIAIOSFODNN7EXAMPLE",
			secretAccessKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
		},
		{
			name:            "static credentials take precedence over profile",
			region:          "ap-southeast-1",
			profile:         "ignored-profile",
			accessKeyID:     "AKIAIOSFODNN7EXAMPLE",
			secretAccessKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := buildVPCCleanupAWSConfig(ctx, tt.region, tt.profile, tt.accessKeyID, tt.secretAccessKey)
			if err != nil {
				t.Fatalf("buildVPCCleanupAWSConfig() error = %v", err)
			}
			if cfg.Region != tt.region {
				t.Fatalf("buildVPCCleanupAWSConfig() region = %q, want %q", cfg.Region, tt.region)
			}
			if cfg.Credentials == nil {
				t.Fatal("buildVPCCleanupAWSConfig() credentials provider is nil")
			}
		})
	}
}

// Made with Bob
