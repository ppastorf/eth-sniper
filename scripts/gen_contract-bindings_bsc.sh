#!/usr/bin/env bash

set -euo pipefail

dex_name=$1
contract_name=$2
address=$3

pkg_name=$(echo $dex_name | awk '{print tolower($0)}')
abi_file="contracts/bsc/${pkg_name}/${contract_name}.abi"
go_file="contracts/bsc/${pkg_name}/${contract_name}.go"

mkdir -p "contracts/bsc/${pkg_name}"

curl -s -X GET "https://api.bscscan.com/api?module=contract&action=getabi&address=${address}" \
    | jq .result | sed -e 's/\\//g' -e 's/^"//g' -e 's/"$//g' \
    | jq > "$abi_file"

abigen --abi "$abi_file" --out "$go_file" --pkg "$contract_name"

sed -i'' "s/package ${contract_name}$/package ${pkg_name}/g" $go_file