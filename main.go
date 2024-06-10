package main

import (
	"context"
	"distributed-price-feed/config"
	"distributed-price-feed/connection"
	"distributed-price-feed/database"
	priceshandling "distributed-price-feed/prices-handling"
	"distributed-price-feed/pubsub"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	libp2p "github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/crypto"
	"go.uber.org/zap"
	log "go.uber.org/zap"
)

var logger, _ = log.NewProduction()

func main() {
	defer logger.Sync()

	config, err := config.ParseFlags()
	if err != nil {
		logger.Fatal("failed to parse flags", zap.Error(err))
	}

	db, err := database.ConnectToDatabase(config)
	if err != nil {
		logger.Fatal("failed to connect to database", zap.Error(err))
	}

	priv, _, err := crypto.GenerateKeyPair(
		crypto.Ed25519,
		-1,
	)
	if err != nil {
		logger.Fatal("failed generating key pair", zap.Error(err))
	}

	node, err := libp2p.New(
		libp2p.Identity(priv),
		libp2p.ListenAddrStrings(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", config.Port)),
		libp2p.ForceReachabilityPublic(),
		// TODO: support TLS connections
		// libp2p.Security(libp2ptls.ID, libp2ptls.New),
	)
	if err != nil {
		logger.Fatal("failed initializing node", zap.Error(err))
	}

	logger = logger.With(log.String("node_id", node.ID().String()))
	logger.Info("Node initialized")

	ctx := context.Background()

	kdht, err := connection.StartDHT(ctx, logger, node, config.BootstrapPeer)
	if err != nil {
		logger.Fatal("failed to start DHT", zap.Error(err))
	}

	topic, sub, err := pubsub.AdvertiseInDHTAndStartPubSub(ctx, logger, node, kdht, config.RendezvousPoint)
	if err != nil {
		logger.Fatal("failed while DHT advertisement or pub sub start", zap.Error(err))
	}

	pricesHandler, err := priceshandling.NewPricesHandler(logger, db, node.ID(), topic, sub)
	if err != nil {
		logger.Fatal("failed to initialize prices handler", zap.Error(err))
	}

	go pricesHandler.SubscribeToEthPriceMessages(ctx)
	go pricesHandler.PollForEthPrice(ctx)

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	<-ch
	logger.Info("Received signal, shutting down...")

	if err := node.Close(); err != nil {
		panic(err)
	}
}
