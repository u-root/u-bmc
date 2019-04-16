// Copyright 2019 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package config

import (
	"crypto/x509"
	"testing"
)

func TestACMECAConfig(t *testing.T) {
	roots := x509.NewCertPool()
	ok := roots.AppendCertsFromPEM([]byte(DefaultConfig.ACME.APICA))
	if !ok {
		t.Fatal("No parsable ACME API CA (ACME.APICA) certificates found")
	}
}
