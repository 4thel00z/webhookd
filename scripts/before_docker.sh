#! /bin/sh

this_dir=$(dirname "$0")
export PREFIX=/usr/local
mkdir -p $PREFIX/go/src/webhookd $PREFIX/go/src/_/builds
cp -r $this_dir/../* $PREFIX/go/src/webhookd
ln -s $PREFIX/go/src/webhookd $PREFIX/go/src/_/builds/webhookd
