package v1alpha1_test

import (
	"testing"

	"github.com/appuio/keycloak-attribute-sync-controller/api/v1alpha1"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestAttributeSync_GetCaSecret(t *testing.T) {
	nsName := "myapp"
	subject := &v1alpha1.AttributeSync{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: nsName,
		},
	}
	t.Run("returns nil if empty", func(t *testing.T) {
		assert.Nil(t, subject.GetCaSecret())
	})
	t.Run("returns default if reference is set and namespace is empty", func(t *testing.T) {
		subject.Spec.CaSecret = &v1alpha1.SecretRef{}
		assert.Equal(t, nsName, subject.GetCaSecret().Namespace)
	})
	t.Run("returns namespace if set", func(t *testing.T) {
		subject.Spec.CaSecret.Namespace = "override"
		assert.Equal(t, "override", subject.GetCaSecret().Namespace)
	})
}

func TestAttributeSync_GetCredentialsSecret(t *testing.T) {
	nsName := "myapp"
	subject := &v1alpha1.AttributeSync{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: nsName,
		},
	}
	t.Run("returns default if namespace is empty", func(t *testing.T) {
		assert.Equal(t, nsName, subject.GetCredentialsSecret().Namespace)
	})
	t.Run("returns namespace if set", func(t *testing.T) {
		subject.Spec.CredentialsSecret.Namespace = "override"
		assert.Equal(t, "override", subject.GetCredentialsSecret().Namespace)
	})
}

func TestAttributeSync_GetLoginRealm(t *testing.T) {
	subject := &v1alpha1.AttributeSync{}
	t.Run("returns default (`master`) if login realm is empty", func(t *testing.T) {
		assert.Equal(t, "master", subject.GetLoginRealm())
	})
	t.Run("returns login realm if set", func(t *testing.T) {
		subject.Spec.LoginRealm = "override"
		assert.Equal(t, "override", subject.GetLoginRealm())
	})
}
