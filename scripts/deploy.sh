#!/bin/bash

kubectl apply -f ./k8s/manifest_bootstrap.yaml

kubectl wait --for=condition=available deployment/libp2p-node-bootstrap

POD_NAME=$(kubectl get pods -l app=libp2p-node-bootstrap -o jsonpath="{.items[0].metadata.name}")
LOGS=$(kubectl logs $POD_NAME)

# Search for peer ID in the first line of bootstrap node logs. Remove quotes and brackets.
PEER_ID=$(echo "$LOGS" | awk 'NR==1{gsub(/"|}/, "", $NF); print $NF}')

SERVICE_NAME="dht"
SERVICE_IP=$(kubectl get svc $SERVICE_NAME -o jsonpath='{.spec.clusterIP}')

kubectl apply -f <(echo "
apiVersion: apps/v1
kind: Deployment
metadata:
  name: libp2p-node
spec:
  replicas: $NODES
  selector:
    matchLabels:
      app: libp2p-node
  template:
    metadata:
      labels:
        app: libp2p-node
        database: postgresql
    spec:
      containers:
      - name: libp2p-node
        image: libp2p-node
        imagePullPolicy: Never
        command: ["/libp2p-node"]
        args:
        - "--peer"
        - "/ip4/$SERVICE_IP/tcp/9000/p2p/$PEER_ID"
        - "--databaseHost"
        - "postgresql"
        ports:
        - containerPort: 9000
")