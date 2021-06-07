#!/bin/bash

set -e

echo ""
echo "--- BENCH websocket PING PONG START ---"
echo ""

cd $(dirname "${BASH_SOURCE[0]}")
function cleanup() {
  echo "--- BENCH websocket PING PONG DONE ---"
  kill -9 $(jobs -rp)
  wait $(jobs -rp) 2>/dev/null
}
trap cleanup EXIT

mkdir -p bin
$(pkill -9 websocket-server || printf "")

function gobench() {
  echo "--- $1 ---"
  if [ "$3" != "" ]; then
    go build -o $2 $3
  fi
  $2 --port $4 --loops 8 &

  sleep 1
  go run websocket/client/main.go -c 3000 -t 5 -m 2048 -a 127.0.0.1:$4
  pkill -9 $2 || printf ""
  echo "--- DONE ---"
  echo ""
}

gobench "Gev-websocket" bin/websocket-server websocket/server.go 6000
