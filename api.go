package main

import (
	"bytes"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"
)

//Connections and returns body responce
func connPublic(pURL string) ([]byte, int) { //Just GET responce without signing

	spaceClient := http.Client{
		Timeout: time.Second * 10, //Max of 10 secs
	}

	req, reqErr := http.NewRequest(http.MethodGet, pURL, nil)
	if reqErr != nil {
		log.Fatal(reqErr)
	}

	req.Header.Set("User-Agent", "Bittrex Bot v.0.1")

	res, resErr := spaceClient.Do(req)
	if resErr != nil {
		log.Fatal(resErr)
	}
	defer res.Body.Close()

	body, bodyErr := ioutil.ReadAll(res.Body)
	if bodyErr != nil {
		log.Fatal(bodyErr)
	}

	if res.StatusCode != 200 {
		log.Println(pURL, res.Status)
		log.Println(string(body))
	}

	return body, res.StatusCode
}

//Bittrex API
func getMarketInfo() map[string]*MarketInfo { //GET /api/v1.1/public/getmarkets

	reqURL := "https://api.bittrex.com/v3/markets"
	body, _ := connPublic(reqURL)

	raw := []map[string]interface{}{}

	if jsonErr := json.Unmarshal(body, &raw); jsonErr != nil {
		log.Fatal(jsonErr)
	}

	asset := make(map[string]*MarketInfo)

	for _, v := range raw {

		item := MarketInfo{}

		if v["status"].(string) == "ONLINE" {

			item.MarketName = v["quoteCurrencySymbol"].(string) + "-" + v["baseCurrencySymbol"].(string) //Revert symbol, eg. BTC-USD for API v.1
			item.BaseCoin = v["baseCurrencySymbol"].(string)
			item.QuotedCoin = v["quoteCurrencySymbol"].(string)
			item.BaseSize = 8
			item.QuotePrice = int(v["precision"].(float64))

			symbol := v["symbol"].(string) //Standart symbol, eg. BTC-USD
			asset[symbol] = &item
		}

	}

	//Correct
	//asset["BTC-USD"].QuotePrice = 3
	//asset["ETH-USD"].QuotePrice = 3

	return asset
}



//getAllTickers fill LTicker in realtime, but ask/bid updated very rarely
func getAllTickers() {

	reqURL := "https://api.bittrex.com/api/v1.1/public/getmarketsummaries"
	body, _ := connPublic(reqURL)

	raw := Responces{}
	if jsonErr := json.Unmarshal(body, &raw); jsonErr != nil {
		log.Fatal(jsonErr)
	}

	for _, v := range raw.Result {

		item := Ticker{}
		coin := strings.Split(v["MarketName"].(string), "-")

		item.Name = v["MarketName"].(string)
		item.Bid = v["Bid"].(float64)
		item.Ask = v["Ask"].(float64)
		item.OpenBuyOrders = int(v["OpenBuyOrders"].(float64))
		item.OpenSellOrders = int(v["OpenSellOrders"].(float64))

		symbol := coin[1] + "/" + coin[0]

		MX.Lock()
		LTicker[symbol] = &item
		MX.Unlock()
	}

}

// Return OrderBook struct ([0]-Price [1]-Volume)
func getOrderBook(symbol string) OrderBook { //GET /public/getorderbook + symbol (Revert)

	res := OrderBook{}

	reqURL := "https://api.bittrex.com/api/v1.1/public/getorderbook?type=both&market=" + symbol
	body, status := connPublic(reqURL)
	if status != 200 {
		log.Fatal("getOrderBook:", status)
	}

	raw := RawOrderBook{}
	if jsonErr := json.Unmarshal(body, &raw); jsonErr != nil {
		log.Fatal(jsonErr)
	}

	for _, v := range raw.Result.Buy {
		p := v["Rate"]
		v := v["Quantity"]
		item := [2]float64{p, v}
		res.Bids = append(res.Bids, item)
	}

	for _, v := range raw.Result.Sell {
		p := v["Rate"]
		v := v["Quantity"]
		item := [2]float64{p, v}
		res.Asks = append(res.Asks, item)
	}

	return res
}

//setOrder return order ID and error
func setOrder(pSym string, pType string, pSide string, pPrice float64, pSize float64) error {

	nonce := fmt.Sprintf("%d", time.Now().UnixNano()/int64(time.Millisecond))

	params := make(map[string]interface{})
	params["marketSymbol"] = pSym
	params["direction"] = pSide // BUY/SELL
	params["type"] = pType      // LIMIT, MARKET, CEILING_LIMIT, CEILING_MARKET
	params["quantity"] = pSize
	//params["ceiling"] = pPrice
	if pType == "LIMIT" {
		params["limit"] = pPrice //(optional, must be included for LIMIT orders and excluded for MARKET orders)
	}
	params["timeInForce"] = "FILL_OR_KILL" //GOOD_TIL_CANCELLED, IMMEDIATE_OR_CANCEL, FILL_OR_KILL, POST_ONLY_GOOD_TIL_CANCELLED, BUY_NOW
	//params["clientOrderId"] = "string uuid" //client-provided identifier for advanced order tracking (optional)
	//params["useAwards"] = true              //option to use Bittrex credits for the order (optional)

	postJSON, _ := json.Marshal(params)
	buffer := bytes.NewBuffer(postJSON)

	reqURL := "https://api.bittrex.com/v3/orders"
	req, err := http.NewRequest("POST", reqURL, buffer)
	if err != nil {
		log.Fatal(err)
	}

	spaceClient := http.Client{
		Timeout: time.Second * 9, //Max of 9 secs
	}

	//sha512 Api-Content-Hash
	sha := sha512.New()
	sha.Write(postJSON)
	contentHash := hex.EncodeToString(sha.Sum(nil))

	//Signature
	preSign := nonce + reqURL + "POST" + contentHash
	sign := getSignature(preSign)

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("API-Key", APIKEY)
	req.Header.Add("Api-Timestamp", nonce)
	req.Header.Add("Api-Content-Hash", contentHash)
	req.Header.Add("API-Signature", sign)

	res, resErr := spaceClient.Do(req)
	if resErr != nil {
		log.Fatal(resErr, "res:", res)
	}
	defer res.Body.Close()

	body, bodyErr := ioutil.ReadAll(res.Body)
	if bodyErr != nil {
		log.Fatal(bodyErr, "body:", string(body))
	}

	if res.StatusCode != 200 {
		log.Printf("HTTP Status: %s\n", res.Status)
	}

	if bytes.Contains(body, []byte("code")) == true { //If Error
		return errors.New(string(body))
	}

	return nil
}

func getWallet() map[string]float64 { //GET /0/private/Balance Return map["BTC"]0.0056432 for trade account

	wallet := make(map[string]float64)
	nonce := time.Now().UnixNano() / int64(time.Millisecond)

	params := make(map[string]interface{})
	params["nonce"] = nonce

	postJSON, _ := json.Marshal(params)
	buffer := bytes.NewBuffer(postJSON)

	reqURL := "https://api.kraken.com/0/private/Balance"
	req, err := http.NewRequest("POST", reqURL, buffer)
	if err != nil {
		log.Fatal(err)
	}

	spaceClient := http.Client{
		Timeout: time.Second * 5, //Max of 5 secs
	}

	signature := createSignature("/0/private/Balance", string(postJSON), fmt.Sprintf("%d", nonce))

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("API-Key", APIKEY)
	req.Header.Add("API-Sign", signature)

	res, resErr := spaceClient.Do(req)
	if resErr != nil {
		log.Fatal(resErr, "res:", res)
	}
	defer res.Body.Close()

	body, bodyErr := ioutil.ReadAll(res.Body)
	if bodyErr != nil {
		log.Fatal(bodyErr, "body:", string(body))
	}

	type Funds struct {
		Error  interface{}       `json:"error"`
		Result map[string]string `json:"result"`
	}

	raw := Funds{}

	if jsonErr := json.Unmarshal(body, &raw); jsonErr != nil {
		log.Fatal(jsonErr)
	}

	/*
		for k, v := range raw.Result {
			vol, _ := strconv.ParseFloat(v, 64)


				//Replace coin name as AltName if exists
				if coin, ok := AltName[k]; ok {
					wallet[coin] = vol
				} else {
					wallet[k] = vol
				}


		}
	*/

	return wallet

}
