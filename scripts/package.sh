#!/bin/bash

go run ./cmd/build
tar -cvzf "site.tar.gz" -C ./build .
