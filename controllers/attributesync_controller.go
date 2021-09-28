/*
Copyright 2021 APPUiO.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"fmt"
	"time"

	"github.com/Nerzal/gocloak/v9"
	keycloakv1alpha1 "github.com/appuio/keycloak-attribute-sync-controller/api/v1alpha1"
	"github.com/go-logr/logr"
	userv1 "github.com/openshift/api/user/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/appuio/keycloak-attribute-sync-controller/internal/pkg/keycloak"
)

// AttributeSyncReconciler reconciles a AttributeSync object
type AttributeSyncReconciler struct {
	client.Client
	Scheme *runtime.Scheme

	KeycloakClientFactory keycloak.ClientFactory
}

//+kubebuilder:rbac:groups=keycloak.appuio.ch,resources=attributesyncs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=keycloak.appuio.ch,resources=attributesyncs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=keycloak.appuio.ch,resources=attributesyncs/finalizers,verbs=update

//+kubebuilder:rbac:groups=user.openshift.io,resources=users,verbs=get;update;patch
//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the AttributeSync object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.8.3/pkg/reconcile
func (r *AttributeSyncReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	instance := &keycloakv1alpha1.AttributeSync{}
	err := r.Client.Get(ctx, req.NamespacedName, instance)
	if err != nil {
		if apierrors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	client := r.KeycloakClientFactory(instance.Spec.URL)

	username, password, err := r.fetchCredentials(ctx, instance.Spec.CredentialsSecret.Name, instance.Spec.CredentialsSecret.Namespace)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed fetching credentials: %w", err)
	}

	token, err := client.LoginAdmin(ctx, username, password, instance.Spec.LoginRealm)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed binding to keycloak: %w", err)
	}

	users, err := client.GetUsers(ctx, token.AccessToken, instance.Spec.Realm, gocloak.GetUsersParams{})
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("error fetching users: %w", err)
	}

	err = r.syncUsers(ctx, logger, users, instance.Spec.Attribute, instance.Spec.TargetLabel, instance.Spec.TargetAnnotation)

	return ctrl.Result{}, err
}

// SetupWithManager sets up the controller with the Manager.
func (r *AttributeSyncReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&keycloakv1alpha1.AttributeSync{}).
		Complete(r)
}

func (r *AttributeSyncReconciler) fetchCredentials(ctx context.Context, secretName, secretNamespace string) (string, string, error) {
	fmtErr := func(field string) error {
		return fmt.Errorf("missing field `%s` in secret `%s/%s`", field, secretName, secretNamespace)
	}

	secret := &corev1.Secret{}
	err := r.Client.Get(ctx, types.NamespacedName{Name: secretName, Namespace: secretNamespace}, secret)
	if err != nil {
		return "", "", err
	}

	username, ok := secret.Data["username"]
	if !ok {
		return "", "", fmtErr("username")
	}

	password, ok := secret.Data["password"]
	if !ok {
		return "", "", fmtErr("password")
	}

	return string(username), string(password), nil
}

func (r *AttributeSyncReconciler) syncUsers(ctx context.Context, l logr.Logger, users []*gocloak.User, attributeKey, targetLabel, targetAnnotation string) error {
	syncTime := time.Now().UTC().Format(time.RFC3339)
	for _, user := range users {
	RETRY:
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

		ocpuser := userv1.User{}
		err := r.Client.Get(ctx, types.NamespacedName{Name: *user.Username}, &ocpuser)
		if err != nil {
			if apierrors.IsNotFound(err) {
				l.V(1).Info("no OCP user object found - skipping")
				continue
			}
			return fmt.Errorf("error fetching user: %w", err)
		}

		if ocpuser.ObjectMeta.Annotations == nil {
			ocpuser.ObjectMeta.Annotations = map[string]string{}
		}
		if targetAnnotation != "" {
			ocpuser.ObjectMeta.Annotations[targetAnnotation] = attribute
		}
		if targetLabel != "" {
			if ocpuser.ObjectMeta.Labels == nil {
				ocpuser.ObjectMeta.Labels = map[string]string{}
			}
			ocpuser.ObjectMeta.Labels[targetLabel] = attribute
		}

		ocpuser.ObjectMeta.Annotations["TODO"] = syncTime
		if err := r.Client.Update(ctx, &ocpuser); err != nil {
			if apierrors.IsConflict(err) {
				// The User has been updated since we read it.
				goto RETRY
			}
			if apierrors.IsNotFound(err) {
				continue
			}
			return fmt.Errorf("unable to update user: %w", err)
		}
	}

	return nil
}
