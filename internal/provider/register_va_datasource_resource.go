// Copyright (c) IBM Corporation
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"crypto/sha256"
	"fmt"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"strconv"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.ibm.com/Activity-Insights/terraform-provider-guardium-data-protection/internal/gdp"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &registerVADatasourceResource{}
	_ resource.ResourceWithConfigure   = &registerVADatasourceResource{}
	_ resource.ResourceWithImportState = &registerVADatasourceResource{}
)

// NewRegisterVADatasourceResource is a helper function to simplify the provider implementation.
func NewRegisterVADatasourceResource() resource.Resource {
	return &registerVADatasourceResource{}
}

// registerVADatasourceResource is the resource implementation.
type registerVADatasourceResource struct {
	client *gdp.Client
}

// registerVADatasourceResourceModel maps the resource schema data.
type registerVADatasourceResourceModel struct {
	ID                 types.String `tfsdk:"id"`
	AccessToken        types.String `tfsdk:"access_token"`
	Payload            types.String `tfsdk:"payload"`
	CAPath             types.String `tfsdk:"ca_path"`
	LastRegisteredTime types.String `tfsdk:"last_registered_time"`
}

func (r *registerVADatasourceResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_register_va_datasource"
}

func (r *registerVADatasourceResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Resource for registering a VA datasource",

		Attributes: map[string]schema.Attribute{
			"payload": schema.StringAttribute{
				MarkdownDescription: "Access token for authentication",
				Required:            true,
				Sensitive:           true,
			},
			"access_token": schema.StringAttribute{
				MarkdownDescription: "Access token for authentication",
				Required:            true,
				Sensitive:           true,
			},
			"ca_path": schema.StringAttribute{
				MarkdownDescription: "Guardium Data Protection certificate authority",
				Optional:            true,
			},
			"last_registered_time": schema.StringAttribute{
				MarkdownDescription: "Timestamp of the last registration",
				Computed:            true,
			},
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Identifier of the resource",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *registerVADatasourceResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*gdp.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *gdp.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = client
}

func (r *registerVADatasourceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data registerVADatasourceResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var (
		payload = data.Payload.ValueString()
		err     error
	)

	if payload[0] == '"' {
		payload, err = strconv.Unquote(data.Payload.ValueString())
		if err != nil {
			tflog.Info(ctx, fmt.Sprintf("payload %s", data.Payload.ValueString()))
			resp.Diagnostics.AddError(
				"Failed to unquote payload",
				fmt.Sprintf("Failed to unquote payload: %s.", err.Error()),
			)
			return
		}
	}

	if data.CAPath.IsNull() {
		err := r.client.NewInsecureClient().RegisterVADataSource(ctx, data.AccessToken.ValueString(), []byte(payload))
		if err != nil {
			resp.Diagnostics.AddError(
				"Failed to register va",
				fmt.Sprintf("Failed to register va: %s.", err.Error()),
			)
			return
		}
	}

	// Set computed values
	currentTime := time.Now().Format(time.RFC3339)
	data.LastRegisteredTime = types.StringValue(currentTime)

	// Set ID based on datasource name and host
	hash := sha256.New()

	data.ID = types.StringValue(fmt.Sprintf("%s-%s", hash.Sum([]byte(data.Payload.ValueString())), currentTime))

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *registerVADatasourceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data registerVADatasourceResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// In a real implementation, you would query the API to check if the datasource exists
	// For now, we'll just keep the state as is since the GDP API doesn't provide a way to check
	// if a datasource exists by ID

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *registerVADatasourceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data registerVADatasourceResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var (
		payload = data.Payload.ValueString()
		err     error
	)

	if payload[0] == '"' {
		payload, err = strconv.Unquote(data.Payload.ValueString())
		if err != nil {
			tflog.Info(ctx, fmt.Sprintf("payload %s", data.Payload.ValueString()))
			resp.Diagnostics.AddError(
				"Failed to unquote payload",
				fmt.Sprintf("Failed to unquote payload: %s.", err.Error()),
			)
			return
		}
	}

	if data.CAPath.IsNull() {
		err := r.client.NewInsecureClient().RegisterVADataSource(ctx, data.AccessToken.ValueString(), []byte(payload))
		if err != nil {
			resp.Diagnostics.AddError(
				"Failed to register va",
				fmt.Sprintf("Failed to register va: %s.", err.Error()),
			)
			return
		}
	}

	// Set computed values
	currentTime := time.Now().Format(time.RFC3339)
	data.LastRegisteredTime = types.StringValue(currentTime)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *registerVADatasourceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data registerVADatasourceResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// In a real implementation, you would call the API to delete the datasource
	// For now, we'll just remove it from state since the GDP API doesn't provide a way to delete
	// a datasource by ID

	// No action needed on delete - the resource will be removed from state
}

func (r *registerVADatasourceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
