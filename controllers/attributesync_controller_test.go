package controllers

import (
	"context"

	"github.com/Nerzal/gocloak/v9"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	userv1 "github.com/openshift/api/user/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	keycloakv1alpha1 "github.com/appuio/keycloak-attribute-sync-controller/api/v1alpha1"
	"github.com/appuio/keycloak-attribute-sync-controller/internal/pkg/keycloak"
)

var _ = Describe("AttributeSync controller", func() {
	Context("When having a valid sync configuration", func() {
		const (
			username  = "mytestuser"
			attribute = "example.com/organization"
			value     = "IgniteCyber"
			target    = "example.com/keycloak-organization"
		)

		It("Should sync attributes from keycloak users to user objects", func() {
			ctx := context.Background()

			By("By creating a keycloak user")
			keycloakFakeClient.Users = []*gocloak.User{
				keycloak.UserWithAttribute(username, attribute, value),
			}

			By("By creating an openshift user object")
			ocpUser := &userv1.User{
				TypeMeta: metav1.TypeMeta{APIVersion: "user.openshift.io/v1", Kind: "User"},
				ObjectMeta: metav1.ObjectMeta{
					Name: username,
				},
			}
			Expect(k8sClient.Create(ctx, ocpUser)).Should(Succeed())

			By("By creating a sync config")
			attributeSyncSecret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "sync-organization",
					Namespace: "default",
				},
				Data: map[string][]byte{
					"username": []byte("user"),
					"password": []byte("pw"),
				},
			}
			Expect(k8sClient.Create(ctx, attributeSyncSecret)).Should(Succeed())

			attributeSync := &keycloakv1alpha1.AttributeSync{
				TypeMeta: metav1.TypeMeta{APIVersion: "keycloak.appuio.ch/v1alpha1", Kind: "AttributeSync"},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "sync-organization",
					Namespace: "default",
				},
				Spec: keycloakv1alpha1.AttributeSyncSpec{
					Attribute:         attribute,
					TargetAnnotation:  target,
					CredentialsSecret: &keycloakv1alpha1.SecretRef{Name: "sync-organization", Namespace: "default"},
				},
			}
			Expect(k8sClient.Create(ctx, attributeSync)).Should(Succeed())

			Eventually(func() (string, error) {
				ocpUser := &userv1.User{}
				err := k8sClient.Get(ctx, types.NamespacedName{Namespace: "", Name: username}, ocpUser)
				if err != nil {
					return "", err
				}
				return ocpUser.ObjectMeta.Annotations[target], nil
			}, "10s", "250ms").Should(Equal(value))
		})
	})
})
