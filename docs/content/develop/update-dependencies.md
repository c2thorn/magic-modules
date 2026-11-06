---
title: "Update dependencies"
weight: 90
params:
  bookToC: true
aliases:
  - /docs/update-dependencies
---

# Update provider dependencies

The Magic Modules repository does not contain a complete Go module, preventing the use of automated tooling like `go get` from that repository. To add or update provider dependencies, use standard Go tooling to update the dependency in the `terraform-provider-google` provider and copy the results to the upstream files in Magic Modules.

Below are the steps you can follow to make the change:

1. Navigate to the local `google` provider directory:
   ```bash
   cd $GOPATH/src/github.com/hashicorp/terraform-provider-google
   ``` 
2. Update dependencies using the narrowest command that matches your goal (do not run bare `go get` without arguments):
   * To add or upgrade a specific package to a target version (preferred when unblocking a feature or fix):
   ```bash
   go get google.golang.org/api@<version>
   go mod tidy
   ```
   * To upgrade all module and test dependencies across the provider (for dedicated dependency-maintenance PRs only):
   ```bash
   go get -u ./...
   go mod tidy
   ```
3. Copy the contents of the updated `go.mod` and `go.sum` files into [`mmv1/third_party/terraform/go.mod`](https://github.com/GoogleCloudPlatform/magic-modules/blob/main/mmv1/third_party/terraform/go.mod) and [`mmv1/third_party/terraform/go.sum`](https://github.com/GoogleCloudPlatform/magic-modules/blob/main/mmv1/third_party/terraform/go.sum) in `magic-modules` respectively.
4. Resolve any build or test errors caused by the dependency upgrade.
5. Open a standalone pull request (`release-note:none`) for broad or multi-package dependency upgrades (plus any required compatibility fixes). If a feature or bugfix PR requires new fields from a targeted single-package bump (such as `google.golang.org/api`), you may include that clean `go.mod`/`go.sum` bump in the same PR without unrelated dependency churn.
