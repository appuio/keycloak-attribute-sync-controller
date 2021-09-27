package keycloak

import (
	"context"

	"github.com/Nerzal/gocloak/v9"
	"github.com/go-resty/resty/v2"
)

type ClientFactory func(basePath string) Client

type Client interface {
	// RestyClient returns a resty client that gocloak uses
	RestyClient() *resty.Client
	// Sets the resty Client that gocloak uses
	SetRestyClient(restyClient *resty.Client)

	LoginAdmin(ctx context.Context, username string, password string, realm string) (*gocloak.JWT, error)
	GetUsers(ctx context.Context, accessToken string, realm string, params gocloak.GetUsersParams) ([]*gocloak.User, error)
}

func NewClient(basePath string) Client {
	return gocloak.NewClient(basePath)
}
