# Copyright 2018 u-root Authors
#
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file

apparmor/profiles.go: $(shell find $(ROOT_DIR)apparmor -name \*profile.go -type f)

apparmor/%.profile.go: apparmor/%.profile
	$(ROOT_DIR)apparmor/compile.sh $(shell basename $<) > $@
