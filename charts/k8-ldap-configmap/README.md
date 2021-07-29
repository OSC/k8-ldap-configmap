# k8-ldap-configmap

## Install

```
helm repo add k8-ldap-configmap https://osc.github.io/k8-ldap-configmap
helm install k8-ldap-configmap k8-ldap-configmap/k8-ldap-configmap -n k8-ldap-configmap --create-namespace \
  --set ldapUrl=ldaps://ldap.example.com:636 \
  --set ldapUserBaseDN=ou=People,dc=example,dc=com \
  --set ldapGroupBaseDN=ou=Groups,dc=example,dc=com
```

## Configuration

| Parameter | Description | Default |
|-----------|-------------|---------|
| ldapUrl | The LDAP URL | **Required**|
| ldapTLS | Enable TLS for the LDAP connection | `false` |
| ldapTLSVerify | Enable TLS verification | `true` |
| ldapTLSCACert | The contents of the TLS CA cert to verify TLS connections | `nil` |
| ldapGroupBaseDN | The base DN to search for groups | **Required** |
| ldapUserBaseDN | The base DN to search for users | **Required** |
| ldapBindDN | The bind DN for authenticated binds with LDAP | `nil` |
| ldapBindPassword | The bind password for authenticated binds with LDAP | `nil` |
| ldapGroupFilter | The search filter for groups | `(objectClass=posixGroup)` |
| ldapUserFilter | The search filter for users | `(objectClass=posixAccount)` |
| ldapMemberScheme | The method to determine group membership | `memberof` |
| ldapUserAttrMap | The user attribute map | `name=uid,uid=uidNumber,gid=gidNumber` |
|ldapGroupAttrMap | The group attribute map | `name=cn,gid=gidNumber` |
| mappers | The mappers to enable | `user-uid,user-gid` |
| mappersUserFilter | Mapper specific user filter | `[]` |
| mappersGroupFilter | Mapper specific group filter | `[]` |
| userPrefix | The username prefix when saving usernames to ConfigMaps | `nil` |
| interval | The interval to sync LDAP to ConfigMaps | `5m` |
| namespaceConfigMap | The namespace of generated ConfigMaps | The namespace used to deploy chart |
| extraArgs | Extra arguments | `[]` |
| image.repository | Image repository | `docker.io/ohiosupercomputer/k8-ldap-configmap` |
| image.pullPolicy | Image pull policy | `IfNotPresent` |
| image.tag: | Image tag | The application version of chart |
| imagePullSecrets | Image pull secrets | `[]` |
| nameOverride | Override the name of the chart | `nil` |
| fullnameOverride | Override full name used for chart resources | `nil` |
| namespace | Override the namespace used | Chart release namespace |
| rbac.create | create cluster roles and role bindings and service account | `true` |
| rbac.serviceAccount.create | create the service account | `true` |
| rbac.serviceAccount.annotations | Service account annotations | `{}` |
| rbac.serviceAccount.name | Service account name | Full name of chart |
| podAnnotations | Pod annotations | `{}` |
| podSecurityContext | Pod security context | `{}` |
| securityContext | Security context of deployment | `{ allowPrivilegeEscalation: false, capabilities: {drop [all]}, privileged: false, readOnlyRootFilesystem: true, runAsNonRoot: true, runAsUser: 65534 }` |
| service.create | Sets if the service should be created | `true` |
| service.type | Service type | `ClusterIP` |
| service.port | Service port | `8080` |
| service.annotations | Service annotations | `{prometheus.io/scrape: 'true', prometheus.io/path: /metrics}` |
| resources | Resources for deployment | `{limits: {memory: 128Mi}, requests: {cpu: 100m, memory: 50mi}}` |
| nodeSelector | Deployment nodeSelector | `{}` |
| tolerations | Deployment tolerations | `[]` |
| affinity | Deployment affinity | `{}` |
