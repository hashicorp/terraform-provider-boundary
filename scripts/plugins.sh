#!/usr/bin/env bash
# Copyright IBM Corp. 2020, 2025
# SPDX-License-Identifier: MPL-2.0

#
# This script builds the required plugins.
set -e

if [[ -z "$GOOS" ]]; then
    echo "Must provide GOOS in environment" 1>&2
    exit 1
fi

if [[ -z "GOARCH" ]]; then
    echo "Must provide GOARCH in environment" 1>&2
    exit 1
fi

BINARY_SUFFIX=""
if [ "${GOOS}x" = "windowsx" ]; then
    BINARY_SUFFIX=".exe"
fi

# Get the parent directory of where this script is.
SOURCE="${BASH_SOURCE[0]}"
while [ -h "$SOURCE" ] ; do SOURCE="$(readlink "$SOURCE")"; done
export DIR="$( cd -P "$( dirname "$SOURCE" )/.." && pwd )"

echo "==> Building kms plugins for ${GOOS}-${GOARCH}..."
rm -f $DIR/plugins/kms/assets/${GOOS}/${GOARCH}/boundary-plugin-kms-*
for CURR_PLUGIN in $(ls $DIR/plugins/kms/mains); do
    echo "==> Building $CURR_PLUGIN plugin..."
    cd $DIR/plugins/kms/mains/$CURR_PLUGIN;
    go build -v -o $DIR/plugins/kms/assets/${GOOS}/${GOARCH}/boundary-plugin-kms-${CURR_PLUGIN}${BINARY_SUFFIX} .;
    cd $DIR;
done;
cd $DIR/plugins/kms/assets/${GOOS}/${GOARCH};
for CURR_PLUGIN in $(ls boundary-plugin-kms-*); do
    echo "==> gzip $CURR_PLUGIN plugin..."
    gzip -f -9 $CURR_PLUGIN;
done;
cd $DIR;
