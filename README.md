[![CI Status](https://github.com/OSC/k8-ldap-configmap/actions/workflows/test.yaml/badge.svg?branch=main)](https://github.com/OSC/k8-ldap-configmap/actions?query=workflow%3Atest)
[![GitHub release](https://img.shields.io/github/v/release/OSC/k8-ldap-configmap?include_prereleases&sort=semver)](https://github.com/OSC/k8-ldap-configmap/releases/latest)
![GitHub All Releases](https://img.shields.io/github/downloads/OSC/k8-ldap-configmap/total)
![Docker Pulls](https://img.shields.io/docker/pulls/ohiosupercomputer/k8-ldap-configmap)
[![Go Report Card](https://goreportcard.com/badge/github.com/OSC/k8-ldap-configmap?ts=1)](https://goreportcard.com/report/github.com/OSC/k8-ldap-configmap)
[![codecov](https://codecov.io/gh/OSC/k8-ldap-configmap/branch/main/graph/badge.svg)](https://codecov.io/gh/OSC/k8-ldap-configmap)

# k8-ldap-configmap

Kubernetes service that generates Kubernetes ConfigMap resources based on LDAP data.

The purpose of the LDAP data in a ConfigMap is to allow policy engines such as [Kyverno](https://kyverno.io/) to have policies that make use of the LDAP data stored in ConfigMaps.

This service uses predefined mappers to build the data for each ConfigMap.  Current mappers are:

* user-uid - The key is the username and the value is the user UID
* user-gid - The Key is the username and the value is the user GID
* user-groups - The key is the username and the value is JSON string that is array of groups that user is a member of
* user-gids - The key is the username and the value is JSON string that is array of group GIDs that user is a member of (GIDs are strings)

## Kubernetes support

Currently this code is built and tested against Kubernetes 1.21.

## Install

### Install with Helm

Only Helm 3 is supported.

```
helm repo add k8-ldap-configmap https://osc.github.io/k8-ldap-configmap
helm install k8-ldap-configmap k8-ldap-configmap/k8-ldap-configmap -n k8-ldap-configmap --create-namespace \
  --set ldapUrl=ldaps://ldap.example.com:636 \
  --set ldapUserBaseDN=ou=People,dc=example,dc=com \
  --set ldapGroupBaseDN=ou=Groups,dc=example,dc=com
```

See [chart README](charts/k8-ldap-configmap/README.md) for documentation on options.

### Install with YAML

First install the necessary Namespace and RBAC resources.

```
kubectl apply -f https://github.com/OSC/k8-ldap-configmap/releases/latest/download/namespace-rbac.yaml
```

The deployment should be downloaded as adjustments are needed to arguments to at minimum supply values for empty arguments.

```
wget https://github.com/OSC/k8-ldap-configmap/releases/latest/download/deployment.yaml
# Make changes to arguments
kubectl apply -f deployment.yaml
```

## Configuration

The k8-ldap-configmap is intended to be deployed inside a Kubernetes cluster. It can also be run outside the cluster as a service.

For Active Directory it's likely paged searches are required so at minimum the `--ldap-paged-search` flag would be required.

The default filters for searching users and groups (`--ldap-user-filter` and `--ldap-group-filter`) can be overridden for specific mappers.
For example, to override the group filter for `user-gids` mapper: `--mappers-group-filter=user-gids=(objectClass=posixAccount)`.
Each mapper override must be seperated by a comma.

The following flags and environment variables can modify the behavior of the k8-ldap-configmap:

| Flag    | Environment Variable | Description | Default/Required |
|---------|----------------------|-------------|------------------|
| --ldap-url | LDAP_URL | LDAP URL to query, example: `ldap://ldap.example.com:389` | **Required** |
| --ldap-tls | LDAP_TLS | Enable TLS when connecting to LDAP | `false` |
| --no-ldap-tls-verify | LDAP_TLS_VERIFY=false | Disable TLS verification when connecting to LDAP | `true` |
| --ldap-tls-ca-cert | LDAP_TLS_CA_CERT | The contents of TLS CA cert when the certificate needs to be verified and not in global trust store | None |
| --ldap-group-base-dn | LDAP_GROUP_BASE_DN | Base DN of the Groups OU in LDAP | **Required** |
| --ldap-user-base-dn | LDAP_USER_BASE_DN | Base DN of the Users OU in LDAP | **Required** |
| --ldap-bind-dn | LDAP_BIND_DN | Bind DN when connecting to LDAP | None (anonymous binds) |
| --ldap-bind-password | LDAP_BIND_PASSWORD | Bind password when connecting to LDAP | None (anonymous binds) |
| --ldap-group-filter | LDAP_GROUP_FILTER | Group LDAP filter | `(objectClass=posixGroup)` |
| --ldap-user-filter | LDAP_USER_FILTER | User LDAP filter | `(objectClass=posixAccount)` |
| --ldap-paged-search | LDAP_PAGED_SEARCH | Enable paged searches against LDAP | `false` |
| --ldap-paged-search-size | LDAP_PAGED_SEARCH_SIZE | Size of searches when using paged searches | `1000` |
| --ldap-member-scheme | LDAP_MEMBER_SCHEME | How group members are defined, `memberof`, `member` or `memberuid` | `memberof` |
| --ldap-user-attr-map | LDAP_USER_ATTR_MAP | Attribute map for users | `name=uid,uid=uidNumber,gid=gidNumber` |
| --ldap-group-attr-map | LDAP_GROUP_ATTR_MAP | Attribute map for groups | `name=cn,gid=gidNumber` |
| --mappers | MAPPERS | The mappers to run | `user-uid,user-gid` |
| --mappers-group-filter | MAPPERS_GROUP_FILTER | The mapper specific group filters | None (use `--ldap-group-filter`) |
| --mappers-user-filter | MAPPERS_USER_FILTER | The mapper specific user filters | None (use `--ldap-user-filter`) |
| --namespace | NAMESPACE | The namespace to write ConfigMaps to | **Required** |
| --user-prefix | USER_PREFIX | Prefix to add to all username values | None |
| --interval | INTERLVAL | Interval to run LDAP sync to ConfigMaps | `5m`
| --kubeconfig | KUBECONFIG | The path to Kubernetes config, required when run outside Kubernetes |
| --listen-address | LISTEN_ADDRESS=:8080| Address to listen for HTTP requests |
| --no-process-metrics | PROCESS_METRICS=false | Disable metrics about the running processes such as CPU, memory and Go stats |
| --log-level=info | LOG_LEVEL=info | The logging level One of: [debug, info, warn, error] |
| --log-format=logfmt | LOG_FORMAT=logfmt | The logging format, either logfmt or json |
