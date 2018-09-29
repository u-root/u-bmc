# Copyright 2018 the u-root Authors. All rights reserved
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file.

all: version.go ssh_keys.go

ssh_keys.go: ../ssh_keys.pub
	cat "$<" | ./ssh_keys.sh > "$@"

version.go:
	./version.sh > "$@"

.PHONY: all version.go
