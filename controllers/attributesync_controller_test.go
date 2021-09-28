package controllers

import (
	"context"
	"fmt"
	"time"

	"github.com/Nerzal/gocloak/v9"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	userv1 "github.com/openshift/api/user/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	keycloakv1alpha1 "github.com/appuio/keycloak-attribute-sync-controller/api/v1alpha1"
	"github.com/appuio/keycloak-attribute-sync-controller/internal/pkg/keycloak"
)

var _ = Describe("AttributeSync controller", func() {
	const (
		username  = "mytestuser"
		attribute = "example.com/organization"
		value     = "IgniteCyber"
		target    = "example.com/keycloak-organization"
	)

	Context("When having a environment with matching OCP and Keycloak users", func() {
		BeforeEach(func() {
			ctx := context.Background()

			By("By having keycloak users")
			keycloakFakeClient.Users = []*gocloak.User{
				keycloak.UserWithAttribute(username, attribute, value),
				keycloak.UserWithAttribute("second-user", attribute, "SuperCyberBlockchainAI"),
				{Username: stringPtr("nil-attributes")},
				{Username: stringPtr("no-attributes"), Attributes: &map[string][]string{}},
			}

			By("By creating an openshift user object")
			ocpUser := &userv1.User{
				ObjectMeta: metav1.ObjectMeta{
					Name: username,
				},
			}
			Expect(k8sClient.Create(ctx, ocpUser)).Should(Succeed())

			By("By creating secret with keycloak credentials")
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
		})

		AfterEach(func() {
			ctx := context.Background()

			k8sClient.DeleteAllOf(ctx, &keycloakv1alpha1.AttributeSync{}, client.InNamespace("default"))
			k8sClient.DeleteAllOf(ctx, &corev1.Secret{}, client.InNamespace("default"))
			k8sClient.DeleteAllOf(ctx, &userv1.User{})
		})

		It("It should sync attributes from keycloak users to user annotations", func() {
			ctx := context.Background()

			By("By creating a sync config with target annotation")
			reconcileTime := time.Now()
			attributeSync := &keycloakv1alpha1.AttributeSync{
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

			By("By querying user annotations")
			Eventually(lookupAnnotationOnUser(ctx, username, target), "10s", "250ms").Should(Equal(value))
			Eventually(lookupAnnotationOnUser(ctx, username, "attributesync.keycloak.appuio.ch/sync-time"), "10s", "250ms").Should(
				WithTransform(mustParseRFC3339, BeTemporally(">=", reconcileTime.Truncate(time.Second))),
			)
		})

		It("It should sync attributes from keycloak users to user labels", func() {
			ctx := context.Background()

			By("By creating a sync config with target label")
			reconcileTime := time.Now()
			attributeSyncLabelTarget := &keycloakv1alpha1.AttributeSync{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "sync-organization-label",
					Namespace: "default",
				},
				Spec: keycloakv1alpha1.AttributeSyncSpec{
					Attribute:         attribute,
					TargetLabel:       target,
					CredentialsSecret: &keycloakv1alpha1.SecretRef{Name: "sync-organization", Namespace: "default"},
				},
			}
			Expect(k8sClient.Create(ctx, attributeSyncLabelTarget)).Should(Succeed())

			By("By querying user annotations")
			Eventually(lookupLabelOnUser(ctx, username, target), "10s", "250ms").Should(Equal(value))
			Eventually(lookupAnnotationOnUser(ctx, username, "attributesync.keycloak.appuio.ch/sync-time"), "10s", "250ms").Should(
				WithTransform(mustParseRFC3339, BeTemporally(">=", reconcileTime.Truncate(time.Second))),
			)
		})

		When("When setting a schedule", func() {
			It("It should sync periodically", func() {
				ctx := context.Background()

				By("By creating a sync config with target annotation")
				attributeSync := &keycloakv1alpha1.AttributeSync{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "sync-organization",
						Namespace: "default",
					},
					Spec: keycloakv1alpha1.AttributeSyncSpec{
						Attribute:         attribute,
						TargetAnnotation:  target,
						Schedule:          "@every 1s",
						CredentialsSecret: &keycloakv1alpha1.SecretRef{Name: "sync-organization", Namespace: "default"},
					},
				}
				Expect(k8sClient.Create(ctx, attributeSync)).Should(Succeed())
				Eventually(lookupAnnotationOnUser(ctx, username, target), "10s", "250ms").Should(Equal(value))

				updatedValue := "UpdatedOrganization"
				Expect(keycloakFakeClient.FakeClientSetUserAttribute(username, attribute, updatedValue)).Should(Succeed())
				Eventually(lookupAnnotationOnUser(ctx, username, target), "10s", "250ms").Should(Equal(updatedValue))
			})
		})
	})
})

func lookupAnnotationOnUser(ctx context.Context, username, annotation string) func() (string, error) {
	return func() (string, error) {
		ocpUser := &userv1.User{}
		err := k8sClient.Get(ctx, types.NamespacedName{Namespace: "", Name: username}, ocpUser)
		if err != nil {
			return "", err
		}
		return ocpUser.ObjectMeta.Annotations[annotation], nil
	}
}

func lookupLabelOnUser(ctx context.Context, username, label string) func() (string, error) {
	return func() (string, error) {
		ocpUser := &userv1.User{}
		err := k8sClient.Get(ctx, types.NamespacedName{Namespace: "", Name: username}, ocpUser)
		if err != nil {
			return "", err
		}
		return ocpUser.ObjectMeta.Labels[label], nil
	}
}

func mustParseRFC3339(r string) time.Time {
	t, err := time.Parse(time.RFC3339, r)
	if err != nil {
		panic(fmt.Errorf("could not parse time: %w", err))
	}
	return t
}

func stringPtr(s string) *string {
	return &s
}
