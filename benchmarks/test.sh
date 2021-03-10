#!/bin/bash

set -e

echo ""
echo "--- BENCH PING PONG START ---"
echo ""

cd $(dirname "${BASH_SOURCE[0]}")
function cleanup {
    echo "--- BENCH PING PONG DONE ---"
    kill -9 $(jobs -rp)
    wait $(jobs -rp) 2>/dev/null
}
trap cleanup EXIT

mkdir -p bin
$(pkill -9 gev-echo-server || printf "")

function gobench {
    echo "--- $1 ---"
    if [ "$3" != "" ]; then
        go build -o $2 $3
    fi
    GOMAXPROCS=4 $2 --port $4 --loops 4 &

    sleep 1
    echo "*** 1000 connections, 10 seconds, 4096 byte packets"
    GOMAXPROCS=4 go run client/main.go -c 1000 -t 100 -m 4096 -a 127.0.0.1:$4
    echo "--- DONE ---"
    echo ""
}

gobench "GEV"  bin/gev-echo-server ../example/echo/echo.go 5000

