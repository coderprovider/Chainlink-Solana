#!/usr/bin/env bash

docker run --rm -it -v $(pwd):/workdir --entrypoint bash projectserum/build:v0.19.0