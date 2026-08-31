// Copyright (c) IBM Corporation
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.ibm.com/Activity-Insights/terraform-provider-guardium-data-protection/internal/edgeclient"
	"github.ibm.com/Activity-Insights/terraform-provider-guardium-data-protection/internal/gdp"
	"github.ibm.com/Activity-Insights/terraform-provider-guardium-data-protection/internal/k3sclient"
	"github.ibm.com/Activity-Insights/terraform-provider-guardium-data-protection/internal/rookcephclient"
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

// UnifiedClient holds all client instances
type UnifiedClient struct {
	GDPClient      *gdp.Client
	EdgeClient     *edgeclient.Client
	K3sClient      *k3sclient.Client
	RookCephClient *rookcephclient.Client
}

type guardiumDataProtectionModel struct {
	// Original GDP fields
	Host       types.String `tfsdk:"host"`
	Port       types.String `tfsdk:"port"`
	CACertPath types.String `tfsdk:"ca_cert_path"`

	// K3s config
	K3sSSHUser             types.String `tfsdk:"k3s_ssh_user"`
	K3sSSHPassword         types.String `tfsdk:"k3s_ssh_password"`
	K3sConnectTimeout      types.Int64  `tfsdk:"k3s_connect_timeout"`
	K3sServerAliveInterval types.Int64  `tfsdk:"k3s_server_alive_interval"`
	K3sServerAliveCount    types.Int64  `tfsdk:"k3s_server_alive_count"`

	// Rook-Ceph config
	RookCephSSHUser             types.String `tfsdk:"rook_ceph_ssh_user"`
	RookCephSSHPassword         types.String `tfsdk:"rook_ceph_ssh_password"`
	RookCephConnectTimeout      types.Int64  `tfsdk:"rook_ceph_connect_timeout"`
	RookCephServerAliveInterval types.Int64  `tfsdk:"rook_ceph_server_alive_interval"`
	RookCephServerAliveCount    types.Int64  `tfsdk:"rook_ceph_server_alive_count"`

	// Edge config
	CMUrl               types.String `tfsdk:"cm_url"`
	OAuthToken          types.String `tfsdk:"oauth_token"`
	Platform            types.String `tfsdk:"platform"`
	SSHUser             types.String `tfsdk:"ssh_user"`
	SSHPassword         types.String `tfsdk:"ssh_password"`
	SSHKeyPath          types.String `tfsdk:"ssh_key_path"`
	AWSRegion           types.String `tfsdk:"aws_region"`
	AWSProfile          types.String `tfsdk:"aws_profile"`
	AWSAccessKey        types.String `tfsdk:"aws_access_key"`
	AWSSecretKey        types.String `tfsdk:"aws_secret_key"`
	EKSSSHUser          types.String `tfsdk:"eks_ssh_user"`
	EKSSSHKeyPath       types.String `tfsdk:"eks_ssh_key_path"`
	EKSSSHKeyPassphrase types.String `tfsdk:"eks_ssh_key_passphrase"`
	EKSHostnameType     types.String `tfsdk:"eks_hostname_type"`
	// OpenShift native OAuth authentication
	OCPServer             types.String `tfsdk:"ocp_server"`
	OCPUsername           types.String `tfsdk:"ocp_username"`
	OCPPassword           types.String `tfsdk:"ocp_password"`
	OCPToken              types.String `tfsdk:"ocp_token"`
	OCPInsecureSkipVerify types.Bool   `tfsdk:"ocp_insecure_skip_verify"`
}

func (p *GuardiumDataProtectionProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "guardium-data-protection"
	resp.Version = p.version
}

func (p *GuardiumDataProtectionProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Guardium Data Protection provider for managing GDP resources and deploying Edge components using native Kubernetes API.",
		Attributes: map[string]schema.Attribute{
			// Original GDP attributes
			"host": schema.StringAttribute{
				MarkdownDescription: "The Guardium Data Protection host",
				Optional:            true,
			},
			"port": schema.StringAttribute{
				MarkdownDescription: "The Guardium Data Protection port",
				Optional:            true,
			},
			"ca_cert_path": schema.StringAttribute{
				MarkdownDescription: "Path to the GDP TLS CA certificate PEM file. Required for self-signed GDP certificates (the default auto-generated certificate). Leave empty to use the OS system trust store (for CA-signed certificates). Can also be set via the GDP_CA_CERT_PATH environment variable.",
				Optional:            true,
			},

			// K3s attributes
			"k3s_ssh_user": schema.StringAttribute{
				MarkdownDescription: "SSH username for K3s nodes. Can also be set via K3S_SSH_USER environment variable.",
				Optional:            true,
			},
			"k3s_ssh_password": schema.StringAttribute{
				MarkdownDescription: "SSH password for K3s nodes. Can also be set via K3S_SSH_PASSWORD environment variable.",
				Optional:            true,
				Sensitive:           true,
			},
			"k3s_connect_timeout": schema.Int64Attribute{
				MarkdownDescription: "SSH connection timeout in seconds for K3s operations. Defaults to 30.",
				Optional:            true,
			},
			"k3s_server_alive_interval": schema.Int64Attribute{
				MarkdownDescription: "SSH keepalive interval in seconds for K3s operations. Defaults to 10.",
				Optional:            true,
			},
			"k3s_server_alive_count": schema.Int64Attribute{
				MarkdownDescription: "SSH keepalive count before disconnect for K3s operations. Defaults to 3.",
				Optional:            true,
			},

			// Rook-Ceph attributes
			"rook_ceph_ssh_user": schema.StringAttribute{
				MarkdownDescription: "SSH username for Rook-Ceph operations. Can also be set via ROOK_CEPH_SSH_USER environment variable.",
				Optional:            true,
			},
			"rook_ceph_ssh_password": schema.StringAttribute{
				MarkdownDescription: "SSH password for Rook-Ceph operations. Can also be set via ROOK_CEPH_SSH_PASSWORD environment variable.",
				Optional:            true,
				Sensitive:           true,
			},
			"rook_ceph_connect_timeout": schema.Int64Attribute{
				MarkdownDescription: "SSH connection timeout in seconds for Rook-Ceph operations. Defaults to 30.",
				Optional:            true,
			},
			"rook_ceph_server_alive_interval": schema.Int64Attribute{
				MarkdownDescription: "SSH keepalive interval in seconds for Rook-Ceph operations. Defaults to 10.",
				Optional:            true,
			},
			"rook_ceph_server_alive_count": schema.Int64Attribute{
				MarkdownDescription: "SSH keepalive count before disconnect for Rook-Ceph operations. Defaults to 3.",
				Optional:            true,
			},

			// Edge schemas
			"cm_url": schema.StringAttribute{
				MarkdownDescription: "Central Manager URL for downloading edge bundles",
				Optional:            true,
			},
			"oauth_token": schema.StringAttribute{
				MarkdownDescription: "OAuth token for authenticating with Central Manager",
				Optional:            true,
				Sensitive:           true,
			},
			"platform": schema.StringAttribute{
				MarkdownDescription: "Target platform: k3s, eks, or openshift",
				Optional:            true,
			},
			"ssh_user": schema.StringAttribute{
				MarkdownDescription: "SSH username for connecting to nodes",
				Optional:            true,
			},
			"ssh_password": schema.StringAttribute{
				MarkdownDescription: "SSH password for connecting to nodes",
				Optional:            true,
				Sensitive:           true,
			},
			"ssh_key_path": schema.StringAttribute{
				MarkdownDescription: "Path to SSH private key file",
				Optional:            true,
			},
			"aws_region": schema.StringAttribute{
				MarkdownDescription: "AWS region for EKS cluster",
				Optional:            true,
			},
			"aws_profile": schema.StringAttribute{
				MarkdownDescription: "AWS CLI profile name",
				Optional:            true,
			},
			"aws_access_key": schema.StringAttribute{
				MarkdownDescription: "AWS access key ID",
				Optional:            true,
				Sensitive:           true,
			},
			"aws_secret_key": schema.StringAttribute{
				MarkdownDescription: "AWS secret access key",
				Optional:            true,
				Sensitive:           true,
			},
			"eks_ssh_user": schema.StringAttribute{
				MarkdownDescription: "SSH username for EKS worker nodes",
				Optional:            true,
			},
			"eks_ssh_key_path": schema.StringAttribute{
				MarkdownDescription: "Path to SSH key for EKS worker nodes",
				Optional:            true,
			},
			"eks_ssh_key_passphrase": schema.StringAttribute{
				MarkdownDescription: "Passphrase for the EKS SSH key (if the key is passphrase-protected)",
				Optional:            true,
				Sensitive:           true,
			},
			"eks_hostname_type": schema.StringAttribute{
				MarkdownDescription: "EKS hostname type: private-dns or ip-address",
				Optional:            true,
			},
			// OpenShift native OAuth authentication
			"ocp_server": schema.StringAttribute{
				MarkdownDescription: "OpenShift API server URL (e.g., https://api.cluster.example.com:6443). When set with ocp_username/ocp_password or ocp_token, uses native OAuth instead of kubeconfig.",
				Optional:            true,
			},
			"ocp_username": schema.StringAttribute{
				MarkdownDescription: "OpenShift username for OAuth authentication",
				Optional:            true,
			},
			"ocp_password": schema.StringAttribute{
				MarkdownDescription: "OpenShift password for OAuth authentication",
				Optional:            true,
				Sensitive:           true,
			},
			"ocp_token": schema.StringAttribute{
				MarkdownDescription: "OpenShift OAuth token (alternative to username/password, can be obtained via 'oc whoami -t')",
				Optional:            true,
				Sensitive:           true,
			},
			"ocp_insecure_skip_verify": schema.BoolAttribute{
				MarkdownDescription: "Skip TLS certificate verification for OpenShift API server (default: false)",
				Optional:            true,
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

	// ===== GDP Client Configuration =====
	var gdpClient *gdp.Client
	host := getStringValue(data.Host, "GDP_HOST")
	port := getStringValue(data.Port, "GDP_PORT")
	caCertPath := getStringValue(data.CACertPath, "GDP_CA_CERT_PATH")
	if host != "" && port != "" {
		gdpClient = gdp.NewClient(host, port)
		gdpClient.CACertPath = caCertPath
	}

	// ===== Edge Client Configuration =====
	cmUrl := getStringValue(data.CMUrl, "GDP_CM_URL")
	oauthToken := getStringValue(data.OAuthToken, "GDP_OAUTH_TOKEN")
	platform := getStringValue(data.Platform, "GDP_PLATFORM")
	sshUser := getStringValue(data.SSHUser, "GDP_SSH_USER")
	sshPassword := getStringValue(data.SSHPassword, "GDP_SSH_PASSWORD")
	sshKeyPath := getStringValue(data.SSHKeyPath, "GDP_SSH_KEY_PATH")
	awsRegion := getStringValue(data.AWSRegion, "AWS_REGION")
	awsProfile := getStringValue(data.AWSProfile, "AWS_PROFILE")
	awsAccessKey := getStringValue(data.AWSAccessKey, "AWS_ACCESS_KEY_ID")
	awsSecretKey := getStringValue(data.AWSSecretKey, "AWS_SECRET_ACCESS_KEY")
	eksSSHUser := getStringValue(data.EKSSSHUser, "GDP_EKS_SSH_USER")
	eksSSHKeyPath := getStringValue(data.EKSSSHKeyPath, "GDP_EKS_SSH_KEY_PATH")
	eksSSHKeyPassphrase := getStringValue(data.EKSSSHKeyPassphrase, "GDP_EKS_SSH_KEY_PASSPHRASE")
	eksHostnameType := getStringValue(data.EKSHostnameType, "GDP_EKS_HOSTNAME_TYPE")

	// OpenShift native OAuth authentication
	ocpServer := getStringValue(data.OCPServer, "OCP_SERVER")
	ocpUsername := getStringValue(data.OCPUsername, "OCP_USERNAME")
	ocpPassword := getStringValue(data.OCPPassword, "OCP_PASSWORD")
	ocpToken := getStringValue(data.OCPToken, "OCP_TOKEN")
	ocpInsecureSkipVerify := getBoolValue(data.OCPInsecureSkipVerify, "OCP_INSECURE_SKIP_VERIFY")

	// Default SSH user
	if sshUser == "" {
		sshUser = "root"
	}

	edgeClient := edgeclient.NewClient(edgeclient.Config{
		CMUrl:                 cmUrl,
		OAuthToken:            oauthToken,
		Platform:              platform,
		SSHUser:               sshUser,
		SSHPassword:           sshPassword,
		SSHKeyPath:            sshKeyPath,
		AWSRegion:             awsRegion,
		AWSProfile:            awsProfile,
		AWSAccessKey:          awsAccessKey,
		AWSSecretKey:          awsSecretKey,
		EKSSSHUser:            eksSSHUser,
		EKSSSHKeyPath:         eksSSHKeyPath,
		EKSSSHKeyPassphrase:   eksSSHKeyPassphrase,
		EKSHostnameType:       eksHostnameType,
		OCPServer:             ocpServer,
		OCPUsername:           ocpUsername,
		OCPPassword:           ocpPassword,
		OCPToken:              ocpToken,
		OCPInsecureSkipVerify: ocpInsecureSkipVerify,
	})

	// ===== K3s Client Configuration =====
	k3sSSHUser := getStringValue(data.K3sSSHUser, "K3S_SSH_USER")
	if k3sSSHUser == "" {
		k3sSSHUser = "root"
	}
	k3sSSHPassword := getStringValue(data.K3sSSHPassword, "K3S_SSH_PASSWORD")

	k3sConnectTimeout := int(data.K3sConnectTimeout.ValueInt64())
	if k3sConnectTimeout == 0 {
		k3sConnectTimeout = 30
	}
	k3sServerAliveInterval := int(data.K3sServerAliveInterval.ValueInt64())
	if k3sServerAliveInterval == 0 {
		k3sServerAliveInterval = 10
	}
	k3sServerAliveCount := int(data.K3sServerAliveCount.ValueInt64())
	if k3sServerAliveCount == 0 {
		k3sServerAliveCount = 3
	}

	var k3sClient *k3sclient.Client
	if k3sSSHPassword != "" {
		k3sClient = k3sclient.NewClient(k3sSSHUser, k3sSSHPassword, k3sclient.SSHOptions{
			ConnectTimeout:      k3sConnectTimeout,
			ServerAliveInterval: k3sServerAliveInterval,
			ServerAliveCount:    k3sServerAliveCount,
		})
	}

	// ===== Rook-Ceph Client Configuration =====
	rookCephSSHUser := getStringValue(data.RookCephSSHUser, "ROOK_CEPH_SSH_USER")
	if rookCephSSHUser == "" {
		rookCephSSHUser = "root"
	}
	rookCephSSHPassword := getStringValue(data.RookCephSSHPassword, "ROOK_CEPH_SSH_PASSWORD")

	rookCephConnectTimeout := int(data.RookCephConnectTimeout.ValueInt64())
	if rookCephConnectTimeout == 0 {
		rookCephConnectTimeout = 30
	}
	rookCephServerAliveInterval := int(data.RookCephServerAliveInterval.ValueInt64())
	if rookCephServerAliveInterval == 0 {
		rookCephServerAliveInterval = 10
	}
	rookCephServerAliveCount := int(data.RookCephServerAliveCount.ValueInt64())
	if rookCephServerAliveCount == 0 {
		rookCephServerAliveCount = 3
	}

	var rookCephClient *rookcephclient.Client
	if rookCephSSHPassword != "" {
		rookCephClient = rookcephclient.NewClient(rookCephSSHUser, rookCephSSHPassword, rookcephclient.SSHOptions{
			ConnectTimeout:      rookCephConnectTimeout,
			ServerAliveInterval: rookCephServerAliveInterval,
			ServerAliveCount:    rookCephServerAliveCount,
		})
	}

	// ===== Create Unified Client =====
	unifiedClient := &UnifiedClient{
		GDPClient:      gdpClient,
		EdgeClient:     edgeClient,
		K3sClient:      k3sClient,
		RookCephClient: rookCephClient,
	}

	resp.DataSourceData = unifiedClient
	resp.ResourceData = unifiedClient
	tflog.Info(ctx, "provider configuration configured")
}

func (p *GuardiumDataProtectionProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		// Original GDP resources
		NewImportProfilesResource,
		NewInstallConnectorResource,
		NewRegisterVADatasourceResource,
		NewConfigureVADatasourceResource,
		NewConfigureVANotificationsResource,
		NewAWSSecretsManagerResource,
		// Edge resources
		NewEdgeDeploymentResource,
		NewK3SClusterResource,
		NewRookCephClusterResource,
		// AWS utilities for edge
		NewAWSVPCCleanupResource,
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

// getStringValue returns the value from types.String or environment variable fallback
func getStringValue(v types.String, envKey string) string {
	if !v.IsNull() && !v.IsUnknown() && v.ValueString() != "" {
		return v.ValueString()
	}
	return os.Getenv(envKey)
}

// getBoolValue returns the value from types.Bool or environment variable fallback
func getBoolValue(v types.Bool, envKey string) bool {
	if !v.IsNull() && !v.IsUnknown() {
		return v.ValueBool()
	}
	envVal := os.Getenv(envKey)
	return envVal == "true" || envVal == "1" || envVal == "yes"
}

// Made with Bob
