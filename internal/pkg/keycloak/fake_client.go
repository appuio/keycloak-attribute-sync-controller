package keycloak

import (
	"context"
	"fmt"

	"github.com/Nerzal/gocloak/v9"
)

type FakeClient struct {
	Users []*gocloak.User
	err   error
}

var _ Client = &FakeClient{}

func (f *FakeClient) GetUsers(ctx context.Context, realm string, params gocloak.GetUsersParams) ([]*gocloak.User, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.Users, nil
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

func (f *FakeClient) FakeClientSetError(err error) {
	f.err = err
}

func UserWithAttribute(username string, attributeKey string, attributeValues ...string) *gocloak.User {
	return &gocloak.User{Username: &username, Attributes: &map[string][]string{attributeKey: attributeValues}}
}
