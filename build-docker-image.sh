#!/bin/bash

make

cp algorand-go-implementation ./docker/algorand-go-implementation

docker build -t algorand:latest -f ./docker/Dockerfile ./docker/
