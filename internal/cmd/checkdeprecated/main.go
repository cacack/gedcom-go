// Command checkdeprecated is the deprecation gate of scripts/check-api-compat.sh.
//
// It reads `apidiff -m` output on stdin and requires every incompatible change
// to be either the removal of a symbol whose doc carried a Deprecated:
// paragraph in the baseline tree, or an entry in the allowlist whose
// migration-guide anchor resolves. It exits 1 when any change is neither.
//
//	go run ./internal/cmd/checkdeprecated -baseline DIR -module PATH -allowlist FILE < apidiff.txt
package main

import "os"

func main() {
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}
