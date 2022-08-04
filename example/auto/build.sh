#!/bin/bash

cd `dirname $0`/../

HOMEDIR=`pwd`
OUTDIR="$HOMEDIR/output"

# make output dir
if [ ! -d "$OUTDIR" ];then
    mkdir $OUTDIR
fi
rm -rf "$OUTDIR/*"

function buildpkg() {
    output=$1
    pkg=$2

    version=`git rev-parse --abbrev-ref HEAD`
    if [ $? != 0 ]; then
        version="unknow"
    fi
    
    commitId=`git rev-parse --short HEAD`
    if [ $? != 0 ]; then
        commitId="unknow"
    fi

    buildTime=$(date "+%Y-%m-%d-%H:%M:%S")

    # build
    if [ ! -d "$OUTDIR/bin" ]; then
        mkdir "$OUTDIR/bin"
    fi

    ldflags="-X main.Version=$version -X main.BuildTime=$buildTime -X main.CommitID=$commitId"
    echo "go build -o "$OUTDIR/bin/$output" -ldflags \"$ldflags\" $pkg"

    go build -o "$OUTDIR/bin/$output" -ldflags \
        "-X main.Version=$version -X main.BuildTime=$buildTime -X main.CommitID=$commitId" $pkg
}

# build chain
buildpkg chain "$HOMEDIR/cmd/chain/main.go"
# adapetr client
buildpkg client "$HOMEDIR/cmd/client/main.go"

# build output
cp -r "$HOMEDIR/conf" "$OUTDIR"
sed -i 's/rootPath: \S*/rootPath:/g' "$OUTDIR/conf/env.yaml"
cp "$HOMEDIR/auto/control.sh" "$OUTDIR"
mkdir -p "$OUTDIR/data"
cp -r "$HOMEDIR/data/genesis" "$OUTDIR/data"
cp -r "$HOMEDIR/data/keys" "$OUTDIR/data"
cp -r "$HOMEDIR/data/netkeys" "$OUTDIR/data"

echo "compile done!"
