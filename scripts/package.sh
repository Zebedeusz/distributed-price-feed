#!/bin/bash

docker build -f Dockerfile --tag libp2p-node . 
kind load docker-image libp2p-node 