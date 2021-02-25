module github.com/OSC/k8-ldap-configmap

go 1.15

require (
	github.com/go-kit/kit v0.10.0
	github.com/go-ldap/ldap/v3 v3.2.4
	github.com/lor00x/goldap v0.0.0-20180618054307-a546dffdd1a3
	github.com/prometheus/client_golang v1.9.0
	github.com/prometheus/common v0.17.0
	github.com/vjeantet/ldapserver v1.0.1
	golang.org/x/sync v0.0.0-20190911185100-cd5d95a43a6e
	gopkg.in/alecthomas/kingpin.v2 v2.2.6
	k8s.io/api v0.19.8
	k8s.io/apimachinery v0.19.8
	k8s.io/client-go v0.19.8
)
