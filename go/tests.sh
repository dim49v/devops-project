#!/usr/bin/env bash

unset GOPATH

go test -coverprofile=count.cov ./cmd/ -v > ./test/test.txt 2>&1
go tool cover -func count.cov > ./test/cover.txt 2>&1
go tool cover -html count.cov -o ./test/cover.html
tail -f /dev/null


