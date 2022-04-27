package keycloak

import (
	"context"
	"crypto/tls"
	"fmt"

	"github.com/Nerzal/gocloak/v9"
)

type Client interface {
	GetUsers(ctx context.Context, realm string, params gocloak.GetUsersParams) ([]*gocloak.User, error)
}

type gocloakClient struct {
	client gocloak.GoCloak

	loginRealm         string
	username, password string
}

func NewClient(baseUrl, loginRealm, username, password string, tlsConfig *tls.Config) Client {
	client := gocloak.NewClient(baseUrl)
	client.SetRestyClient(client.RestyClient().SetTLSClientConfig(tlsConfig))

	return &gocloakClient{
		client: client,

		loginRealm: loginRealm,
		username:   username,
		password:   password,
	}
}

func (g *gocloakClient) GetUsers(ctx context.Context, realm string, params gocloak.GetUsersParams) ([]*gocloak.User, error) {
	token, err := g.client.LoginAdmin(ctx, g.username, g.password, g.loginRealm)
	if err != nil {
		return nil, fmt.Errorf("failed binding to keycloak: %w", err)
	}
	// `admin-cli` is the magic client used when authenticating to the admin API
	defer g.client.LogoutPublicClient(ctx, "admin-cli", g.loginRealm, token.AccessToken, token.RefreshToken)

	return g.client.GetUsers(ctx, token.AccessToken, realm, params)
}
