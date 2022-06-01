#!/usr/bin/env bash
#
# This script builds the required plugins.
set -e

BINARY_SUFFIX=""
if [ "${GOOS}x" = "windowsx" ]; then
    BINARY_SUFFIX=".exe"
fi

# Get the parent directory of where this script is.
SOURCE="${BASH_SOURCE[0]}"
while [ -h "$SOURCE" ] ; do SOURCE="$(readlink "$SOURCE")"; done
export DIR="$( cd -P "$( dirname "$SOURCE" )/.." && pwd )"

echo "==> Building kms plugins..."
rm -f $DIR/plugins/kms/assets/boundary-plugin-kms*
for CURR_PLUGIN in $(ls $DIR/plugins/kms/mains); do
    echo "==> Building $CURR_PLUGIN plugin..."
    cd $DIR/plugins/kms/mains/$CURR_PLUGIN;
    go build -v -o $DIR/plugins/kms/assets/boundary-plugin-kms-${CURR_PLUGIN}${BINARY_SUFFIX} .;
    cd $DIR;
done;
cd $DIR/plugins/kms/assets;
for CURR_PLUGIN in $(ls boundary-plugin*); do
    echo "==> gzip $CURR_PLUGIN plugin..."
    gzip -f -9 $CURR_PLUGIN;
done;
cd $DIR;
