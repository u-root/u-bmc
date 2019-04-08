// Copyright 2019 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cert

import (
	"io"
	"log"
	"testing"
)

// From https://github.com/mmcloughlin/avo/blob/master/internal/test/utils.go
func Logger(tb testing.TB, what string) *log.Logger {
	return log.New(Writer(tb), what+" ", log.LstdFlags)
}

type writer struct {
	tb testing.TB
}

func Writer(tb testing.TB) io.Writer {
	return writer{tb}
}

func (w writer) Write(p []byte) (n int, err error) {
	w.tb.Log(string(p))
	return len(p), nil
}
