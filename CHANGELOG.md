# Changelog

## [3.0.0](https://github.com/brunojet/go-edge-cache/compare/v2.0.0...v3.0.0) (2026-06-06)


### ⚠ BREAKING CHANGES

* None (CloudFront handles cache, Lambda behavior unchanged externally)

### Features

* Apply transfer manager tuning from environment variables ([23c8b29](https://github.com/brunojet/go-edge-cache/commit/23c8b2998ec5c2c121f387c2bba7099b27bc95b5))
* cache refactor with CloudFront and Origin Shield ([#4](https://github.com/brunojet/go-edge-cache/issues/4)) ([15c2a88](https://github.com/brunojet/go-edge-cache/commit/15c2a88c4efee38febd2cd9e97dd29f5e9d79bb0))
* Conservative transfer manager tuning (25MB/50MB) ([b84e929](https://github.com/brunojet/go-edge-cache/commit/b84e929279d2c569c246b1c3ec27d76acb9964cc))
* distributed lock with error handling and cost optimization ([#5](https://github.com/brunojet/go-edge-cache/issues/5)) ([6bf89a6](https://github.com/brunojet/go-edge-cache/commit/6bf89a647e75e3e207cbb2ea64924b6209528e55))
* implement distributed locks for concurrent cache protection ([#3](https://github.com/brunojet/go-edge-cache/issues/3)) ([7245553](https://github.com/brunojet/go-edge-cache/commit/7245553788afbcb964f3c38df676021dbb262cf3))
* Integrate AWS Secrets Manager for CloudFront signing ([dc2e466](https://github.com/brunojet/go-edge-cache/commit/dc2e466bda1bbb48cee37fa432c998e2f8b12d90))
* security hardening, risk mitigations and observability ([#6](https://github.com/brunojet/go-edge-cache/issues/6)) ([2c13b46](https://github.com/brunojet/go-edge-cache/commit/2c13b466bcc4d5c1ba16b0f70df51910c812cee5))
* Update sign-url to use new secret rotation contract ([b5dcb43](https://github.com/brunojet/go-edge-cache/commit/b5dcb4378dd8ab2f267a1f53c0a9300de5828d2a))
* Upgrade to go-infra-adapters v4.1.0 and tune Lambda for 900MB extreme test ([8f84978](https://github.com/brunojet/go-edge-cache/commit/8f849786a8de98489c7eee4649969cc13d8ebf4c))


### Bug Fixes

* Escape % in log.Printf format string (100%% streaming) ([9886168](https://github.com/brunojet/go-edge-cache/commit/9886168ee699633bb0fa232d566f67f427ef950e))
* Handle errcheck and gosec linter warnings ([647695c](https://github.com/brunojet/go-edge-cache/commit/647695c55a1563be36a5b547bf6db001bb11b175))
* Match AWS CloudFront Custom Policy signing format ([ed23013](https://github.com/brunojet/go-edge-cache/commit/ed2301334d13c9dcfc287c017fd6e47ca113e192))
* Rename -file flag to -path to clarify URL vs filesystem ([df4df69](https://github.com/brunojet/go-edge-cache/commit/df4df6963e006b32abcb4f8b491a75dd393760e9))
* Update to v4.0.0 and add NewCloudFrontSigner file wrapper ([fa0555f](https://github.com/brunojet/go-edge-cache/commit/fa0555f7a127ff350f30d13575ee92c5f8c1318f))


### Reverts

* Lambda doesn't need Secrets Manager access ([6c26198](https://github.com/brunojet/go-edge-cache/commit/6c2619806679335d02df001b0be238439471cfd1))


### Documentation

* add architecture doc with mermaid diagrams ([#10](https://github.com/brunojet/go-edge-cache/issues/10)) ([3688619](https://github.com/brunojet/go-edge-cache/commit/36886197475e007e5e6bfa7d9fd4183ee1584fc0))
* add branch/PR workflow (no squash) to CLAUDE.md ([ca6812a](https://github.com/brunojet/go-edge-cache/commit/ca6812a2541327b6f682ac81784b801524dba7d4))
* Add upgrade guide for go-infra-adapters v4.0.0 ([cf948a2](https://github.com/brunojet/go-edge-cache/commit/cf948a2bbd4faf188c0962572bba83dc70ac0811))
* restructure markdown into docs/ and prune stale files ([#9](https://github.com/brunojet/go-edge-cache/issues/9)) ([cee97aa](https://github.com/brunojet/go-edge-cache/commit/cee97aa690f5487344b60a79b32c318c2de919a9))


### Code Refactoring

* Implement AWS CloudFront Canned Policy signing ([4a01798](https://github.com/brunojet/go-edge-cache/commit/4a017984844f04ceff1d8f9a9dedf68a53eef683))
* replace local CloudFrontSigner with go-infra-adapters v4.0.0 ([1508676](https://github.com/brunojet/go-edge-cache/commit/1508676363c20d56634de776c3b54c94bd185288))
* Use go-infra-adapters storage API instead of raw transfer manager ([67505c9](https://github.com/brunojet/go-edge-cache/commit/67505c981b275603d66e66b3c23ac527f5697a72))

## [2.0.0](https://github.com/brunojet/go-edge-cache/compare/v1.0.0...v2.0.0) (2026-06-06)


### ⚠ BREAKING CHANGES

* None (CloudFront handles cache, Lambda behavior unchanged externally)

### Features

* Apply transfer manager tuning from environment variables ([23c8b29](https://github.com/brunojet/go-edge-cache/commit/23c8b2998ec5c2c121f387c2bba7099b27bc95b5))
* cache refactor with CloudFront and Origin Shield ([#4](https://github.com/brunojet/go-edge-cache/issues/4)) ([15c2a88](https://github.com/brunojet/go-edge-cache/commit/15c2a88c4efee38febd2cd9e97dd29f5e9d79bb0))
* Conservative transfer manager tuning (25MB/50MB) ([b84e929](https://github.com/brunojet/go-edge-cache/commit/b84e929279d2c569c246b1c3ec27d76acb9964cc))
* distributed lock with error handling and cost optimization ([#5](https://github.com/brunojet/go-edge-cache/issues/5)) ([6bf89a6](https://github.com/brunojet/go-edge-cache/commit/6bf89a647e75e3e207cbb2ea64924b6209528e55))
* implement distributed locks for concurrent cache protection ([#3](https://github.com/brunojet/go-edge-cache/issues/3)) ([7245553](https://github.com/brunojet/go-edge-cache/commit/7245553788afbcb964f3c38df676021dbb262cf3))
* Integrate AWS Secrets Manager for CloudFront signing ([dc2e466](https://github.com/brunojet/go-edge-cache/commit/dc2e466bda1bbb48cee37fa432c998e2f8b12d90))
* security hardening, risk mitigations and observability ([#6](https://github.com/brunojet/go-edge-cache/issues/6)) ([2c13b46](https://github.com/brunojet/go-edge-cache/commit/2c13b466bcc4d5c1ba16b0f70df51910c812cee5))
* Update sign-url to use new secret rotation contract ([b5dcb43](https://github.com/brunojet/go-edge-cache/commit/b5dcb4378dd8ab2f267a1f53c0a9300de5828d2a))
* Upgrade to go-infra-adapters v4.1.0 and tune Lambda for 900MB extreme test ([8f84978](https://github.com/brunojet/go-edge-cache/commit/8f849786a8de98489c7eee4649969cc13d8ebf4c))


### Bug Fixes

* Escape % in log.Printf format string (100%% streaming) ([9886168](https://github.com/brunojet/go-edge-cache/commit/9886168ee699633bb0fa232d566f67f427ef950e))
* Handle errcheck and gosec linter warnings ([647695c](https://github.com/brunojet/go-edge-cache/commit/647695c55a1563be36a5b547bf6db001bb11b175))
* Match AWS CloudFront Custom Policy signing format ([ed23013](https://github.com/brunojet/go-edge-cache/commit/ed2301334d13c9dcfc287c017fd6e47ca113e192))
* Rename -file flag to -path to clarify URL vs filesystem ([df4df69](https://github.com/brunojet/go-edge-cache/commit/df4df6963e006b32abcb4f8b491a75dd393760e9))
* Update to v4.0.0 and add NewCloudFrontSigner file wrapper ([fa0555f](https://github.com/brunojet/go-edge-cache/commit/fa0555f7a127ff350f30d13575ee92c5f8c1318f))


### Reverts

* Lambda doesn't need Secrets Manager access ([6c26198](https://github.com/brunojet/go-edge-cache/commit/6c2619806679335d02df001b0be238439471cfd1))


### Documentation

* add architecture doc with mermaid diagrams ([#10](https://github.com/brunojet/go-edge-cache/issues/10)) ([3688619](https://github.com/brunojet/go-edge-cache/commit/36886197475e007e5e6bfa7d9fd4183ee1584fc0))
* add branch/PR workflow (no squash) to CLAUDE.md ([ca6812a](https://github.com/brunojet/go-edge-cache/commit/ca6812a2541327b6f682ac81784b801524dba7d4))
* Add upgrade guide for go-infra-adapters v4.0.0 ([cf948a2](https://github.com/brunojet/go-edge-cache/commit/cf948a2bbd4faf188c0962572bba83dc70ac0811))
* restructure markdown into docs/ and prune stale files ([#9](https://github.com/brunojet/go-edge-cache/issues/9)) ([cee97aa](https://github.com/brunojet/go-edge-cache/commit/cee97aa690f5487344b60a79b32c318c2de919a9))


### Code Refactoring

* Implement AWS CloudFront Canned Policy signing ([4a01798](https://github.com/brunojet/go-edge-cache/commit/4a017984844f04ceff1d8f9a9dedf68a53eef683))
* replace local CloudFrontSigner with go-infra-adapters v4.0.0 ([1508676](https://github.com/brunojet/go-edge-cache/commit/1508676363c20d56634de776c3b54c94bd185288))
* Use go-infra-adapters storage API instead of raw transfer manager ([67505c9](https://github.com/brunojet/go-edge-cache/commit/67505c981b275603d66e66b3c23ac527f5697a72))
