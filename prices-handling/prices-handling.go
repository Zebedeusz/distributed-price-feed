package priceshandling

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	coingeckoclient "distributed-price-feed/coingecko-client"
	"distributed-price-feed/database"
	"distributed-price-feed/model"
	"encoding/json"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/peer"
	"go.uber.org/zap"
	log "go.uber.org/zap"
)

type PricesHandler struct {
	logger     *log.Logger
	db         *sqlx.DB
	hostID     peer.ID
	topic      *pubsub.Topic
	subscriber *pubsub.Subscription

	// This key should be at least encoded with a different key that's owned by the node
	ecdsaKey *ecdsa.PrivateKey
}

func NewPricesHandler(logger *log.Logger, db *sqlx.DB, hostID peer.ID, topic *pubsub.Topic, subscriber *pubsub.Subscription) (*PricesHandler, error) {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}

	return &PricesHandler{
		logger:     logger,
		db:         db,
		hostID:     hostID,
		topic:      topic,
		subscriber: subscriber,
		ecdsaKey:   privateKey,
	}, nil
}

func (handler *PricesHandler) PollForEthPrice(ctx context.Context) {
	ticker := time.NewTicker(time.Minute * 10)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			price, err := coingeckoclient.GetEthPrice()
			if err != nil {
				handler.logger.Warn("failure getting eth price from coingecko", zap.Error(err))
				continue
			}

			message := model.EthPriceMessage{
				Price: *price,
			}
			message.Signatures = make(map[model.NodeID]model.PayloadSignature)

			sig, err := ecdsa.SignASN1(rand.Reader, handler.ecdsaKey, []byte(price.String()))
			if err != nil {
				handler.logger.Error("failure signing message", zap.Error(err))
				continue
			}

			message.Signatures[model.NodeID(handler.hostID.String())] = model.PayloadSignature(sig)

			if err := handler.marshalAndPublish(ctx, &message); err != nil {
				handler.logger.Error("failure marshalling and publishing message", zap.Error(err))
			}
		}
	}
}

func (handler *PricesHandler) marshalAndPublish(ctx context.Context, message *model.EthPriceMessage) error {
	messageBytes, err := json.Marshal(message)
	if err != nil {
		return err
	}

	if err := handler.topic.Publish(ctx, messageBytes); err != nil {
		return err
	}

	return nil
}

func (handler *PricesHandler) SubscribeToEthPriceMessages(ctx context.Context) {
	for {
		msg, err := handler.subscriber.Next(ctx)
		if err != nil {
			// Assuming an error is a temporary failure
			// Should be checked, fatal may be more appropriate
			handler.logger.Error("failed to get next message", zap.Error(err))
			continue
		}
		if msg.ReceivedFrom.String() == handler.hostID.String() {
			continue
		}

		var message model.EthPriceMessage
		if err := json.Unmarshal(msg.GetData(), &message); err != nil {
			// This could happen if we get a message of different structure
			// Thus just a warning since no need to handle such messages
			handler.logger.Warn("failed to unmarshal message", zap.Error(err))
			continue
		}

		if !message.IsSignedByNode(handler.hostID.String()) {
			sig, err := ecdsa.SignASN1(rand.Reader, handler.ecdsaKey, []byte(message.Price.String()))
			if err != nil {
				handler.logger.Error("failure signing message", zap.Error(err))
				continue
			}

			message.Signatures[model.NodeID(handler.hostID.String())] = model.PayloadSignature(sig)

			handler.logger.Debug("signed received message",
				zap.String("from", msg.ReceivedFrom.String()),
				zap.String("message_price", message.Price.String()),
				zap.Int("number_of_signatures", len(message.Signatures)))
		}

		if len(message.Signatures) < 3 {
			if err := handler.marshalAndPublish(ctx, &message); err != nil {
				handler.logger.Error("failure marshalling and publishing message", zap.Error(err))
			}
			continue
		}

		tx, err := handler.db.BeginTxx(ctx, nil)
		if err != nil {
			handler.logger.Error("failed to begin transaction", zap.Error(err))
			continue
		}

		// The most strict transaction isolation to make sure only one node can create price entry in the table at a time.
		// Might be an overkill, meaning less strict isolation could suffice, to check.
		if err := database.SetTxSerializable(ctx, tx); err != nil {
			handler.logger.Error("failed to set transaction isolation level", zap.Error(err))
			continue
		}

		dbEthPrice, err := database.GetMostRecentEthPrice(ctx, tx)
		if err != nil && err.Error() != "no prices found" {
			handler.logger.Error("failed to get most recent eth price from the database", zap.Error(err))
			tx.Rollback()
			continue
		}

		// dbEthPrice == nil means that there are no entries in the database, thus we may go ahead and create one
		if dbEthPrice != nil && !dbEthPrice.CreatedAt.Before(time.Now().UTC().Add(-30*time.Second)) {
			if err := handler.marshalAndPublish(ctx, &message); err != nil {
				handler.logger.Error("failure marshalling and publishing message", zap.Error(err))
			}
			tx.Rollback()
			continue
		}

		if err := database.InsertEthPrice(ctx, tx, message.Price); err != nil {
			// another node is writing to the table at the same time if the error contains the string below - so all good
			if !strings.Contains(err.Error(), "could not serialize access") {
				handler.logger.Error("failed to insert new eth price to the database", zap.Error(err))
			}
			tx.Rollback()
			continue
		}

		if err := tx.Commit(); err != nil {
			// as above
			if !strings.Contains(err.Error(), "could not serialize access") {
				handler.logger.Error("failed to commit transaction", zap.Error(err))
			}
			continue
		}

		handler.logger.Info("new eth price added",
			zap.String("message_price", message.Price.String()),
			zap.Int("number_of_signatures", len(message.Signatures)))
	}
}
