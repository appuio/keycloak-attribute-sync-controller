package controllers

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"time"

	keycloakv1alpha1 "github.com/appuio/keycloak-attribute-sync-controller/api/v1alpha1"
	"github.com/robfig/cron"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/appuio/keycloak-attribute-sync-controller/internal/pkg/keycloak"
	"github.com/appuio/keycloak-attribute-sync-controller/internal/pkg/sync"
)

// AttributeSyncReconciler reconciles a AttributeSync object
type AttributeSyncReconciler struct {
	client.Client
	Scheme *runtime.Scheme

	KeycloakClientBuilder func(baseUrl, loginRealm, username, password string, tlsConfig *tls.Config) keycloak.Client
}

//+kubebuilder:rbac:groups=keycloak.appuio.io,resources=attributesyncs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=keycloak.appuio.io,resources=attributesyncs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=keycloak.appuio.io,resources=attributesyncs/finalizers,verbs=update

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
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	if !instance.ObjectMeta.DeletionTimestamp.IsZero() {
		// Object is in the process of beeing deleted.
		return ctrl.Result{}, nil
	}

	username, password, err := r.fetchCredentials(ctx, instance.GetCredentialsSecret())
	if err != nil {
		err := fmt.Errorf("failed fetching credentials: %w", err)
		r.setError(ctx, instance, err)
		return ctrl.Result{}, err
	}

	tlsConfig, err := keycloakTLSConfig(ctx, r.Client, instance.GetCaSecret())
	if err != nil {
		err := fmt.Errorf("failed setting up tls config: %w", err)
		r.setError(ctx, instance, err)
		return ctrl.Result{}, err
	}

	client := r.KeycloakClientBuilder(
		instance.Spec.URL,
		instance.GetLoginRealm(),
		username, password,
		tlsConfig,
	)

	syncer := sync.UserSyncer{KeycloakClient: client, K8sClient: r.Client}
	err = syncer.Sync(ctx, instance.Spec.Realm, instance.Spec.Attribute, instance.Spec.TargetLabel, instance.Spec.TargetAnnotation)
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

func (r *AttributeSyncReconciler) fetchCredentials(ctx context.Context, secretRef keycloakv1alpha1.SecretRef) (string, string, error) {
	fmtErr := func(field string) error {
		return fmt.Errorf("missing field `%s` in secret `%s/%s`", field, secretRef.Name, secretRef.Namespace)
	}

	secret := &corev1.Secret{}
	err := r.Client.Get(ctx, types.NamespacedName{Name: secretRef.Name, Namespace: secretRef.Namespace}, secret)
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

func keycloakTLSConfig(ctx context.Context, client client.Client, caSecretRef *keycloakv1alpha1.SecretRef) (*tls.Config, error) {
	const caSecretKey = "ca.crt"
	conf := &tls.Config{}

	if caSecretRef == nil {
		return conf, nil
	}

	caSecret := &corev1.Secret{}
	err := client.Get(ctx, types.NamespacedName{Namespace: caSecretRef.Namespace, Name: caSecretRef.Name}, caSecret)
	if err != nil {
		return nil, fmt.Errorf("error fetching CA secret: %w", err)
	}

	ca, found := caSecret.Data[caSecretKey]
	if !found {
		return nil, fmt.Errorf("found no certificate in '%s/%s' with key '%s'", caSecretRef.Namespace, caSecretRef.Name, caSecretKey)
	}

	if conf.RootCAs == nil {
		conf.RootCAs = x509.NewCertPool()
	}
	conf.RootCAs.AppendCertsFromPEM(ca)

	return conf, nil
}
