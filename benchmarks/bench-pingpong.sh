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
$(pkill -9 net-echo-server || printf "")
$(pkill -9 evio-echo-server || printf "")
$(pkill -9 eviop-echo-server || printf "")
$(pkill -9 gev-echo-server || printf "")
$(pkill -9 gnet-echo-server || printf "")

function gobench {
    echo "--- $1 ---"
    if [ "$3" != "" ]; then
        go build -o $2 $3
    fi
    GOMAXPROCS=4 $2 --port $4 --loops 4 &

    sleep 1
    echo "*** 1000 connections, 10 seconds, 4096 byte packets"
    GOMAXPROCS=4 go run client/main.go -c 1000 -t 10 -m 4096 -a 127.0.0.1:$4
    echo "--- DONE ---"
    echo ""
}

gobench "GEV"  bin/gev-echo-server echo/echo.go 5000
gobench "GNET" bin/gnet-echo-server gnet-echo-server/main.go 5001
gobench "GO STDLIB" bin/net-echo-server net-echo-server/main.go 5004

