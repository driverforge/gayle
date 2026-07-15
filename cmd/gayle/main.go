// Command gayle deploys configuration and secrets to AWS SSM Parameter Store or
// Azure Key Vault from a gayle.yml. See internal/cli for the command tree.
package main

import (
	"os"

	"github.com/driverforge/gayle/internal/cli"
)

func main() {
	os.Exit(cli.Execute())
}
