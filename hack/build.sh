#!/bin/bash
echo "This file is gonna be deleted"
export GOPATH=$GOPATH:`pwd`/vendor
echo "Set GOPATH at $GOPATH"

os=$1
# try to detect the os
if [ -z "$os" ] ; then
    
    if [ "$(uname)" == "Darwin" ]; then
        os="osx"
    elif [ "$(expr substr $(uname -s) 1 5)" == "Linux" ]; then
        os="linux"
    else
        echo "Unable to auto detect the Os"
        exit 1
    fi

    echo "Set os to $os (auto detected)"
fi

# finaly launch the make
make $os
