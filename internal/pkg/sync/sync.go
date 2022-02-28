package sync

import (
	"context"
	"fmt"
	"time"

	userv1 "github.com/openshift/api/user/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/Nerzal/gocloak/v9"
	"github.com/appuio/keycloak-attribute-sync-controller/internal/pkg/keycloak"
)

type UserSyncer struct {
	KeycloakClient keycloak.Client
	K8sClient      client.Client
}

func (u *UserSyncer) Sync(ctx context.Context, realm, attribute, targetLabel, targetAnnotation string) error {
	users, err := u.KeycloakClient.GetUsers(ctx, realm, gocloak.GetUsersParams{
		Max: gocloak.IntP(-1),
	})
	if err != nil {
		return fmt.Errorf("error fetching users: %w", err)
	}

	err = u.syncUsers(ctx, users, attribute, targetLabel, targetAnnotation)
	if err != nil {
		return fmt.Errorf("error syncing users: %w", err)
	}
	return nil
}

func (u *UserSyncer) syncUsers(ctx context.Context, users []*gocloak.User, attributeKey, targetLabel, targetAnnotation string) error {
	l := log.FromContext(ctx)
	l.Info("Syncing users", "count", len(users))
	syncedCount := 0

	for _, user := range users {
		l := l.WithValues("userid", user.ID, "username", user.Username)
		if user.Attributes == nil {
			l.V(1).Info("user has no attributes - skipping")
			continue
		}
		attributes, ok := (*user.Attributes)[attributeKey]
		if !ok || len(attributes) < 1 {
			l.V(1).Info("user has no attribute - skipping", "attribute", attributeKey)
			continue
		}
		attribute := attributes[0]

		err := u.setAttributeOnUser(ctx, types.NamespacedName{Name: *user.Username}, attribute, targetLabel, targetAnnotation)
		if err != nil {
			return err
		}
		syncedCount++
	}

	l.Info("Synced users", "synced", syncedCount, "skipped", len(users)-syncedCount)
	return nil
}

func (u *UserSyncer) setAttributeOnUser(ctx context.Context, key types.NamespacedName, attribute, targetLabel, targetAnnotation string) error {
	l := log.FromContext(ctx)

	ocpuser := userv1.User{}
	err := u.K8sClient.Get(ctx, key, &ocpuser)
	if err != nil {
		if apierrors.IsNotFound(err) {
			l.V(1).Info("no OCP user object found - skipping")
			return nil
		}
		return fmt.Errorf("error fetching user: %w", err)
	}

	if targetAnnotation != "" {
		metaSetAnnotation(&ocpuser.ObjectMeta, targetAnnotation, attribute)
	}
	if targetLabel != "" {
		metaSetLabel(&ocpuser.ObjectMeta, targetLabel, attribute)
	}
	metaSetAnnotation(&ocpuser.ObjectMeta, "attributesync.keycloak.appuio.io/sync-time", time.Now().Format(time.RFC3339Nano))

	if err := u.K8sClient.Update(ctx, &ocpuser); err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("unable to update user: %w", err)
	}

	return nil
}

func metaSetAnnotation(meta *metav1.ObjectMeta, key, value string) {
	if meta.Annotations == nil {
		meta.Annotations = map[string]string{}
	}
	meta.Annotations[key] = value
}

func metaSetLabel(meta *metav1.ObjectMeta, key, value string) {
	if meta.Labels == nil {
		meta.Labels = map[string]string{}
	}
	meta.Labels[key] = value
}
