package keycloak

import (
	"context"

	"github.com/Nerzal/gocloak/v9"
)

type ClientFactory func(basePath string) Client

type Client interface {
	LoginAdmin(ctx context.Context, username string, password string, realm string) (*gocloak.JWT, error)
	GetUsers(ctx context.Context, accessToken string, realm string, params gocloak.GetUsersParams) ([]*gocloak.User, error)
}

func NewClient(basePath string) Client {
	return gocloak.NewClient(basePath)
}
