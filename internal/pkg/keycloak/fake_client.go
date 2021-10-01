package keycloak

import (
	"context"
	"fmt"

	"github.com/Nerzal/gocloak/v9"
	"github.com/go-resty/resty/v2"
)

type FakeClient struct {
	token *gocloak.JWT
	Users []*gocloak.User

	loginError error
}

var _ Client = &FakeClient{}

func (f *FakeClient) LoginAdmin(ctx context.Context, username, password, realm string) (*gocloak.JWT, error) {
	f.token = &gocloak.JWT{}

	return f.token, f.loginError
}

func (f *FakeClient) GetUsers(ctx context.Context, accessToken, realm string, params gocloak.GetUsersParams) ([]*gocloak.User, error) {
	return f.Users, nil
}

func (f *FakeClient) RestyClient() *resty.Client {
	return resty.New()
}

func (f *FakeClient) SetRestyClient(*resty.Client) {}

func (f *FakeClient) FakeClientSetLoginError(err error) {
	f.loginError = err
}

func (f *FakeClient) FakeClientSetUserAttribute(username string, attributeKey string, attributeValues ...string) error {
	for _, user := range f.Users {
		if user.Username == nil || *user.Username != username {
			continue
		}
		if user.Attributes == nil {
			user.Attributes = &map[string][]string{}
		}
		attrs := *user.Attributes
		attrs[attributeKey] = attributeValues
		return nil
	}
	return fmt.Errorf("user '%s' not found", username)
}

func UserWithAttribute(username string, attributeKey string, attributeValues ...string) *gocloak.User {
	return &gocloak.User{Username: &username, Attributes: &map[string][]string{attributeKey: attributeValues}}
}
