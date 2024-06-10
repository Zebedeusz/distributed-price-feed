package connection

import (
	"context"

	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	log "go.uber.org/zap"
)

func StartDHT(ctx context.Context, logger *log.Logger, host host.Host, bootstrapPeer string) (*dht.IpfsDHT, error) {
	var options []dht.Option
	if bootstrapPeer == "" {
		options = append(options, dht.Mode(dht.ModeServer))
	}

	kdht, err := dht.New(ctx, host, options...)
	if err != nil {
		return nil, errors.Wrap(err, "failed to initialize DHT")
	}

	if err = kdht.Bootstrap(ctx); err != nil {
		return nil, errors.Wrap(err, "failed to bootstrap DHT")
	}

	if bootstrapPeer != "" {
		multiAddr, err := multiaddr.NewMultiaddr(bootstrapPeer)
		if err != nil {
			return nil, errors.Wrap(err, "failed to parse bootstrap peer as multiaddress")
		}
		peerinfo, err := peer.AddrInfoFromP2pAddr(multiAddr)
		if err != nil {
			return nil, errors.Wrap(err, "failed to parse multiaddress as address info")
		}
		if err := host.Connect(ctx, *peerinfo); err != nil {
			return nil, errors.Wrap(err, "failed to connect to bootstrap peer")
		}
		logger.Info("connected to bootstrap peer", zap.String("peer", peerinfo.String()))
	}

	return kdht, nil
}
