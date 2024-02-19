package global

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strconv"
)

type ExchangeRate struct {
	Data struct {
		Currency string            `json:"currency"`
		Rates    map[string]string `json:"rates"`
	} `json:"data"`
}

// Fetch eth usd price from coinbase
func GetETHPrice() (float64, error) {
	url := "https://api.coinbase.com/v2/exchange-rates?currency=ETH"

	response, err := http.Get(url)
	if err != nil {
		return 0, err
	}
	defer response.Body.Close()

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return 0, err
	}

	var cgResponse ExchangeRate
	err = json.Unmarshal(body, &cgResponse)
	if err != nil {
		return 0, err
	}

	val := cgResponse.Data.Rates["USD"]
	usdPrice, err := strconv.ParseFloat(val, 64)
	return usdPrice, err
}
