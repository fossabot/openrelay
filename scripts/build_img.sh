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

source ~/.bash_profile
HOME_PATH=${HOME}
ENTRY_POINT=${REPO_ROOT_PATH}/cmd/openrelay/main.go
GOCC=go
GOXC=gox
GIT_COMMIT=$(git rev-parse HEAD)
LD_FLAGS="-X main.GitCommit=${GIT_COMMIT} $LD_FLAGS"
export PKG_CONFIG_PATH=/usr/local/lib/pkgconfig
SOURCES_PATH=${REPO_ROOT_PATH}/build/rpms/SOURCES

#XC_ARCH=${XC_ARCH:-"386 amd64 arm"}
#XC_OS=${XC_OS:-linux darwin windows freebsd openbsd solaris}
#XC_EXCLUDE_OSARCH="!darwin/arm !darwin/386"

if [[ -n "${OR_RELEASE}" ]]; then
    LD_FLAGS="-s -w"
fi

# clean directories
rm -rf ${REPO_ROOT_PATH}/bin/${IMAGE_NAME}
rm -rf ${REPO_ROOT_PATH}/pkg/*
rm -rf ${SOURCES_PATH}/*

# preprocess here.

echo ${GOCC} build -o ${REPO_ROOT_PATH}/bin/${IMAGE_NAME} -ldflags \"$LD_FLAGS\" ${ENTRY_POINT}
${GOCC} build -o ${REPO_ROOT_PATH}/bin/${IMAGE_NAME} -ldflags "$LD_FLAGS" ${ENTRY_POINT}
#go build -o ${REPO_ROOT_PATH}/bin/replay cmd/openrelay/replay.go

mkdir -p ${SOURCES_PATH}/${IMAGE_FULLNAME}
cp ${REPO_ROOT_PATH}/bin/${IMAGE_NAME} ${SOURCES_PATH}/${IMAGE_FULLNAME}/
cp ${REPO_ROOT_PATH}/configs/${IMAGE_NAME}-boot.sh ${SOURCES_PATH}/${IMAGE_FULLNAME}/
cp ${REPO_ROOT_PATH}/configs/${IMAGE_NAME}.service ${SOURCES_PATH}/${IMAGE_FULLNAME}/
cp ${REPO_ROOT_PATH}/configs/${IMAGE_NAME}.env ${SOURCES_PATH}/${IMAGE_FULLNAME}/
cp ${REPO_ROOT_PATH}/extlib/libczmq.so.*.*.* ${SOURCES_PATH}/${IMAGE_FULLNAME}/
cp ${REPO_ROOT_PATH}/extlib/libsodium.so.*.*.* ${SOURCES_PATH}/${IMAGE_FULLNAME}/
cp ${REPO_ROOT_PATH}/extlib/libzmq.so.*.*.* ${SOURCES_PATH}/${IMAGE_FULLNAME}/

cd ${SOURCES_PATH}
tar zcvf ${IMAGE_FULLNAME}.tar.gz ${IMAGE_FULLNAME}
cd -

cd ${RET_DIR}