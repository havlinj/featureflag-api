// seedpass is a small CLI that outputs a bcrypt hash of the password given as first argument.
// Used by scripts/seed_admin.sh to create the first admin user.
package main

import (
	"fmt"
	"os"

	"github.com/jan-havlin-dev/featureflag-api/internal/auth"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: seedpass <password>")
		os.Exit(1)
	}
	hash, err := auth.HashPassword(os.Args[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "hash password: %v\n", err)
		os.Exit(1)
	}
	fmt.Print(hash)
}
