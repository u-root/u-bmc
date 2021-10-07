// Copyright 2018 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build arm || arm64 || amd64

package bmc

import (
	"encoding/binary"
)

func NativeEndian() binary.ByteOrder {
	return binary.LittleEndian
}
