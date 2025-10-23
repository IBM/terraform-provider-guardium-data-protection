package provider

import (
	"context"
	"fmt"
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
	_ resource.Resource                = &configureVADatasourceResource{}
	_ resource.ResourceWithConfigure   = &configureVADatasourceResource{}
	_ resource.ResourceWithImportState = &configureVADatasourceResource{}
)

// NewConfigureVADatasourceResource is a helper function to simplify the provider implementation.
func NewConfigureVADatasourceResource() resource.Resource {
	return &configureVADatasourceResource{}
}

// configureVADatasourceResource is the resource implementation.
type configureVADatasourceResource struct {
	client *gdp.Client
}

// configureVADatasourceResourceModel maps the resource schema data.
type configureVADatasourceResourceModel struct {
	ID                 types.String `tfsdk:"id"`
	DatasourceName     types.String `tfsdk:"datasource_name"`
	AssessmentSchedule types.String `tfsdk:"assessment_schedule"`
	AssessmentDay      types.String `tfsdk:"assessment_day"`
	AssessmentTime     types.String `tfsdk:"assessment_time"`
	Enabled            types.Bool   `tfsdk:"enabled"`
	AccessToken        types.String `tfsdk:"access_token"`
	LastConfiguredTime types.String `tfsdk:"last_configured_time"`
	CAPath             types.String `tfsdk:"ca_path"`
}

func (r *configureVADatasourceResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_configure_va_datasource"
}

func (r *configureVADatasourceResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Resource for configuring vulnerability assessment for a datasource",

		Attributes: map[string]schema.Attribute{
			"datasource_name": schema.StringAttribute{
				MarkdownDescription: "Name of the datasource to configure VA for",
				Required:            true,
			},
			"assessment_schedule": schema.StringAttribute{
				MarkdownDescription: "Schedule frequency for vulnerability assessment (e.g., daily, weekly, monthly)",
				Required:            true,
			},
			"assessment_day": schema.StringAttribute{
				MarkdownDescription: "Day for vulnerability assessment (e.g., Monday for weekly, 1 for monthly)",
				Required:            true,
			},
			"assessment_time": schema.StringAttribute{
				MarkdownDescription: "Time for vulnerability assessment (e.g., 23:00)",
				Required:            true,
			},
			"enabled": schema.BoolAttribute{
				MarkdownDescription: "Whether vulnerability assessment is enabled",
				Optional:            true,
				Computed:            true,
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
			"last_configured_time": schema.StringAttribute{
				MarkdownDescription: "Timestamp of the last configuration",
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

func (r *configureVADatasourceResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *configureVADatasourceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data configureVADatasourceResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Create HTTP client

	// Prepare the payload
	payload, err := gdp.NewConfigureDatasourcePayloadBuilder().
		DatasourceName(data.DatasourceName.ValueString()).
		Frequency(data.AssessmentSchedule.ValueString()).
		Day(data.AssessmentDay.ValueString()).
		Time(data.AssessmentTime.ValueString()).
		Enabled(true).Build()

	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to build payload",
			fmt.Sprintf("Failed to build payload: %s.", err.Error()),
		)
		return
	}

	if data.CAPath.IsNull() {
		err = r.client.NewInsecureClient().ConfigureVADataSource(ctx, data.AccessToken.ValueString(), payload)
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
	data.LastConfiguredTime = types.StringValue(currentTime)

	// Set ID based on datasource name
	data.ID = types.StringValue(fmt.Sprintf("va-config-%s", data.DatasourceName.ValueString()))

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *configureVADatasourceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data configureVADatasourceResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// In a real implementation, you would query the API to check if the VA configuration exists
	// For now, we'll just keep the state as is since the GDP API might not provide a way to check
	// if a VA configuration exists by datasource name

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *configureVADatasourceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data configureVADatasourceResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Prepare the payload
	payload, err := gdp.NewConfigureDatasourcePayloadBuilder().
		DatasourceName(data.DatasourceName.ValueString()).
		Frequency(data.AssessmentSchedule.ValueString()).
		Day(data.AssessmentDay.ValueString()).
		Time(data.AssessmentTime.ValueString()).
		Enabled(true).Build()

	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to build payload",
			fmt.Sprintf("Failed to build payload: %s.", err.Error()),
		)
		return
	}

	if data.CAPath.IsNull() {
		err = r.client.NewInsecureClient().ConfigureVADataSource(ctx, data.AccessToken.ValueString(), payload)
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
	data.LastConfiguredTime = types.StringValue(currentTime)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *configureVADatasourceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data configureVADatasourceResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *configureVADatasourceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
