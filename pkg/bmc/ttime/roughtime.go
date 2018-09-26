// Copyright 2018 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ttime

import (
	"time"
)

const (
	KEY_TYPE_ED25519 = 1
)

type RoughtimeServer struct {
	Protocol      string
	Address       string
	PublicKey     string
	PublicKeyType int
}

type NtpServer string

func AcquireTime(rs []RoughtimeServer, ntps []NtpServer) (time.Time, error) {
	// TODO(bluecmd): Implement
	return time.Now(), nil
}
