// Command go-imapsync is a one-way IMAP mailbox synchronizer (host1 → host2).
//
// Build with: make build
// Version is injected via -ldflags into package cmd.
package main

import "go-imapsync/cmd"

func main() {
	cmd.Execute()
}
