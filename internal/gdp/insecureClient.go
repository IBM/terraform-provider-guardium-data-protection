// Copyright (c) IBM Corporation
// SPDX-License-Identifier: Apache-2.0

package gdp

import (
	"context"
	"crypto/tls"
	"net/http"
)

type InsecureClient struct {
	Client Client
}

func (c *Client) NewInsecureClient() *InsecureClient {

	return &InsecureClient{
		Client{
			Host:     c.Host,
			port:     c.port,
			protocol: "https",
		},
	}
}

func (i *InsecureClient) ImportProfilesFromFile(ctx context.Context, accessToken, pathToFile string, updateMode bool) error {
	insecureClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}

	return i.Client.ImportProfilesFromFile(ctx, insecureClient, accessToken, pathToFile, updateMode)
}

func (i *InsecureClient) GenerateAccessToken(ctx context.Context, clientSecret, username, password, clientId string) (string, error) {
	insecureClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}

	otr, err := i.Client.generateAccessToken(ctx, insecureClient, clientSecret, username, password, clientId)
	if err != nil {
		return "", err
	}

	return otr.AccessToken, nil
}

func (i *InsecureClient) BulkInstallConnector(ctx context.Context, accessToken, udcName, gdpMuHost string) error {
	insecureClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}

	return i.Client.BulkInstallConnector(ctx, insecureClient, accessToken, udcName, gdpMuHost)
}

func (i *InsecureClient) CreateAWSSecretsManager(ctx context.Context, accessToken string, config *AWSSecretsManagerConfig) error {
	insecureClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}

	return i.Client.CreateAWSSecretsManager(ctx, insecureClient, accessToken, config)
}

func (i *InsecureClient) GetAWSSecretsManager(ctx context.Context, accessToken string, name string) (*AWSSecretsManagerConfig, error) {
	insecureClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}

	return i.Client.GetAWSSecretsManager(ctx, insecureClient, accessToken, name)
}

func (i *InsecureClient) GetExistingAWSSecretsManagerNames(ctx context.Context, accessToken string) ([]string, error) {
	insecureClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}

	return i.Client.GetExistingAWSSecretsManagerNames(ctx, insecureClient, accessToken)
}

func (i *InsecureClient) UpdateAWSSecretsManager(ctx context.Context, accessToken string, config *AWSSecretsManagerConfig) error {
	insecureClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}

	return i.Client.UpdateAWSSecretsManager(ctx, insecureClient, accessToken, config)
}

func (i *InsecureClient) DeleteAWSSecretsManager(ctx context.Context, accessToken string, name string) error {
	insecureClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}

	return i.Client.DeleteAWSSecretsManager(ctx, insecureClient, accessToken, name)
}

func (i *InsecureClient) RegisterVADataSource(ctx context.Context, accessToken string, payload []byte) error {
	insecureClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}

	return i.Client.RegisterVADataSource(ctx, insecureClient, accessToken, payload)
}

func (i *InsecureClient) ConfigureVADataSource(ctx context.Context, accessToken string, payload []byte) error {
	insecureClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}

	return i.Client.ConfigureVADataSource(ctx, insecureClient, accessToken, payload)
}

func (i *InsecureClient) ConfigureVANotifications(ctx context.Context, accessToken string, payload []byte) error {
	insecureClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}

	return i.Client.ConfigureVANotifications(ctx, insecureClient, accessToken, payload)
}
