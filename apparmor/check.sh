#!/bin/bash
# Copyright 2019 u-root Authors
#
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file

set -eou pipefail

cd $(dirname $0)

for i in *.profile
do
  ./compile.sh "$i" > "${i}.go.new"
  diff -u "${i}.go" "${i}.go.new"
  rm -f "${i}.go.new"
done
