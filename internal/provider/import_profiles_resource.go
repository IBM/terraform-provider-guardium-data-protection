package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.ibm.com/Activity-Insights/terraform-provider-guardium-data-protection/internal/gdp"
)

// ImportProfilesResource defines the resource implementation
type ImportProfilesResource struct {
	client *gdp.Client
}

// ImportProfilesResourceModel describes the resource data model
type ImportProfilesResourceModel struct {
	AccessToken types.String `tfsdk:"access_token"`
	PathToFile  types.String `tfsdk:"path_to_file"`
	UpdateMode  types.Bool   `tfsdk:"update_mode"`
	ID          types.String `tfsdk:"id"`
	CaPath      types.String `tfsdk:"ca_path"`
}

func NewImportProfilesResource() resource.Resource {
	return &ImportProfilesResource{}
}

// Metadata returns the resource type name
func (r *ImportProfilesResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_import_profiles"
}

// Schema defines the schema for the resource
func (r *ImportProfilesResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Import profiles from a file",
		Attributes: map[string]schema.Attribute{
			"access_token": schema.StringAttribute{
				MarkdownDescription: "Access token for authentication",
				Required:            true,
				Sensitive:           true,
			},
			"path_to_file": schema.StringAttribute{
				MarkdownDescription: "Path to the file to import",
				Required:            true,
			},
			"ca_path": schema.StringAttribute{
				MarkdownDescription: "Path to the file to import",
				Optional:            true,
			},
			"update_mode": schema.BoolAttribute{
				MarkdownDescription: "Update mode",
				Required:            true,
			},
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Resource identifier",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

// Configure adds the provider configured client to the resource
func (r *ImportProfilesResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*gdp.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Resource Configure Type", fmt.Sprintf("Expected *gdp.Client, got: %T", req.ProviderData))
		return
	}

	r.client = client
}

// Create creates the resource and sets the initial Terraform state
func (r *ImportProfilesResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ImportProfilesResourceModel
	diags := req.Plan.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if data.CaPath.IsNull() {
		c := r.client.NewInsecureClient()
		if err := c.ImportProfilesFromFile(ctx, data.AccessToken.ValueString(), data.PathToFile.ValueString(), data.UpdateMode.ValueBool()); err != nil {
			resp.Diagnostics.AddError("Error importing profiles", fmt.Sprintf("Could not import profiles: %s", err))
			return
		}
	}

	// Set a unique ID for the resource
	data.ID = types.StringValue(fmt.Sprintf("%s-%s", r.client.Host, data.PathToFile.ValueString()))

	// Set state
	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

// Read refreshes the Terraform state with the latest data
func (r *ImportProfilesResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ImportProfilesResourceModel
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
func (r *ImportProfilesResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data ImportProfilesResourceModel
	diags := req.Plan.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if data.CaPath.IsNull() {
		c := r.client.NewInsecureClient()
		if err := c.ImportProfilesFromFile(ctx, data.AccessToken.ValueString(), data.PathToFile.ValueString(), data.UpdateMode.ValueBool()); err != nil {
			resp.Diagnostics.AddError(
				"Error importing profiles",
				fmt.Sprintf("Could not import profiles: %s", err),
			)
			return
		}
	}

	// Set state
	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

// Delete deletes the resource and removes the Terraform state
func (r *ImportProfilesResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// This is a no-op as there's nothing to delete
}
