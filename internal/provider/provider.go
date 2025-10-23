package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.ibm.com/Activity-Insights/terraform-provider-guardium-data-protection/internal/gdp"
)

// Ensure GuardiumDataProtectionProvider satisfies various provider interfaces.
// these are unused functions and are purely defined for the debugging to ensure we are
// satisfying all interfaces correctly
var _ provider.Provider = &GuardiumDataProtectionProvider{}

// GuardiumDataProtectionProvider defines the provider implementation.
type GuardiumDataProtectionProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

type guardiumDataProtectionModel struct {
	Host string `tfsdk:"host"`
	Port string `tfsdk:"port"`
}

func (p *GuardiumDataProtectionProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "guardium-data-protection"
	resp.Version = p.version
}

func (p *GuardiumDataProtectionProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"host": schema.StringAttribute{
				MarkdownDescription: "The Guardium Data Protection host",
				Required:            true,
			},
			"port": schema.StringAttribute{
				MarkdownDescription: "The Guardium Data Protection host",
				Required:            true,
			},
		},
	}
}

// Configure takes in the defined parameters in the TF module and creates a template gdp client for future use
func (p *GuardiumDataProtectionProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	tflog.Info(ctx, "configuration provider configuration")
	var data guardiumDataProtectionModel
	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	client := gdp.NewClient(data.Host, data.Port)

	resp.DataSourceData = client
	resp.ResourceData = client
	tflog.Info(ctx, "provider configuration configured")
}

func (p *GuardiumDataProtectionProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewImportProfilesResource,
		NewInstallConnectorResource,
		NewRegisterVADatasourceResource,
		NewConfigureVADatasourceResource,
		NewConfigureVANotificationsResource,
		NewAWSSecretsManagerResource,
	}
}

func (p *GuardiumDataProtectionProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewAuthenticationDataSource,
	}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &GuardiumDataProtectionProvider{
			version: version,
		}
	}
}
