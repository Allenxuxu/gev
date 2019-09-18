#!/bin/bash

set -e

cd $(dirname "${BASH_SOURCE[0]}")

mkdir -p out/

./bench-echo.sh 2>&1 | tee out/echo.txt

go run analyze.go