#!/bin/bash
# Copyright 2018 the u-root Authors. All rights reserved
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file.

cat << _EOF_
// AUTOGENERATE BY ssh_keys.sh
package bmc

var (
	authorizedKeys = []string{
_EOF_

IFS='
'
while IFS='' read -r line || [[ -n "$line" ]]
do
  echo -e "\t\t\"$line\","
done
cat << _EOF_
	}
)
_EOF_
