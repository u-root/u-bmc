# Copyright 2018 the u-root Authors. All rights reserved
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file.

all: version.go ssh_keys.go acme.go sim_pebble.go

../ssh_keys.pub:
	touch $@

ssh_keys.go: ../ssh_keys.pub
	cat "$<" | ./ssh_keys.sh > "$@"

version.go:
	./version.sh > "$@"

acme.go: i_agree_to_the_acme_terms
	cp acme.go.tmpl acme.go

sim-pebble.key:
	openssl ecparam -name secp256r1 -genkey -noout -out sim-pebble.key

sim-pebble.crt: sim-pebble.key
	openssl req \
		-new \
		-x509 \
		-key sim-pebble.key \
		-out sim-pebble.crt \
		-subj /CN=sim-pebble \
		-reqexts SAN \
		-extensions SAN \
		-config sim-pebble.cnf \
		-sha256 \
		-days 36500

sim_pebble.go: sim-pebble.crt
	./sim_pebble.sh > "$@"

.PHONY: all version.go
