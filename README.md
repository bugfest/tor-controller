<p align="center">
  <img height="100" src="https://sr.ht/2mc0.png">
</p>

<h1 align="center">tor-controller</h1>

[![Build multiarch image - latest](https://github.com/bugfest/tor-controller/actions/workflows/main.yml/badge.svg)](https://github.com/bugfest/tor-controller/actions/workflows/main.yml)
[![Build multiarch image - tag](https://github.com/bugfest/tor-controller/actions/workflows/main-tag.yml/badge.svg)](https://github.com/bugfest/tor-controller/actions/workflows/main-tag.yml)
[![Release Charts](https://github.com/bugfest/tor-controller/actions/workflows/release.yml/badge.svg)](https://github.com/bugfest/tor-controller/actions/workflows/release.yml)
[![pages-build-deployment](https://github.com/bugfest/tor-controller/actions/workflows/pages/pages-build-deployment/badge.svg)](https://github.com/bugfest/tor-controller/actions/workflows/pages/pages-build-deployment)

| **NOTICE** |
| --- |
| This project started as an exercise to update `kragniz`'s https://github.com/kragniz/tor-controller version. If you want to migrate to this implementation, update your OnionService manifests |

# Table of Contents
1. [Changes](#changes)
1. [Roadmap](#roadmap)
1. [Install](#install)
1. [How to](#how-to)
   1. [Quickstart with random address](#quickstart-with-random-address)
   1. [Random service names](#random-service-names)
   1. [Bring your own secret](#bring-your-own-secret)
   1. [Using with nginx-ingress](#using-with-nginx-ingress)
   1. [HA Onionbalance Hidden Services](#ha-onionbalance-hidden-services)
   1. [Service Monitors](#service-monitors)
1. [Tor](#tor)
1. [How it works](#how-it-works)
1. [Builds](#builds)
1. [Versions](#versions)
1. [References](#references)
   1. [Tor Documentation](#tor-documentation)
   1. [Utils](#utils)
   1. [Other projects](#other-projects)

Changes
-------

- **(v0.0.0)** Go updated to `1.17`
- **(v0.0.0)** Code ported to kubebuilder version `3`
- **(v0.0.0)** Domain updated moved from `tor.k8s.io` (protected) to `k8s.torproject.org` (see https://github.com/kubernetes/enhancements/pull/1111)
- **(v0.0.0)** Added `OnionBalancedService` type
- **(v0.0.0)** New OnionService version v1alpha2
- **(v0.0.0)** Migrate clientset code to controller-runtime
- **(v0.3.1)** Helm chart
- **(v0.3.2)** MultiArch images. Supported architectures: amd64, arm, arm64
- **(v0.4.0)** Implement `OnionBalancedService` resource (HA Onion Services)
- **(v0.5.0)** Tor & OnionBalance metric exporters. Prometheus ServiceMonitor integration

Changelog: [CHANGELOG](CHANGELOG.md)

Roadmap / TODO
--------------

- Tor daemon management via socket (e.g: config reload)
- Tor proxy resource: expose socks and http k8s's svc
- Tor relays:
  - Non exit: Bridge, Sbowflake, Middle/Guard
  - Exit relay: Tor Exit
- Tor-Istio plugin/extension to route pod egress traffic thru Tor

Install
-------

Using helm (recommended):

    $ helm repo add bugfest https://bugfest.github.io/tor-controller
    $ helm upgrade --install \
      --create-namespace --namespace tor-controller \
      tor-controller bugfest/tor-controller

Install tor-controller directly using the manifest:

    $ kubectl apply -f hack/install.yaml

Resources
---------

| Name | Shortnames | Api Version | Namespaced | Kind |
| ---- | ---------- | ----------- | ---------- | ---- |
|onionservices|onion,os|tor.k8s.torproject.org/v1alpha2|true|OnionService|
|onionbalancedservices|onionha,oha,obs|tor.k8s.torproject.org/v1alpha2|true|OnionBalancedService|
|projectconfigs| | config.k8s.torproject.org/v2|true|ProjectConfig|


**OnionService**: Exposes a set of k8s services using as a Tor Hidden Service. By default it generates a random .onion adress

**OnionBalancedService**: Exposes a set of k8s services using [Onionbalance](https://gitlab.torproject.org/tpo/core/onionbalance.git). It creates multiple backends providing some sort of HA. Users connect to the OnionBalancedService address and the requests are managed by one of the registered backends.

How to
------

Some examples you can use to start using tor-controller in your cluster 

Quickstart with random onion address
------------------------------------

Create some deployment to test against, in this example we'll deploy an echoserver. You can find the definition at [hack/sample/echoserver.yaml](hack/sample/echoserver.yaml):

Apply it:

    $ kubectl apply -f hack/sample/echoserver.yaml

For a fixed address, we need a private key. This should be kept safe, since
someone can impersonate your onion service if it is leaked. Tor-Controller will generate an Onion v3 key-pair for you (stored as a secret), unless it already exists

Create an onion service, [hack/sample/onionservice.yaml](hack/sample/onionservice.yaml), referencing an existing private key is optional:

```yaml
apiVersion: tor.k8s.torproject.org/v1alpha2
kind: OnionService
metadata:
  name: example-onion-service
spec:
  version: 3
  rules:
    - port:
        number: 80
      backend:
        service:
          name: http-app
          port:
            number: 8080
```

Apply it:

    $ kubectl apply -f hack/sample/onionservice.yaml

List active OnionServices:

```bash
$ kubectl get onionservices
NAME                    HOSTNAME                                                         TARGETCLUSTERIP   AGE
example-onion-service   cfoj4552cvq7fbge6k22qmkun3jl37oz273hndr7ktvoahnqg5kdnzqd.onion   10.43.252.41      1m
```

**Note**: you can also the alias `onion` or `os` to interact with these resources. Example: `kubectl get onion`

This service should now be accessable from any tor client,
for example [Tor Browser](https://www.torproject.org/projects/torbrowser.html.en):

Random service names
--------------------

If `spec.privateKeySecret` is not specified, tor-controller will start a service with a random name. The key-pair is stored in the same namespace as the tor-daemon, with the name `ONIONSERVICENAME-tor-secret`

The created secret has the following format:

```yaml
apiVersion: v1
kind: Secret
type: tor.k8s.torproject.org/onion-v3
metadata:
  name: example-tor-secret
data:
  onionAddress: ZWxqZGU2a...
  privateKey: oMLf2tSS2...
  privateKeyFile: PT0gZW...
  publicKey: ItIyeT+kH...
  publicKeyFile: PT0gZWQyNT...
...
```

Bring your own secret
---------------------

Set `spec.privateKeySecret.name` to specify an existing secret. If you don't set `spec.privateKeySecret.key`, the controller expects it to have the following keys:

* `onionAddress` (corresponds to is the `hostname` file usually created by Tor)
* `privateKeyFile` (corresponds to `hs_ed25519_secret_key` file in v3 onion addresses or `private_key` in v2 ones)
* `publicKeyFile` (corresponds to `hs_ed25519_public_key` file in v3 onion addresses or `public_key` in v2 ones)

You can create such a secret with the following command:
```bash
$ kubectl create secret generic my-full-onion-secret \
  --from-file=privateKeyFile=hs_ed25519_secret_key \
  --from-file=publicKeyFile=hs_ed25519_public_key \
  --from-file=onionAddress=hostname
```

Onion Service example referencing a Secret only by name:
```yaml
apiVersion: tor.k8s.torproject.org/v1alpha2
kind: OnionService
metadata:
  name: example-onion-service
spec:
  ...
  privateKeySecret:
    name: my-full-onion-secret
```

If you set `spec.privateKeySecret.key`, the controller expects it to point to a valid `hs_ed25519_secret_key` content.

Secret example:
```bash
$ kubectl create secret generic my-private-onion-secret \
  --from-file=mykeyname=hs_ed25519_secret_key
```

Onion Service example referencing a Secret only by name:
```yaml
apiVersion: tor.k8s.torproject.org/v1alpha2
kind: OnionService
metadata:
  name: example-onion-service
spec:
  ...
  privateKeySecret:
    name: my-private-onion-secret
    key: mykeyname
```

Onion service versions
----------------------

The `spec.version` field specifies which onion protocol to use.
Only v3 is supported. 

tor-controller defaults to using v3 if `spec.version` is not specified.


Using with nginx-ingress
------------------------

tor-controller on its own simply directs TCP traffic to a backend service.
If you want to serve HTTP stuff, you'll probably want to pair it with
nginx-ingress or some other ingress controller.

To do this, first install nginx-ingress normally. Then point an onion service
at the nginx-ingress-controller, for example:

```yaml
apiVersion: tor.k8s.torproject.org/v1alpha2
kind: OnionService
metadata:
  name: example-onion-service
spec:
  version: 3
  rules:
    - port:
        number: 80
      backend:
        service:
          name: http-app
          port:
            number: 8080
```

This can then be used in the same way any other ingress is. You can find a full
example, with a default backend at [hack/sample/full-example.yaml](hack/sample/full-example.yaml)

HA Onionbalance Hidden Services
-------------------------------

(Available since v0.4.0)

Create an onion balanced service, e.g: [hack/sample/onionbalancedservice.yaml](hack/sample/onionbalancedservice.yaml). `spec.replicas` is the number of backends that will be deployed. An additional `onionbalance` pod will be created to act as frontend. The `spec.template.spec` follows the definition of `OnionService` type.

```
apiVersion: tor.k8s.torproject.org/v1alpha2
kind: OnionBalancedService
metadata:
  name: example-onionbalanced-service
spec:
  replicas: 2
  template:
    spec:
      ...
```

Apply it:

    $ kubectl apply -f hack/sample/onionbalancedservice.yaml

List the frontend onion:

```bash
$ kubectl get onionha
NAME                            HOSTNAME                                                         REPLICAS   AGE
example-onionbalanced-service   gyqyiovslcdv3dawfjpewit4vrobf2r4mcmirxqhwrvviv3wd7zn6sqd.onion   2          1m
```

List the backends:

```bash
$ kubectl get onion
NAME                                  HOSTNAME                                                         TARGETCLUSTERIP   AGE
example-onionbalanced-service-obb-1   dpyjx4jv7apmaxy6fl5kbwwhr7sfxmowfi7nydyyuz6npjksmzycimyd.onion   10.43.81.229      1m
example-onionbalanced-service-obb-2   4r4n25aewayyupxby34bckljr5rn7j4xynagvqqgde5xehe4ls7s5qqd.onion   10.43.105.32      1m
```

**Note**: you can also the alias `onionha` or `obs` to interact with OnionBalancedServices resources. Example: `kubectl get onionha`

Service Monitors
----------------

You can get Service Monitors created automatically for `OnionService` and `OnionBalancedService` objects setting `serviceMonitor` to `true`. It will be used by prometheus to scrape metrics.

Examples:

- `OnionService`: [onionservice-monitored.yaml](hack/sample/onionservice-monitored.yaml)
- `OnionBalancedService`: [onionbalancedservice-monitored.yaml](hack/sample/onionbalancedservice-monitored.yaml)

# Tor

Tor is an anonymity network that provides:

- privacy
- enhanced tamperproofing
- freedom from network surveillance
- NAT traversal

tor-controller allows you to create `OnionService` resources in kubernetes.
These services are used similarly to standard kubernetes services, but they
only serve traffic on the tor network (available on `.onion` addresses).

See [this page](https://www.torproject.org/docs/onion-services.html.en) for
more information about onion services.

HA Hidden Services are implemented by [OnionBalance](https://gitlab.torproject.org/tpo/core/onionbalance). Implements round-robin like load balancing on top of Tor onion services. A typical Onionbalance deployment will incorporate one frontend servers and multiple backend instances.` https://onionbalance.readthedocs.io/en/latest/v3/tutorial-v3.html


# How it works

tor-controller creates the following resources for each OnionService:

- tor pod, which contains a tor daemon to serve incoming traffic from the tor
  network, and a management process that watches the kubernetes API and
  generates tor config, signaling the tor daemon when it changes
- rbac rules

Builds
------

| Name                     | Type  | URL                                                         | Comment     |
| -------------------------| ----- | ----------------------------------------------------------- | ----------- |
| helm release             | helm  | https://bugfest.github.io/tor-controller                    |             |
| tor-controller           | image | https://quay.io/repository/bugfest/tor-controller           |             |
| tor-daemon-manager       | image | https://quay.io/repository/bugfest/tor-daemon-manager       | Build requires bugfest/tor |
| tor-onionbalance-manager | image | https://quay.io/repository/bugfest/tor-onionbalance-manager |             |
| tor                      | image | https://quay.io/repository/bugfest/tor                      |             |

Dependencies:
- tor-daemon-manager Dockerfile uses bugfest/tor image (built from source). It is built in a separate project to speed up the compilation: [https://github.com/bugfest/tor-docker](https://github.com/bugfest/tor-docker)

Versions
--------

| Helm Chart version | Tor-Controller version | Tor daemon |
| ----- | ----- | ------- |
| 0.1.0 | 0.3.1 | 0.4.6.8 |
| 0.1.1 | 0.3.2 | 0.4.6.8 |
| 0.1.2 | 0.4.0 | 0.4.6.8 |
| 0.1.3 | 0.5.0 | 0.4.6.10 |

References
----------

## Documentation
- Tor man pages: https://manpages.debian.org/testing/tor/tor.1.en.html
- Onionbalance: https://gitlab.torproject.org/tpo/core/onionbalance
- Onionbalance tutorial: https://onionbalance.readthedocs.io/en/latest/v3/tutorial-v3.html

## Utils
- Helm docs updated with https://github.com/norwoodj/helm-docs

## Other projects
- https://github.com/rdkr/oniongen-go
- https://github.com/ajvb/awesome-tor
