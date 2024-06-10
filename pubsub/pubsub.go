package pubsub

import (
	"context"
	"time"

	dht "github.com/libp2p/go-libp2p-kad-dht"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	drouting "github.com/libp2p/go-libp2p/p2p/discovery/routing"
	dutil "github.com/libp2p/go-libp2p/p2p/discovery/util"
	"go.uber.org/zap"
	log "go.uber.org/zap"
)

func AdvertiseInDHTAndStartPubSub(ctx context.Context, logger *log.Logger, h host.Host, dht *dht.IpfsDHT, rendezvous string) (*pubsub.Topic, *pubsub.Subscription, error) {
	var routingDiscovery = drouting.NewRoutingDiscovery(dht)
	dutil.Advertise(ctx, routingDiscovery, rendezvous)

	ps, err := pubsub.NewGossipSub(ctx, h, pubsub.WithDiscovery(routingDiscovery))
	if err != nil {
		return nil, nil, err
	}

	topic, err := ps.Join("t_x")
	if err != nil {
		return nil, nil, err
	}

	subscriber, err := topic.Subscribe()
	if err != nil {
		return nil, nil, err
	}

	return topic, subscriber, nil
}

// ---
// Below functions were used for tests only. Left here just in case or for future use cases

func AdvertiseInDHTAndStartPubSubTest(ctx context.Context, logger *log.Logger, h host.Host, dht *dht.IpfsDHT, rendezvous string) error {
	var routingDiscovery = drouting.NewRoutingDiscovery(dht)
	dutil.Advertise(ctx, routingDiscovery, rendezvous)

	ps, err := pubsub.NewGossipSub(ctx, h, pubsub.WithDiscovery(routingDiscovery))
	if err != nil {
		return err
	}

	topic, err := ps.Join("t_x")
	if err != nil {
		return err
	}

	go startPublishingRandom(ctx, logger, topic)

	subscriber, err := topic.Subscribe()
	if err != nil {
		return err
	}

	go startSubscribing(ctx, logger, h.ID(), subscriber)

	return nil
}

func startSubscribing(ctx context.Context, logger *log.Logger, hostID peer.ID, subscriber *pubsub.Subscription) {
	for {
		msg, err := subscriber.Next(ctx)
		if err != nil {
			panic(err)
		}
		if msg.ReceivedFrom.String() == hostID.String() {
			continue
		}
		logger.Info("message received", zap.String("from", msg.ReceivedFrom.String()), zap.String("message", string(msg.Data)))
	}
}

func startPublishingRandom(ctx context.Context, logger *log.Logger, topic *pubsub.Topic) {
	ticker := time.NewTicker(time.Second * 1)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-ticker.C:
			if err := topic.Publish(ctx, []byte(msg.String())); err != nil {
				panic(err)
			}
			logger.Info("message published", zap.String("message", msg.String()))
		}
	}
}
