#!/usr/bin/env bash

set -euo pipefail

name=$1
address=$2

abi_file="contracts/bsc/${name}/${name}.abi"
go_file="contracts/bsc/${name}/${name}.go"

mkdir -p "contracts/bsc/${name}"

curl -s -X GET "https://api.bscscan.com/api?module=contract&action=getabi&address=${address}" \
    | jq .result | sed -e 's/\\//g' -e 's/^"//g' -e 's/"$//g' \
    | jq > "$abi_file"

abigen --abi "$abi_file" --out "$go_file" --pkg "$(echo $name | awk '{print tolower($0)}')"f