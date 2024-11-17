#!/usr/bin/bash

cd ../builder
go run main.go
node fix_errors.js
cd out
cp ../../strip/*.js .