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
	userv1 "github.com/openshift/api/user/v1"
	"github.com/redhat-cop/operator-utils/pkg/util/apis"
	"github.com/robfig/cron"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

//+kubebuilder:rbac:groups=user.openshift.io,resources=users,verbs=get;list;watch;update;patch
//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *AttributeSyncReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	l := log.FromContext(ctx)
	l.Info("Reconciling")

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
		err := fmt.Errorf("failed fetching credentials: %w", err)
		r.setError(ctx, instance, err)
		return ctrl.Result{}, err
	}

	token, err := client.LoginAdmin(ctx, username, password, instance.GetLoginRealm())
	if err != nil {
		err := fmt.Errorf("failed binding to keycloak: %w", err)
		r.setError(ctx, instance, err)
		return ctrl.Result{}, err
	}

	users, err := client.GetUsers(ctx, token.AccessToken, instance.Spec.Realm, gocloak.GetUsersParams{})
	if err != nil {
		err := fmt.Errorf("error fetching users: %w", err)
		r.setError(ctx, instance, err)
		return ctrl.Result{}, err
	}

	err = r.syncUsers(ctx, users, instance.Spec.Attribute, instance.Spec.TargetLabel, instance.Spec.TargetAnnotation)
	if err != nil {
		err := fmt.Errorf("error syncing users: %w", err)
		r.setError(ctx, instance, err)
		return ctrl.Result{}, err
	}

	r.setSuccess(ctx, instance)

	if instance.Spec.Schedule != "" {
		// TODO(bastjan): Should have a validating webhook. It's currently not really
		//                possible to use kustomize in commodore so it would be quite
		//                complicated to implement in the component.
		sched, err := cron.ParseStandard(instance.Spec.Schedule)
		if err != nil {
			l.Error(err, "Error parsing reconciling schedule")
			return ctrl.Result{}, err
		}

		currentTime := time.Now()
		nextScheduledTime := sched.Next(currentTime)
		return ctrl.Result{RequeueAfter: nextScheduledTime.Sub(currentTime)}, nil
	}

	return ctrl.Result{}, nil
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

func (r *AttributeSyncReconciler) syncUsers(ctx context.Context, users []*gocloak.User, attributeKey, targetLabel, targetAnnotation string) error {
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

		err := r.setAttributeOnUser(ctx, types.NamespacedName{Name: *user.Username}, attribute, targetLabel, targetAnnotation)
		if err != nil {
			return err
		}
		syncedCount++
	}

	l.Info("Synced users", "synced", syncedCount, "skipped", len(users)-syncedCount)
	return nil
}

func (r *AttributeSyncReconciler) setAttributeOnUser(ctx context.Context, key types.NamespacedName, attribute, targetLabel, targetAnnotation string) error {
	l := log.FromContext(ctx)

RETRY_ON_CONFLICT:
	ocpuser := userv1.User{}
	err := r.Client.Get(ctx, key, &ocpuser)
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
	metaSetAnnotation(&ocpuser.ObjectMeta, "attributesync.keycloak.appuio.ch/sync-time", time.Now().Format(time.RFC3339Nano))

	if err := r.Client.Update(ctx, &ocpuser); err != nil {
		if apierrors.IsConflict(err) {
			// The User has been updated since we read it.
			goto RETRY_ON_CONFLICT
		}
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

func (r *AttributeSyncReconciler) setSuccess(ctx context.Context, instance *keycloakv1alpha1.AttributeSync) {
	l := log.FromContext(ctx)

	successTime := metav1.Now()
	instance.Status.LastSyncSuccessTime = &successTime
	condition := metav1.Condition{
		Type:               apis.ReconcileSuccess,
		LastTransitionTime: successTime,
		ObservedGeneration: instance.GetGeneration(),
		Reason:             apis.ReconcileSuccessReason,
		Status:             metav1.ConditionTrue,
	}
	instance.SetConditions(apis.AddOrReplaceCondition(condition, instance.GetConditions()))
	err := r.Client.Status().Update(ctx, instance)
	if err != nil {
		l.Error(err, "unable to update status")
	}
}

func (r *AttributeSyncReconciler) setError(ctx context.Context, instance *keycloakv1alpha1.AttributeSync, reason error) {
	l := log.FromContext(ctx)

	condition := metav1.Condition{
		Type:               apis.ReconcileError,
		LastTransitionTime: metav1.Now(),
		ObservedGeneration: instance.GetGeneration(),
		Message:            reason.Error(),
		Reason:             apis.ReconcileErrorReason,
		Status:             metav1.ConditionTrue,
	}
	instance.SetConditions(apis.AddOrReplaceCondition(condition, instance.GetConditions()))
	err := r.Client.Status().Update(ctx, instance)
	if err != nil {
		l.Error(err, "unable to update status")
	}
}
