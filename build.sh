#!/bin/bash

go build -o bin/wg-vlan ./

GOOS=linux GOARCH=arm64 go build -o bin/wg-vlan.arm64 ./