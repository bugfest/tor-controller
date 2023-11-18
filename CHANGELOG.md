# Changelog

All notable changes to this project will be documented in this file.

## [0.10.0] - 2023-11-18

### Bug Fixes

- Fix auth client dir permissions

## [0.10.0-rc.2] - 2023-11-13

### Bug Fixes

- Fix tag filter

Signed-off-by: Bug Fest <52962234+bugfest@users.noreply.github.com>

- Fix tag filter II

Signed-off-by: Bug Fest <52962234+bugfest@users.noreply.github.com>

- Fix tag filter III

Signed-off-by: Bug Fest <52962234+bugfest@users.noreply.github.com>

- Fix tag filter IV

Signed-off-by: Bug Fest <52962234+bugfest@users.noreply.github.com>


### Generic

- Rc.2 build

Signed-off-by: Bug Fest <52962234+bugfest@users.noreply.github.com>


## [tor-controller-0.1.15-rc.1] - 2023-11-13

### Bug Fixes

- Avoid failure when /app already exists ([#65](https://github.com/bugfest/tor-controller/issues/65))
- Tor instances missing volume/volumeMounts and default DataDirectory config ([#68](https://github.com/bugfest/tor-controller/issues/68))

### Generic

- Update Onionbalance URL ([#61](https://github.com/bugfest/tor-controller/issues/61))


- Update tor version to 0.4.8.7 ([#63](https://github.com/bugfest/tor-controller/issues/63))

* Update latest tor version - 0.4.8.7
- Tor 0.4.8.8, minor fixes; consistenecy ([#66](https://github.com/bugfest/tor-controller/issues/66))

* update: remove redundant config, tor version 0.4.8.8, Dockerfile, minor fixes, security settings, docs
* fix: permissions on folder
- Release/0.1.15-rc.1

Signed-off-by: Bug Fest <52962234+bugfest@users.noreply.github.com>

- Add release branch filter for helm chart release

Signed-off-by: Bug Fest <52962234+bugfest@users.noreply.github.com>


## [tor-controller-0.1.14] - 2023-08-28

### Bug Fixes

- Invalid selector for OnionBalancedService #58
- Fixed python version breaks tor-onionbalance-manager dependencies install

### Generic

- Helm chart release 0.1.14

- [ci-skip] update changelog

- [ci-skip] forced chart re-release

- [ci-skip] forced chart re-release 0.9.2/0.1.14

- [ci-skip] Add workflow_dispatch to helm chart release


## [tor-controller-0.1.13] - 2023-07-27

### Bug Fixes

- Fix: limit CRD description fields to 80 chars ([#57](https://github.com/bugfest/tor-controller/issues/57))
fix: tor-controller manager path moved to /app/manager
fix: README typos


## [tor-controller-0.1.12] - 2023-07-19

### Documentation

- Update helm chart and docs

### Features

- Implements/fixes #56 - ExtraConfig implementation for onionService type

### Generic

- Linting, formatting and error wrappers ([#45](https://github.com/bugfest/tor-controller/issues/45))

* small changes
* fix logger.Info() usage
* code cleanup
* reduce fmt usage

---------

Signed-off-by: Aleksey Sviridkin <a.sviridkin@sdventures.com>
- Update onibalancer image ([#49](https://github.com/bugfest/tor-controller/issues/49))

* update onionbalancer image
* remove debug info from binary
* bump go version 1.20

---------

Signed-off-by: Aleksey Sviridkin <a.sviridkin@sdventures.com>
- Make improvements to containers ([#50](https://github.com/bugfest/tor-controller/issues/50))

## [tor-controller-0.1.11] - 2023-03-14

### Generic

- Add option to use bridges #38 ([#39](https://github.com/bugfest/tor-controller/issues/39))

Tor daemon and manager bumped to 0.4.7.13 (including obfs4proxy binary)
- Release 0.9.0

* [FEATURE] Add option to use bridges #38
* [FEATURE] Upgrade Tor daemon to 0.4.7.x enhancement #40
* [FEATURE] Controller deployment automatic rollout on chart upgrade #41
* [DOC] Update instructions to use bridges and custom Tor daemon configs

## [tor-controller-0.1.10] - 2023-02-05

### Generic

- Implements #33 - Namespaced Support ([#34](https://github.com/bugfest/tor-controller/issues/34))



## [tor-controller-0.1.9] - 2023-01-18

### Bug Fixes

- Peg alpine version to avoid error with deprecated cryptography dependency ([#31](https://github.com/bugfest/tor-controller/issues/31))

### Generic

- [ci-skip] Release 0.7.2



## [tor-controller-0.1.8] - 2023-01-15

### Bug Fixes

- Ensure daemon starts, even if reload not required ([#29](https://github.com/bugfest/tor-controller/issues/29))

### Generic

- Release 0.7.1

- Upgrade release actions version

- [ci-skip] build release

- [ci-skip] build help release


## [tor-controller-0.1.7] - 2022-09-19

### Bug Fixes

- Update build actions to build bugfest/tor-daemon:latest

### Features

- Onion service authorized clients ([#26](https://github.com/bugfest/tor-controller/issues/26))

### Generic

- [ci-skip] Fix logo aligment


## [0.6.1] - 2022-08-05

### Bug Fixes

- Missing tor-daemon build :_)
- Update ingress example - fixes #24
- Add Vanguards addon deploy/setup to TODO - fixes #23

### Documentation

- New logo

### Generic

- [ci-skip] Reformat README. Fix some typos


## [tor-controller-0.1.6] - 2022-07-28

### Generic

- [ci-skip] Updated helm chart to use version 0.6.0

- Tor crd 0.6.1 ([#20](https://github.com/bugfest/tor-controller/issues/20))

* Tor Spec

* tor_controller schematic

* Controller implementation

* Tor daemon services multiple address support. Listen address and policies using IPv4/IPv6 dual stack.

* chore: use git-cliff to generate CHANGELOG

* chore: better git-cliff docker command

* fix: tor-controller service account ref in Tor-pod instance

* fix: helm chart project config. ClusterRole update. Default Tor daemon image

* chore(release): prepare for 0.6.1

## [tor-controller-0.1.5] - 2022-07-12

### Features

- Add tor service pod template ([#18](https://github.com/bugfest/tor-controller/issues/18))

### Generic

- * Updating dependencies - version v0.6.0
* Updating Helm Chart - preparing helm chart version v0.1.5


## [tor-controller-0.1.4] - 2022-03-10

### Generic

- [ci-skip] Updated helm chart to use version 0.5.1


## [0.5.1] - 2022-03-10

### Generic

- [ci-skip] Update changelog - v0.5.0

- [ci-skip] Fixes #11
Updated multiarch dockerfiles.
Tor is compiled in a separate project: bugfest/tor-docker to speed up the build times
Added QEMU/arm64 how to

- [skip-ci] Fixes #12 Use bugfest/echoserver (multiarch) instead of google's
Add QEMU arm64 instructions to get a k3s sanbox

- Fixes #13 - Bring your own key ([#14](https://github.com/bugfest/tor-controller/issues/14))



## [tor-controller-0.1.3] - 2022-02-20

### Generic

- [ci-skip] Update changelog

- [ci-skip] TOR is Tor. Updated roadmap

- Tor updated to version 0.4.6.10 ([#10](https://github.com/bugfest/tor-controller/issues/10))

Metrics exporters (controller and managers)
Helm chart service monitor creation (controller)
Updated CRDs to enable Service Monitor creation

## [tor-controller-0.1.2] - 2022-02-10

### Generic

- [ci-skip] Update issue templates
- [ci-skip] Updated Changelog

- [ci-skip] Updated helm chart dir in Makefile

- [ci-skip] Updated versions and badges

- OnionBalancedService Initial implementation

- Fix tor reloading crash-loop

- Remove unused function

- Add short names for OnionServices: onion,os and OnionBalancedServices: onionha,oha,obs

- Manager role requires ConfigMap access

- Cleanup and minor fixes

- Helm chart updates

- Helm chart README update

- OnionBalancedService How-To

- Document tor-onionbalance-manager image

- Update workflows to build tor-onionbalance-manager

- Merge pull request #9 from bugfest/onionbalancedservice

OnionBalancedService implementation - Fixes #8 

## [tor-controller-0.1.1] - 2022-01-29

### Generic

- [ci-skip] reverting chart version

- [ci-skip] App version 0.3.2, Chart version 0.1.1

- [ci-skip] Updated Chart Readme


## [0.3.2] - 2022-01-29

### Generic

- Update action branch: master

- Merge pull request #4 from bugfest/multiarch

Update action branch: master
- Fixing tags typo

- Merge pull request #6 from bugfest/multiarch

Fixing tags typo
- [ci-skip] add multiarch build-tag workflow

- App version 0.3.2, Chart version 0.1.1

- Testing CI workflows with a blank commit


## [tor-controller-0.1.0] - 2022-01-29

### Generic

- #1 Helm chart for installing

- Merge pull request #2 from bugfest/helm

Helm chart for installing. Fixes #1 
- Updated CHANGELOG

- Preparing chart-releaser-action changes

- Updating chart repo URL and instructions

- Update action branch: master


## [0.3.1] - 2022-01-05

### Bug Fixes

- Fix repo url references


### Generic

- Initial migration

- Initial v1alpha2

- ProjectConfig initial

- ProjectConfig fix crd

- Projectconfig and manager rbac working

- Get out-of-cluster run work again

- Migrating agents

- V1alpha2 e2e test working

- Update some links

- Improved objects sync

- Added README and LICENSE. Cleaned up old samples

- Generate install manifest

- Update docker container refs

- Integrated install manifests


<!-- generated by git-cliff -->
