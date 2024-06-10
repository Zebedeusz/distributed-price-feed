package database

import (
	"context"
	"distributed-price-feed/config"
	"distributed-price-feed/model"
	"errors"
	"fmt"
	"time"

	"github.com/shopspring/decimal"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

func ConnectToDatabase(config config.Config) (*sqlx.DB, error) {
	// TODO: use SSL certificates
	return sqlx.Connect("postgres", fmt.Sprintf("dbname=postgres user=postgres sslmode=disable host=%s", config.DatabaseHost))
}

func GetMostRecentEthPrice(ctx context.Context, tx *sqlx.Tx) (*model.EthPrice, error) {
	var prices []model.EthPrice

	err := tx.SelectContext(ctx, &prices, "select * from eth_prices order by createdAt desc limit 1;")
	if err != nil {
		return nil, err
	}
	if len(prices) == 0 {
		return nil, errors.New("no prices found")
	}
	return &prices[0], nil
}

func InsertEthPrice(ctx context.Context, tx *sqlx.Tx, price decimal.Decimal) error {
	_, err := tx.ExecContext(ctx, "INSERT INTO eth_prices (price, createdAt) VALUES ($1,$2);", price, time.Now().UTC())
	return err
}

func SetTxSerializable(ctx context.Context, tx *sqlx.Tx) error {
	rows, err := tx.QueryContext(ctx, "SET TRANSACTION ISOLATION LEVEL SERIALIZABLE;")
	if rows != nil {
		rows.Close()
	}
	return err
}
