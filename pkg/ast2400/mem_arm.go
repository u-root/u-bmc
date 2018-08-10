// Copyright 2018 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build arm
// Assume arm hosts are the BMC

package ast2400

func openMem() memProvider {
	return openHostMemory()
}
