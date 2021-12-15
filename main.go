package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

//APIKEY contain KEY for Bittrex
const APIKEY = "Place-your-APIKEY-from-Bittrex-here"

//APISECRET contain private key for Bittrex
const APISECRET = "Place-your-APISECRET-from-Bittrex-here"

//TAXRATE contains current Fee for each order
const TAXRATE = 0.25

//Amount contains user-defined min/max rate value
var Amount = make(map[string][2]float64)

//AssetInfo contains info about All Pairs
var AssetInfo = make(map[string]*MarketInfo)

//Blacklist contains []string, eg. ["BTCLYKKE","EOSUSD"]
var Blacklist []string

//Coins contain Base coins
var Coins = []string{"USD", "USDT"}

//LOrderBook fill from wsOrderBook() and contain actual info about all order books
//var LOrderBook = make(map[string]*OrderBook)

//LTicker e.g ["BTC/USD"]{"USD-BTC", 8152.45, 8155,67, 45, 36}
var LTicker = make(map[string]*Ticker)

//MX is a mutex for LTicker
var MX sync.Mutex

//LockTrade for lock trade
var LockTrade = false

//MXlock is a mutex for LockTrade
var MXlock sync.Mutex

func main() {

	//Create log.txt to current directory
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		log.Fatal(err)
	}

	f, err := os.OpenFile(dir+string(filepath.Separator)+"log.txt", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}
	defer f.Close()

	//Multi log for file and console
	mw := io.MultiWriter(os.Stdout, f)
	log.SetOutput(mw)
	//Add microseconds to output
	log.SetFlags(log.Ldate | log.Lmicroseconds)

	//Preload data and structures
	fmt.Println("Preload data from Bittrex...")
	Blacklist = getBlackList("blacklist.json")
	Amount = getAmountMinMax("amount.json")
	AssetInfo = getMarketInfo()
	allSymbols := getAllSymbols(AssetInfo)
	tripleSymbols := getTripleSymbols(allSymbols)


	//Start
	fmt.Println("Hello, I`m Bittrex Bot!")


	for {
		fmt.Println(time.Now().Format("2 Jan 15:04:05"), "Triple Bittrex bot working...")

		t0 := time.Now().UnixNano() / int64(time.Second) //Start
		Start(tripleSymbols)
		t1 := time.Now().UnixNano() / int64(time.Second) //End
		fmt.Printf("Duration: %d sec\n", t1-t0)

		time.Sleep(10 * time.Second)
	}

}
