package keycloak

import (
	"context"

	"github.com/Nerzal/gocloak/v9"
)

type FakeClient struct {
	token *gocloak.JWT
	Users []*gocloak.User
}

func (f *FakeClient) LoginAdmin(ctx context.Context, username, password, realm string) (*gocloak.JWT, error) {
	f.token = &gocloak.JWT{}

	return f.token, nil
}

func (f *FakeClient) GetUsers(ctx context.Context, accessToken, realm string, params gocloak.GetUsersParams) ([]*gocloak.User, error) {
	return f.Users, nil
}

func UserWithAttribute(username string, attributeKey string, attributeValues ...string) *gocloak.User {
	return &gocloak.User{Username: &username, Attributes: &map[string][]string{attributeKey: attributeValues}}
}
