// Copyright 2019 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package apparmor

import (
	"io/ioutil"
)

var (
	profiles = [][]byte{}
)

func Load() error {
	for _, d := range profiles {
		if err := ioutil.WriteFile("/sys/kernel/security/apparmor/.load", d, 0600); err != nil {
			return err
		}
	}
	return nil
}
