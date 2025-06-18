## v0.12.0 / 2025-06-18

* Update to Go 1.24.4 and update dependencies (#32)

## v0.11.0 / 2025-01-09

* Major updates (#28)
  ** Build using Go 1.23
  ** Update all dependencies including Kubernetes to 1.29.12
  ** Replace go-kit logging with slog from promslog
  ** Update all Github Actions to latest versions

## v0.10.0 / 2023-06-30

* Switch container images to use quay.io (#24)
* Bump golang.org/x/net from 0.5.0 to 0.7.0 (#25)

## v0.9.0 / 2023-06-29

* Update to Go 1.20 (#23)

## v0.8.0 / 2023-02-03

* Update to Go 1.19 and update Go module dependencies (#22)

## v0.7.1 / 2023-01-31

* Fix group mappers for member and memberuid (#21)

## v0.7.0 / 2022-02-28

* Improved pod security

## v0.6.1 / 2021-08-20

* Fix run duration metrics to be more accurate

## v0.6.0 / 2021-07-29

* Allow mapper specific user and group LDAP filters
* Update Kubernetes client-go to v0.21.3 (Kubernetes 1.21) and Go module dependencies

## v0.5.1 / 2021-07-28

* Ensure user-gids mapper values are sorted for each user

## v0.5.0 / 2021-07-28

* Add user-gids mapper

## v0.4.2 / 2021-07-03

* Make memberof user-groups mapping ignore case

## v0.4.1 / 2021-04-16

* Fix create RBAC

## v0.4.0 / 2021-04-14

* Update Kubernetes Go dependencies to 0.20.5 (Kubernetes 1.20)
* Upgrade to Go 1.16

## v0.3.0 / 2021-03-12

* Add last run metric

## v0.2.0 / 2021-03-12

* Define mappers in Helm using array
* Restrict RBAC to resourceNames for ConfigMaps

## v0.1.0 / 2021-03-12

* Initial release
