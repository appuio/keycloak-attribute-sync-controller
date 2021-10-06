# keycloak-attribute-sync-controller

Kubernetes Operator to sync Keycloak attributes to Openshift user objects.

## Installation

The controller can be installed using `kubectl`.

```shell
kubectl apply -k config/default
```

## Usage

User Attributes stored within Keycloak can be synchronized into OpenShift.
The following table describes the set of configuration options for the sync:

| Name                | Description                                                                                                     | Defaults | Required |
| ------------------- | --------------------------------------------------------------------------------------------------------------- | -------- | -------- |
| `caSecret`          | Reference to a secret containing a SSL certificate to use for communication. The CA must have the key `ca.crt`. |          | No       |
| `credentialsSecret` | Reference to a secret containing authentication details (See below)                                             |          | Yes      |
| `loginRealm`        | Realm to authenticate against                                                                                   | `master` | No       |
| `realm`             | Realm to synchronize                                                                                            |          | Yes      |
| `attribute`         | The attribute to sync to the user object                                                                        |          | Yes      |
| `targetAnnotation`  | The annotation to sync the attribute to                                                                         |          | No       |
| `targetLabel`       | The label to sync the attribute to                                                                              |          | No       |

The following is an example of a minimal configuration that can be applied to integrate with a Keycloak provider:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: keycloack-read-users-secrets
type: Opaque
data:
  username: ...
  password: ...
---
apiVersion: keycloak.appuio.io/v1alpha1
kind: AttributeSync
metadata:
  name: sync-special-attribute
spec:
  url: https://keycloak.example.com/
  realm: example
  loginRealm: master
  credentialsSecret:
    name: keycloack-read-users-secrets
    namespace: ...
  attribute: example.com/special-attribute
  targetAnnotation: example.com/special-attribute
  schedule: "@every 5m"
```

### Authenticating to Keycloak

A user with permissions to query for Keycloak groups must be available.
The following permissions must be associated to the user:

* Password must be set (Temporary option unselected) on the _Credentials_ tab
* On the _Role Mappings_ tab, select _master-realm_ or _realm-management_ next to the _Client Roles_ dropdown and then select **query-users** and **view-users**.

A secret must be created in the same namespace that contains the `AttributeSync` resource.
It must contain the following keys for the user previously created:

* `username` - Username for authenticating with Keycloak
* `password` - Password for authenticating with Keycloak

The secret can be created by executing the following command:

```shell
oc create secret generic keycloak-attribute-sync --from-literal=username=<username> --from-literal=password=<password>
```

### Scheduled Execution

A cron style expression can be specified for which a synchronization event will occur.
The following specifies that a synchronization should occur nightly at 3AM

```shell
apiVersion: keycloak.appuio.io/v1alpha1
kind: AttributeSync
metadata:
  name: sync-default-org
spec:
  schedule: "0 3 * * *"
```

If a schedule is not provided, synchronization will occur only when the object is reconciled by the platform.

## Limitations

- Only the first Keycloak attribute under the given key is used.
- The key to look up the OCP user object is the Keycloak field `Username`. This is currently hardcoded.
