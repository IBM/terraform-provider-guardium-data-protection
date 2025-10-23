package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.ibm.com/Activity-Insights/terraform-provider-guardium-data-protection/internal/gdp"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &AuthenticationDataSource{}
var _ datasource.DataSourceWithConfigure = &AuthenticationDataSource{}

func NewAuthenticationDataSource() datasource.DataSource {
	return &AuthenticationDataSource{}
}

// AuthenticationDataSource defines the data source implementation.
type AuthenticationDataSource struct {
	client *gdp.Client
}

// AuthenticationDataSourceModel describes the data source data model.
type AuthenticationDataSourceModel struct {
	ClientSecret types.String `tfsdk:"client_secret"`
	ClientID     types.String `tfsdk:"client_id"`
	Username     types.String `tfsdk:"username"`
	Password     types.String `tfsdk:"password"`
	CAPath       types.String `tfsdk:"ca_path"`
	AccessToken  types.String `tfsdk:"access_token"`
}

// Metadata defines the provider name, since we are being called by a parent `guardium_data_protection`
// the below will be available through `guardium_data_protection_authentication`
func (d *AuthenticationDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_authentication"
}

func (d *AuthenticationDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Example data source",

		// Attributes are the hcl implementation of the above AuthenticationDataSourceModel
		Attributes: map[string]schema.Attribute{
			"client_secret": schema.StringAttribute{
				MarkdownDescription: "Guardium Data Protection Client Secret",
				Required:            true,
				Sensitive:           true,
				// Sensitive means values will still show up in the state, but will be protected at cli logs
			},
			"client_id": schema.StringAttribute{
				MarkdownDescription: "Guardium Data Protection Client Secret",
				Required:            true,
			},
			"username": schema.StringAttribute{
				MarkdownDescription: "Guardium Data Protection username",
				Required:            true,
			},
			"password": schema.StringAttribute{
				MarkdownDescription: "Guardium Data Protection password",
				Required:            true,
				Sensitive:           true,
			},
			"ca_path": schema.StringAttribute{
				MarkdownDescription: "Guardium Data Protection certificate authority",
				Optional:            true,
			},
			"access_token": schema.StringAttribute{
				Computed:  true,
				Sensitive: true,
			},
		},
	}
}

func (d *AuthenticationDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	tflog.Info(ctx, "entering configure for data source")
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*gdp.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *gdp.Client, got: %T.", req.ProviderData),
		)

		return
	}

	d.client = client
}

type AuthenticationResponse struct {
	AccessToken types.String `tfsdk:"access_token"`
}

func (d *AuthenticationDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data = new(AuthenticationDataSourceModel)
	diags := req.Config.Get(ctx, data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	var (
		accessToken string
		err         error
	)

	if data.CAPath.IsNull() {
		accessToken, err = d.client.NewInsecureClient().GenerateAccessToken(ctx, data.ClientSecret.ValueString(), data.Username.ValueString(), data.Password.ValueString(), data.ClientID.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(
				"Failed to retrieve access token",
				fmt.Sprintf("Failed to retrieve access token: %s.", err.Error()),
			)
			return
		}
	}
	tflog.Info(ctx, accessToken)
	data.AccessToken = types.StringValue(accessToken)
	resp.State.Set(ctx, data)
}
