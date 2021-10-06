package controllers

import (
	"context"

	keycloakv1alpha1 "github.com/appuio/keycloak-attribute-sync-controller/api/v1alpha1"
	"github.com/redhat-cop/operator-utils/pkg/util/apis"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func (r *AttributeSyncReconciler) setSuccess(ctx context.Context, instance *keycloakv1alpha1.AttributeSync) {
	l := log.FromContext(ctx)

	condition := metav1.Condition{
		Type:               apis.ReconcileSuccess,
		LastTransitionTime: metav1.Now(),
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
