# Distributed Price Feed

Distributed Price Feed spawns a number of libp2p nodes that form a distributed system. Each node pulls ETH price from an external provider, communicates it to the other nodes and once certain conditions are met they all persist this value in a shared postgres instances.

## Dependencies
`github.com/libp2p/go-libp2p` - to setup and run nodes
`github.com/libp2p/go-libp2p-kad-dht` - implementation of Kademlia DHT. For nodes discovery
`github.com/libp2p/go-libp2p-pubsub` - pubsub implementation for libp2p. So that nodes are able to gossip with each other
`github.com/multiformats/go-multiaddr` - implementation of Multiaddr address format. To connect to bootstrap peer
`github.com/lib/pq` - postgres database driver
`github.com/jmoiron/sqlx` -  nice extensions to standard sql library
`github.com/shopspring/decimal` - to make sure no precision is lost in decimal numbers
`go.uber.org/zap` - logging

## Requiremenets

- docker
- kubernetes

## Usage

I used `kind` to run local cluster. If you choose to use another tool, simply run the docker build command from the `./scripts/package.sh` file in order to build docker image. Then load it to your local cluster. Otherwise you may use the helper script below.

To build `docker` image and add it to `kind` local cluster:
```bash
./scripts/package.sh
```

To deploy the network with 4 nodes:
Note: bootstrap node is always deployed. The number here relates to the number of additional nodes to be deployed so in total it's 1 + NODES.
```bash
NODES=4
./scripts/deploy.sh
```

To stop the system:
```bash
./scripts/clean-deployments.sh
```

## Monitoring

To check if the pods are running:
```bash
kubectl get pods
```

To check logs of a pod:
```bash
kubectl logs {{pod_name}}
```

To check the eth prices added to the database:
```bash
kubectl exec -it {{postgres_pod}} -- psql -d postgres -U postgres
```
then:
```sql
select * from eth_prices order by createdat desc;
```