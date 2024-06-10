package coingeckoclient

import (
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/shopspring/decimal"
)

const EthPriceCoingeckoURL = "https://api.coingecko.com/api/v3/simple/price?ids=ethereum&vs_currencies=usd"

type CoingeckoEthereumPrice struct {
	Ethereum CoingeckoPriceInUsd `json:"ethereum"`
}

type CoingeckoPriceInUsd struct {
	Usd float32 `json:"usd"`
}

func GetEthPrice() (*decimal.Decimal, error) {
retry:
	r, err := http.Get(EthPriceCoingeckoURL)
	if err != nil {
		return nil, err
	}
	defer r.Body.Close()

	if r.StatusCode == 429 {
		waitTime, err := time.ParseDuration(r.Header.Get("Retry-After") + "s")
		if err != nil {
			return nil, err
		}
		time.Sleep(waitTime)
		goto retry
	}

	// valid payload example: {"ethereum":{"usd":2810.52}}

	jsonDataFromHttp, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}

	var price CoingeckoEthereumPrice
	if err := json.Unmarshal([]byte(jsonDataFromHttp), &price); err != nil {
		return nil, err
	}

	priceDecimal := decimal.NewFromFloat32(price.Ethereum.Usd)

	return &priceDecimal, nil
}
