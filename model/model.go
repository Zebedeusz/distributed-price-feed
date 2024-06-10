package model

import (
	"time"

	"github.com/shopspring/decimal"
)

type EthPrice struct {
	ID        int
	Price     decimal.Decimal
	CreatedAt time.Time
}

type EthPriceMessage struct {
	Price      decimal.Decimal
	Signatures map[NodeID]PayloadSignature
}

func (p *EthPriceMessage) IsSignedByNode(nodeID string) bool {
	_, ok := p.Signatures[NodeID(nodeID)]
	return ok
}

type NodeID string

type PayloadSignature string
