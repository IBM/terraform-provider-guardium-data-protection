// Copyright (c) IBM Corporation
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.ibm.com/Activity-Insights/terraform-provider-guardium-data-protection/internal/gdp"
)

// InstallConnectorResource defines the resource implementation
type InstallConnectorResource struct {
	client *gdp.Client
}

// InstallConnectorResourceModel describes the resource data model
type InstallConnectorResourceModel struct {
	AccessToken types.String `tfsdk:"access_token"`
	CAPath      types.String `tfsdk:"ca_path"`
	UdcName     types.String `tfsdk:"udc_name"`
	GdpMuHost   types.String `tfsdk:"gdp_mu_host"`
	ID          types.String `tfsdk:"id"`
}

func NewInstallConnectorResource() resource.Resource {
	return &InstallConnectorResource{}
}

// Metadata returns the resource type name
func (r *InstallConnectorResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_install_connector"
}

// Schema defines the schema for the resource
func (r *InstallConnectorResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Install connector in bulk",
		Attributes: map[string]schema.Attribute{
			"access_token": schema.StringAttribute{
				MarkdownDescription: "Access token for authentication",
				Required:            true,
				Sensitive:           true,
			},
			"ca_path": schema.StringAttribute{
				MarkdownDescription: "Guardium Data Protection server certificate authority path",
				Optional:            true,
			},
			"udc_name": schema.StringAttribute{
				MarkdownDescription: "UDC profile name",
				Required:            true,
			},
			"gdp_mu_host": schema.StringAttribute{
				MarkdownDescription: "GDP MU host",
				Required:            true,
			},
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Resource identifier",
			},
		},
	}
}

// Configure adds the provider configured client to the resource
func (r *InstallConnectorResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*gdp.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *gdp.Client, got: %T", req.ProviderData),
		)
		return
	}

	r.client = client
}

// Create creates the resource and sets the initial Terraform state
func (r *InstallConnectorResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data InstallConnectorResourceModel
	diags := req.Plan.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if data.CAPath.IsNull() {

		// Make the API call to install connector
		err := r.client.NewInsecureClient().BulkInstallConnector(ctx, data.AccessToken.ValueString(), data.UdcName.ValueString(), data.GdpMuHost.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(
				"Error installing connector",
				fmt.Sprintf("Could not install connector: %s", err),
			)
			return
		}
	}

	// Set a unique ID for the resource
	data.ID = types.StringValue(fmt.Sprintf("%s-%s-%s", r.client.Host, data.UdcName.ValueString(), data.GdpMuHost.ValueString()))

	// Set state
	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

// Read refreshes the Terraform state with the latest data
func (r *InstallConnectorResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data InstallConnectorResourceModel
	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// No need to refresh anything for this resource
	// Just set the state as is
	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

// Update updates the resource and sets the updated Terraform state
func (r *InstallConnectorResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data InstallConnectorResourceModel
	diags := req.Plan.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if data.CAPath.IsNull() {
		// Make the API call to install connector
		err := r.client.NewInsecureClient().BulkInstallConnector(ctx, data.AccessToken.ValueString(), data.UdcName.ValueString(), data.GdpMuHost.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Error installing connector", fmt.Sprintf("Could not install connector: %s", err))
			return
		}
	}

	// Set state
	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

// Delete deletes the resource and removes the Terraform state
func (r *InstallConnectorResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// This is a no-op as there's nothing to delete
}
