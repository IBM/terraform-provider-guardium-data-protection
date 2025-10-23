package provider

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.ibm.com/Activity-Insights/terraform-provider-guardium-data-protection/internal/gdp"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &configureVANotificationsResource{}
	_ resource.ResourceWithConfigure   = &configureVANotificationsResource{}
	_ resource.ResourceWithImportState = &configureVANotificationsResource{}
)

// NewConfigureVANotificationsResource is a helper function to simplify the provider implementation.
func NewConfigureVANotificationsResource() resource.Resource {
	return &configureVANotificationsResource{}
}

// configureVANotificationsResource is the resource implementation.
type configureVANotificationsResource struct {
	client *gdp.Client
}

// configureVANotificationsResourceModel maps the resource schema data.
type configureVANotificationsResourceModel struct {
	ID                   types.String   `tfsdk:"id"`
	DatasourceName       types.String   `tfsdk:"datasource_name"`
	NotificationType     types.String   `tfsdk:"notification_type"`
	NotificationEmails   []types.String `tfsdk:"notification_emails"`
	NotificationSeverity types.String   `tfsdk:"notification_severity"`
	Enabled              types.Bool     `tfsdk:"enabled"`
	AccessToken          types.String   `tfsdk:"access_token"`
	LastConfiguredTime   types.String   `tfsdk:"last_configured_time"`
	CAPath               types.String   `tfsdk:"ca_path"`
}

func (r *configureVANotificationsResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_configure_va_notifications"
}

func (r *configureVANotificationsResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Resource for configuring vulnerability assessment notifications for a datasource",

		Attributes: map[string]schema.Attribute{
			"datasource_name": schema.StringAttribute{
				MarkdownDescription: "Name of the datasource to configure notifications for",
				Required:            true,
			},
			"notification_type": schema.StringAttribute{
				MarkdownDescription: "Type of notification (e.g., email)",
				Required:            true,
			},
			"notification_emails": schema.ListAttribute{
				MarkdownDescription: "List of email addresses to send notifications to",
				Required:            true,
				ElementType:         types.StringType,
			},
			"notification_severity": schema.StringAttribute{
				MarkdownDescription: "Severity level for notifications (e.g., high, medium, low)",
				Required:            true,
			},
			"enabled": schema.BoolAttribute{
				MarkdownDescription: "Whether notifications are enabled",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
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

func (r *configureVANotificationsResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *configureVANotificationsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data configureVANotificationsResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Convert notification emails from types.String to string
	notificationEmails := make([]string, 0, len(data.NotificationEmails))
	for _, email := range data.NotificationEmails {
		notificationEmails = append(notificationEmails, email.ValueString())
	}

	// Prepare the payload
	payload, err := gdp.NewConfigureNotificationsPayloadBuilder().
		DatasourceName(data.DatasourceName.ValueString()).
		NotificationType(data.NotificationType.ValueString()).
		Recipients(notificationEmails).
		Severity(data.NotificationSeverity.ValueString()).
		Enabled(true).
		Build()

	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to build payload",
			fmt.Sprintf("Failed to build payload: %s.", err.Error()),
		)
		return
	}

	if data.CAPath.IsNull() {
		err = r.client.NewInsecureClient().ConfigureVANotifications(ctx, data.AccessToken.ValueString(), payload)
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
	data.ID = types.StringValue(fmt.Sprintf("va-notifications-%s", data.DatasourceName.ValueString()))

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *configureVANotificationsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data configureVANotificationsResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// In a real implementation, you would query the API to check if the notifications configuration exists
	// For now, we'll just keep the state as is since the GDP API might not provide a way to check
	// if a notifications configuration exists by datasource name

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *configureVANotificationsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data configureVANotificationsResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Convert notification emails from types.String to string
	notificationEmails := make([]string, 0, len(data.NotificationEmails))
	for _, email := range data.NotificationEmails {
		notificationEmails = append(notificationEmails, email.ValueString())
	}

	// Prepare the payload
	payload, err := gdp.NewConfigureNotificationsPayloadBuilder().
		DatasourceName(data.DatasourceName.ValueString()).
		NotificationType(data.NotificationType.ValueString()).
		Recipients(notificationEmails).
		Severity(data.NotificationSeverity.ValueString()).
		Enabled(true).
		Build()

	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to build payload",
			fmt.Sprintf("Failed to build payload: %s.", err.Error()),
		)
		return
	}

	if data.CAPath.IsNull() {
		err = r.client.NewInsecureClient().ConfigureVANotifications(ctx, data.AccessToken.ValueString(), payload)
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

func (r *configureVANotificationsResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data configureVANotificationsResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// In a real implementation, you might want to disable notifications for the datasource
	// For now, we'll just remove it from state since the GDP API might not provide a way to delete
	// a notifications configuration by datasource name

	// No action needed on delete - the resource will be removed from state
}

func (r *configureVANotificationsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
