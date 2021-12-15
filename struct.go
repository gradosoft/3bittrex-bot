package main

//MarketInfo is an AssetPair info. First coin - Base, second coin - Quoted
type MarketInfo struct {
	MarketName string //Revert pair name, eg. "USD-BTC"
	BaseCoin   string //eg. "ETH" in case ETH-USD
	QuotedCoin string //eg. "USD" in case ETH-USD
	BaseSize   int    //scaling decimal places for volume ETH
	QuotePrice int    //scaling decimal places for price USD
}

//OrderBook fill from getOrderBook(), [0]-Price, [1]-Volume
type OrderBook struct {
	Asks [][2]float64 //[0]-Price, [1]-Volume
	Bids [][2]float64 //[0]-Price, [1]-Volume
}

//Responces is a standart Bittrex responce interface
type Responces struct {
	Success bool                     `json:"success"`
	Message string                   `json:"message"`
	Result  []map[string]interface{} `json:"result"`
}

//Ticker for LTicker data
type Ticker struct {
	Name           string //e.g. "BTC-LTC"
	Bid            float64
	Ask            float64
	OpenBuyOrders  int
	OpenSellOrders int
}

//RawOrderBook fill from getOrderBook()
type RawOrderBook struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Result  struct {
		Buy  []map[string]float64 `json:"buy"`
		Sell []map[string]float64 `json:"sell"`
	} `json:"result"`
}
