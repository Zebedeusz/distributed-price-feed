#!/bin/bash

kubectl delete deployments.apps libp2p-node
kubectl delete deployments.apps libp2p-node-bootstrap
kubectl delete deployments.apps postgres