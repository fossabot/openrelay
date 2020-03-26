#!/bin/sh
# Copyright (c) 2018 FurtherSystem Co.,Ltd. All rights reserved.
#
#   This program is free software; you can redistribute it and/or modify
#   it under the terms of the GNU General Public License as published by
#   the Free Software Foundation; version 2 of the License.
#
#   This program is distributed in the hope that it will be useful,
#   but WITHOUT ANY WARRANTY; without even the implied warranty of
#   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
#   GNU General Public License for more details.
#
#   You should have received a copy of the GNU General Public License
#   along with this program; if not, write to the Free Software
#   Foundation, Inc., 51 Franklin St, Fifth Floor, Boston, MA 02110-1335  USA

source `dirname $0`/common.env

cd ${REPO_ROOT_PATH}

rm -f ${REPO_ROOT_PATH}/bin/${IMAGE_NAME}
LIST=`${PODMAN} images | grep deployments_openrelay | awk '{print $3}' | grep -v IMAGE`
for i in $LIST; do sudo docker rmi -f $i; done
#${PODMAN} system prune --all -f

rm -rf ${REPO_ROOT_PATH}/extlib
#rm -rf ~/go

#${DNF} -y remove epel-release
#${DNF} -y remove tar make gcc gcc-c++ libtool automake autoconf git pkgconfig libunwind libunwind-devel

cd ${RET_DIR}