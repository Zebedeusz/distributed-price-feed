package config

import (
	"flag"
)

type Config struct {
	Port            int
	RendezvousPoint string
	BootstrapPeer   string
	DatabaseHost    string
}

func ParseFlags() (Config, error) {
	config := Config{}
	flag.StringVar(&config.RendezvousPoint, "rendezvous", "London Eye", "Unique string to identify within the group of nodes")
	flag.IntVar(&config.Port, "port", 9000, "Port that the node will run on")
	flag.StringVar(&config.BootstrapPeer, "peer", "", "Bootstrap Peer Multiaddress")
	flag.StringVar(&config.DatabaseHost, "databaseHost", "127.0.0.1", "URL of Postgres database instance")
	flag.Parse()

	return config, nil
}
