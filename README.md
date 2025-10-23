# Guardium Data Protection Provider

## Requirements

- [Terraform](https://developer.hashicorp.com/terraform/downloads) >= 1.0
- [Go](https://golang.org/doc/install) >= 1.23
- Ensure ` go env GOBIN` is set
  - If not set add `export GOBIN=/Users/<user>/go/bin/` to your bashrc

## Building The Provider

1. Clone the repository
2. Enter the repository directory
3. Build the provider using the Go `install` command from the root of the project:


```shell
go install .
```

- Add the following to your `$HOME/terraformrc` file 
```terraformrc
provider_installation {

  dev_overrides {
      "ibm/guardium-data-protection" = "/Users/<user>/go/bin"
  }

  # For all other providers, install them directly from their origin provider
  # registries as normal. If you omit this, Terraform will _only_ use
  # the dev_overrides block, and so no other providers will be available.
  direct {}
}
```

- Start the mock authentication server in a different shell 
```bash
go run test/main.go
``` 

- Navigate to `cd examples/data-sources/authentication_example/`
- Run `terraform plan` and see mock access token output

## Publishing The Provider

### Prerequisites
1. Ensure you have [goreleaser](https://goreleaser.com/install/) installed

### Building Release Binaries
1. Test the build process locally:
   ```shell
   goreleaser release --snapshot --clean
   ```
   > Note: This will build all the binaries and place them in the `dist` directory

### Release Process
1. Create a git tag from the branch you wish to build:
   ```shell
   git tag -a v0.0.2
   ```
2. Push this tag to git:
   ```shell
   git push origin v0.0.2
   ```
3. Run goreleaser to build the binaries with this version:
   ```shell
   goreleaser release --snapshot  --clean
   ```