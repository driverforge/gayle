# Third-Party Licenses

gayle is built with the Go standard library and a number of open-source modules, including the [AWS SDK for Go](https://github.com/aws/aws-sdk-go-v2) (Apache-2.0), the [Azure SDK for Go](https://github.com/Azure/azure-sdk-for-go) (MIT), [Cobra](https://github.com/spf13/cobra) (Apache-2.0), and the [Charm](https://charm.land) libraries (MIT).

No third-party source is vendored into this repository. All dependencies are fetched as Go modules at build time; each is distributed under its own license, declared in that module's own repository. See [`go.mod`](go.mod) for the direct dependency set and [`go.sum`](go.sum) for the full resolved tree.

If you believe a bundled or referenced component is missing appropriate attribution here, please open an issue or email **oss@driverforge.com**.
