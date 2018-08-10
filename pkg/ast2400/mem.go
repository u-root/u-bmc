// Copyright 2018 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ast2400

// TODO(bluecmd): Since we support building for both host and BMC this should
// be uint32 instead of uintptr
type memProvider interface {
	MustRead32(uintptr) uint32
	MustRead8(uintptr) uint8
	MustWrite32(uintptr, uint32)
	MustWrite8(uintptr, uint8)
	Close()
}

var mem memProvider

func (a *Ast) Mem() memProvider {
	return a.mem
}
