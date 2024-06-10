package main

import (
	"context"
	"time"

	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	drouting "github.com/libp2p/go-libp2p/p2p/discovery/routing"
	dutil "github.com/libp2p/go-libp2p/p2p/discovery/util"
	"go.uber.org/zap"
	log "go.uber.org/zap"
)

func AdvertiseItselfAndDiscoverPeersViaDHT(ctx context.Context, logger *log.Logger, h host.Host, dht *dht.IpfsDHT, rendezvous string) (<-chan network.Conn, error) {
	var routingDiscovery = drouting.NewRoutingDiscovery(dht)
	dutil.Advertise(ctx, routingDiscovery, rendezvous)

	connsChann := make(chan network.Conn)

	// Just to check if it succeeds
	_, err := dutil.FindPeers(ctx, routingDiscovery, rendezvous)
	if err != nil {
		return connsChann, err
	}

	go func() {
		ticker := time.NewTicker(time.Second * 1)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				peers, err := dutil.FindPeers(ctx, routingDiscovery, rendezvous)
				if err != nil {
					close(connsChann)
				}

				logger.Info("peers len", zap.Int("peers len", len(peers)))

				for _, p := range peers {
					if p.ID == h.ID() {
						continue
					}

					streamCreated, err := h.Peerstore().Get(p.ID, "stream_created")
					if err != nil || !streamCreated.(bool) {
						co, err := h.Network().DialPeer(ctx, p.ID)
						if err != nil {
							logger.Warn("failed to connect to peer", zap.String("peer", p.String()), zap.Error(err))
							continue
						}
						logger.Info("connected to peer", zap.String("peer", p.String()))

						h.Peerstore().Put(p.ID, "stream_created", true)

						connsChann <- co
					}
				}
			}
		}
	}()

	return connsChann, nil
}
