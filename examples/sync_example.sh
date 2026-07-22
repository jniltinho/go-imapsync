#!/bin/sh
# Example: dry just-folders run (replace credentials).
# See README.md for env-based passwords.

set -eu

./dist/go-imapsync \
  --host1 test1.example.com --user1 user1 --password1 'secret1' \
  --host2 test2.example.com --user2 user2 --password2 'secret2' \
  --justfolders --dry "$@"
