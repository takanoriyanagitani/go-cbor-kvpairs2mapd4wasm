#!/bin/bash

bin="./cmd/kvpairs2mapd4wasm/kvpairs2mapd4wasm"

export ENV_WASM_PATH=./example.wasm
export ENV_WASM_SIZE_MAX=1048576
export ENV_WASM_MAPPER_NAME=map_kvpair

input1(){
	printf 82 | xxd -r -ps; # an array w/ 2 items

	printf Bhw; # b"hw"
	printf Bwl; # b"wl"
}

input2(){
	input1
	input1
}

input3(){
	input1
	input1
	input1
}

input3 |
	"${bin}" |
	python3 -m cbor2.tool --pretty --sequence | jq -c
