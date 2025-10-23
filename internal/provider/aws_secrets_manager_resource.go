package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.ibm.com/Activity-Insights/terraform-provider-guardium-data-protection/internal/gdp"
)

// AWSSecretsManagerResource defines the resource implementation
type AWSSecretsManagerResource struct {
	client *gdp.Client
}

// AWSSecretsManagerResourceModel describes the resource data model
type AWSSecretsManagerResourceModel struct {
	AccessToken       types.String `tfsdk:"access_token"`
	Name              types.String `tfsdk:"name"`
	AuthType          types.String `tfsdk:"auth_type"`
	AccessKeyID       types.String `tfsdk:"access_key_id"`
	SecretAccessKey   types.String `tfsdk:"secret_access_key"`
	SecretKeyUsername types.String `tfsdk:"secret_key_username"`
	SecretKeyPassword types.String `tfsdk:"secret_key_password"`
	ID                types.String `tfsdk:"id"`
	CaPath            types.String `tfsdk:"ca_path"`
}

func NewAWSSecretsManagerResource() resource.Resource {
	return &AWSSecretsManagerResource{}
}

// Metadata returns the resource type name
func (r *AWSSecretsManagerResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_aws_secrets_manager"
}

// Schema defines the schema for the resource
func (r *AWSSecretsManagerResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "AWS Secrets Manager configuration for Guardium Data Protection",
		Attributes: map[string]schema.Attribute{
			"access_token": schema.StringAttribute{
				MarkdownDescription: "Access token for authentication",
				Required:            true,
				Sensitive:           true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the AWS Secrets Manager configuration",
				Required:            true,
			},
			"auth_type": schema.StringAttribute{
				MarkdownDescription: "Authentication type (e.g., Security-Credentials)",
				Required:            true,
			},
			"access_key_id": schema.StringAttribute{
				MarkdownDescription: "AWS Access Key ID",
				Required:            true,
				Sensitive:           true,
			},
			"secret_access_key": schema.StringAttribute{
				MarkdownDescription: "AWS Secret Access Key",
				Required:            true,
				Sensitive:           true,
			},
			"secret_key_username": schema.StringAttribute{
				MarkdownDescription: "Secret Key Username",
				Required:            true,
			},
			"secret_key_password": schema.StringAttribute{
				MarkdownDescription: "Secret Key Password",
				Required:            true,
				Sensitive:           true,
			},
			"ca_path": schema.StringAttribute{
				MarkdownDescription: "Path to CA certificate",
				Optional:            true,
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
func (r *AWSSecretsManagerResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
func (r *AWSSecretsManagerResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data AWSSecretsManagerResourceModel
	diags := req.Plan.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "Creating AWS Secrets Manager configuration")

	config := gdp.NewAWSSecretsManagerConfig(
		data.Name.ValueString(),
		data.AuthType.ValueString(),
		data.AccessKeyID.ValueString(),
		data.SecretAccessKey.ValueString(),
		data.SecretKeyUsername.ValueString(),
		data.SecretKeyPassword.ValueString(),
	)

	if data.CaPath.IsNull() {
		c := r.client.NewInsecureClient()

		// Check if a configuration with this name already exists
		existingConfig, err := c.GetAWSSecretsManager(ctx, data.AccessToken.ValueString(), data.Name.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Error checking for existing AWS Secrets Manager configuration", fmt.Sprintf("Could not check for existing configuration: %s", err))
			return
		}

		if existingConfig != nil {
			// Configuration already exists, update it
			if err := c.UpdateAWSSecretsManager(ctx, data.AccessToken.ValueString(), config); err != nil {
				resp.Diagnostics.AddError("Error updating existing AWS Secrets Manager configuration", fmt.Sprintf("Could not update existing configuration: %s", err))
				return
			}
		} else {
			// Configuration doesn't exist, create it
			if err := c.CreateAWSSecretsManager(ctx, data.AccessToken.ValueString(), config); err != nil {
				resp.Diagnostics.AddError("Error creating AWS Secrets Manager configuration", fmt.Sprintf("Could not create configuration: %s", err))
				return
			}
		}
	}

	// Set a unique ID for the resource
	data.ID = types.StringValue(data.Name.ValueString())

	// Set state
	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

// Read refreshes the Terraform state with the latest data
func (r *AWSSecretsManagerResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data AWSSecretsManagerResourceModel
	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "Reading AWS Secrets Manager configuration")

	if data.CaPath.IsNull() {
		c := r.client.NewInsecureClient()
		config, err := c.GetAWSSecretsManager(ctx, data.AccessToken.ValueString(), data.Name.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Error reading AWS Secrets Manager configuration", fmt.Sprintf("Could not read AWS Secrets Manager configuration: %s", err))
			return
		}

		// If the resource doesn't exist, remove it from state
		if config == nil {
			resp.State.RemoveResource(ctx)
			return
		}

		// Update the data with the values from the API
		data.Name = types.StringValue(config.Name)
		data.AuthType = types.StringValue(config.AuthType)
		// We don't update sensitive fields from the API response
	}

	// Set state
	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

// Update updates the resource and sets the updated Terraform state
func (r *AWSSecretsManagerResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data AWSSecretsManagerResourceModel
	diags := req.Plan.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "Updating AWS Secrets Manager configuration")

	config := gdp.NewAWSSecretsManagerConfig(
		data.Name.ValueString(),
		data.AuthType.ValueString(),
		data.AccessKeyID.ValueString(),
		data.SecretAccessKey.ValueString(),
		data.SecretKeyUsername.ValueString(),
		data.SecretKeyPassword.ValueString(),
	)

	if data.CaPath.IsNull() {
		c := r.client.NewInsecureClient()
		if err := c.UpdateAWSSecretsManager(ctx, data.AccessToken.ValueString(), config); err != nil {
			resp.Diagnostics.AddError("Error updating AWS Secrets Manager configuration", fmt.Sprintf("Could not update AWS Secrets Manager configuration: %s", err))
			return
		}
	}

	// Set state
	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

// Delete deletes the resource and removes the Terraform state
func (r *AWSSecretsManagerResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data AWSSecretsManagerResourceModel
	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "Deleting AWS Secrets Manager configuration")

	if data.CaPath.IsNull() {
		c := r.client.NewInsecureClient()
		if err := c.DeleteAWSSecretsManager(ctx, data.AccessToken.ValueString(), data.Name.ValueString()); err != nil {
			resp.Diagnostics.AddError("Error deleting AWS Secrets Manager configuration", fmt.Sprintf("Could not delete AWS Secrets Manager configuration: %s", err))
			return
		}
	}
}
