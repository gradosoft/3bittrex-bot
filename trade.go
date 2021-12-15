package main

import (
	"fmt"
	"log"
	"strings"
	"time"
)

func planBSS(pCoin string, pSym1 string, pSym2 string, pSym3 string) string {
	var startPrice, endPrice float64
	ok := "OK"
	var profit float64
	rate := Amount[pCoin] //rate[0] - min, rate[1] - max
	var price1, qty1, price2, qty2, price3, qty3 float64

	//Fill  order books
	book1 := getOrderBook(AssetInfo[pSym1].MarketName) //Use reverse symbol
	book2 := getOrderBook(AssetInfo[pSym2].MarketName)
	book3 := getOrderBook(AssetInfo[pSym3].MarketName)

	if len(book1.Asks) < 5 || len(book2.Bids) < 5 || len(book3.Bids) < 5 {
		fmt.Printf("%s -> %s -> %s: %s. %s\n", pSym1, pSym2, pSym3, "NO 5 ORDERS", "BSS")
		return "NO 5 ORDERS"
	}

	//Start values for price and qty
	price1 = book1.Asks[0][0]                  //first price for ask
	qty1 = book1.Asks[0][1]                    //first size for ask
	price2 = book2.Bids[1][0]                  //second price for bid
	qty2 = book2.Bids[0][1] + book2.Bids[1][1] //first + second size
	price3 = book3.Bids[1][0]                  //second price for bid
	qty3 = book3.Bids[0][1] + book3.Bids[1][1] //first + second size

	//Calculate Max Rate
	eqvRate1 := price1 * qty1
	eqvRate2 := qty2 * price1
	eqvRate3 := qty3 * price3

	//Calculate for minAmount, increase price and qty
	for i := 1; i < len(book1.Asks); i++ {
		if eqvRate1 < rate[0] {
			price1 = book1.Asks[i][0]
			qty1 += book1.Asks[i][1]
			eqvRate1 = price1 * qty1
		} else {
			break
		}
	}

	for i := 2; i < len(book2.Bids); i++ { //From third item
		if eqvRate2 < rate[0] {
			price2 = book2.Bids[i][0]
			qty2 += book2.Bids[i][1]
			eqvRate2 = qty2 * price1
		} else {
			break
		}
	}

	for i := 2; i < len(book3.Bids); i++ { //From third item
		if eqvRate3 < rate[0] {
			price3 = book3.Bids[i][0]
			qty3 += book3.Bids[i][1]
			eqvRate3 = qty3 * price3
		} else {
			break
		}
	}

	if eqvRate1 < rate[0] || eqvRate2 < rate[0] || eqvRate3 < rate[0] {
		fmt.Printf("%s -> %s -> %s: %s. %s\n", pSym1, pSym2, pSym3, "NO VOLUME", "BSS")
		return "NO VOLUME"
	}

	maxRate := rate[1] //Max Rate user defined

	if maxRate > eqvRate1 {
		maxRate = eqvRate1 //Set maxRate
	}

	if maxRate > eqvRate2 {
		maxRate = eqvRate2 //Set maxRate
	}

	if maxRate > eqvRate3 {
		maxRate = eqvRate3 //Set maxRate
	}

	// Maximal rate -20%
	maxRate -= maxRate / 100 * 20
	if maxRate < rate[0] {
		maxRate = rate[0]
	}

	//Calculate Profit
	startPrice = price1
	endPrice = price2 * price3

	profit = (endPrice - startPrice) * 100.0 / startPrice //calculate procents
	ok = fmt.Sprintf("%.2f %%", profit)

	if profit  > 0.95 {

		/*
		//Check for multithreading lock
		if LockTrade == true {
			log.Printf("BSS: %s -> %s -> %s = %.2f%%...Trade is Busy. Return\n", pSym1, pSym2, pSym3, profit)
			return ok
		}

		//Lock trade operation
		MXlock.Lock()
		LockTrade = true
		MXlock.Unlock()
		*/

		log.Printf("Currency: %s, minAmount = %f, maxAmount = %f, Plan = BSS\n", pCoin, rate[0], maxRate)
		log.Printf("%s -> %s -> %s = %.2f%%\n", pSym1, pSym2, pSym3, profit)
		log.Printf("%s: Price: %.8f, QTY: %.8f = %f %s\n", pSym1, price1, qty1, eqvRate1, pCoin)
		log.Printf("%s: Price: %.8f, QTY: %.8f = %f %s\n", pSym2, price2, qty2, eqvRate2, pCoin)
		log.Printf("%s: Price: %.8f, QTY: %.8f = %f %s\n\n", pSym3, price3, qty3, eqvRate3, pCoin)

		//Calculate QTY, tax and presicion for order
		prec1 := AssetInfo[pSym1].BaseSize //Precision as BaseCoin in pSym1
		prec2 := AssetInfo[pSym2].BaseSize //Precision as BaseCoin in pSym2
		prec3 := AssetInfo[pSym3].BaseSize //Precision as BaseCoin in pSym3

		trnQtyAsk1 := trnFloat(maxRate/price1, prec1) //Step 1
		tax1 := trnQtyAsk1 / 100 * TAXRATE
		trnQtyAsk1Tax := trnFloat(trnQtyAsk1-tax1, prec2) //Step 2

		trnQtyBid2 := trnFloat(trnQtyAsk1Tax*price2, prec3) //Step 3
		tax2 := trnQtyBid2 / 100 * TAXRATE
		trnQtyBid2Tax := trnFloat(trnQtyBid2-tax2, prec3)

		trnQtyBid3 := trnFloat(trnQtyBid2Tax*price3, prec3) //Step 3
		tax3 := trnQtyBid3 / 100 * TAXRATE
		trnQtyBid3Tax := trnFloat(trnQtyBid3-tax3, 8)

		fee := maxRate / 100 * (TAXRATE * 3)

		log.Printf("%s: Buy %.8f for %.8f %s\n", pSym1, trnQtyAsk1, maxRate, pCoin)
		log.Printf("%s: Sell %.8f for %.8f %s\n", pSym2, trnQtyAsk1, trnQtyBid2, AssetInfo[pSym3].BaseCoin)
		log.Printf("%s: Sell %.8f for %.8f %s\n", pSym3, trnQtyBid2Tax, trnQtyBid3Tax, pCoin)
		log.Printf("Fee = %.8f, Earnings = %.8f %s\n\n", fee, trnQtyBid3Tax-maxRate, pCoin)

		//Set Market Orders for Best Price
		coin1 := AssetInfo[pSym1].BaseCoin
		coin2 := AssetInfo[pSym2].BaseCoin
		coin3 := AssetInfo[pSym3].BaseCoin

		inc := profit/4 - TAXRATE //Volume for increase/decrease price

		bestRate := price1 + (price1 / 100.0 * inc) // Increase price
		prcPrice := AssetInfo[pSym1].QuotePrice
		bestRate = trnFloat(bestRate, prcPrice) //Precision as BasePriceInc in pSym1

		//Check APIKEY, APISECRET
		if len(APIKEY) != 32 || len(APISECRET) != 32 {
		fmt.Println("Sorry, you should set the variables APIKEY and APISECRET from Binance...")		
		fmt.Println("Limit Orders don't set. Press the Enter for continue or CTRL-C for Exit.")
		fmt.Scanln() // wait for Enter Key
		return "APIKEY or APISECRET don't set"
		}

		log.Printf("Step 1: Buy LIMIT %.8f %s, BestPrice = %.8f...\n", trnQtyAsk1, coin1, bestRate)

		if err := setOrder(pSym1, "LIMIT", "BUY", bestRate, trnQtyAsk1); err != nil {
			log.Println(err)

			if fok := strings.Contains(err.Error(), "FILL_OR_KILL_NOT_MET"); fok == true {
				log.Printf("ERROR: Orders Chains don`t complete!\n\n")

				//Unlock trade operation
				MXlock.Lock()
				LockTrade = false
				MXlock.Unlock()

				return "SET ORDER ERROR"
			}
		}

		bestRate = price2 - (price2 / 100.0 * inc) // Decrease price
		prcPrice = AssetInfo[pSym2].QuotePrice
		bestRate = trnFloat(bestRate, prcPrice) //Precision as BasePriceInc in pSym2

		log.Printf("Step 2: Sell MARKET %.8f %s, BestPrice = %.8f...\n", trnQtyAsk1, coin2, bestRate)

		if err := setOrder(pSym2, "MARKET", "SELL", bestRate, trnQtyAsk1); err != nil {
			log.Println(err)
		}

		bestRate = price3 - (price3 / 100.0 * inc) //Decrease price
		prcPrice = AssetInfo[pSym3].QuotePrice
		bestRate = trnFloat(bestRate, prcPrice) //Precision as BasePriceInc in pSym3


		log.Printf("Step 3: Sell MARKET %.8f %s, BestPrice = %.8f...\n", trnQtyBid2Tax, coin3, bestRate)

		if err := setOrder(pSym3, "MARKET", "SELL", bestRate, trnQtyBid2Tax); err != nil {
			log.Println(err)
		}

		log.Printf("Order Chains Complete!\n\n\n")

		/*
		//Unlock trade operation
		MXlock.Lock()
		LockTrade = false
		MXlock.Unlock()
		*/

	}

	fmt.Printf("%s -> %s -> %s: %s. %s\n", pSym1, pSym2, pSym3, ok, "BSS")
	return ok
}

func planBBS(pCoin string, pSym1 string, pSym2 string, pSym3 string) string {
	var startPrice, endPrice float64
	ok := "OK"
	var profit float64
	rate := Amount[pCoin] //rate[0] - min, rate[1] - max
	var price1, qty1, price2, qty2, price3, qty3 float64

	//Fill  order books
	book1 := getOrderBook(AssetInfo[pSym1].MarketName) //Use reverse symbol
	book2 := getOrderBook(AssetInfo[pSym2].MarketName)
	book3 := getOrderBook(AssetInfo[pSym3].MarketName)

	if len(book1.Asks) < 5 || len(book2.Asks) < 5 || len(book3.Bids) < 5 {
		fmt.Printf("%s -> %s -> %s: %s. %s\n", pSym1, pSym2, pSym3, "NO 5 ORDERS", "BBS")
		return "NO 5 ORDERS"
	}

	//Start values for price and qty
	price1 = book1.Asks[0][0]                  //first price for ask
	qty1 = book1.Asks[0][1]                    //first for size
	price2 = book2.Asks[1][0]                  //second price for ask
	qty2 = book2.Asks[0][1] + book2.Asks[1][1] //1 + 2 size
	price3 = book3.Bids[1][0]                  //second price for bid
	qty3 = book3.Bids[0][1] + book3.Bids[1][1] //1 + 2size

	//Calculate Max Rate
	eqvRate1 := price1 * qty1
	eqvRate2 := qty2 * price3
	eqvRate3 := qty3 * price3

	//Calculate for minAmount, increase price and qty
	for i := 1; i < len(book1.Asks); i++ {
		if eqvRate1 < rate[0] {
			price1 = book1.Asks[i][0]
			qty1 += book1.Asks[i][1]
			eqvRate1 = price1 * qty1
		} else {
			break
		}
	}

	for i := 2; i < len(book2.Asks); i++ { //From 3-th item
		if eqvRate2 < rate[0] {
			price2 = book2.Asks[i][0]
			qty2 += book2.Asks[i][1]
			eqvRate2 = qty2 * price3
		} else {
			break
		}
	}

	for i := 2; i < len(book3.Bids); i++ { //From 3-th item
		if eqvRate3 < rate[0] {
			price3 = book3.Bids[i][0]
			qty3 += book3.Bids[i][1]
			eqvRate3 = qty3 * price3
		} else {
			break
		}
	}

	if eqvRate1 < rate[0] || eqvRate2 < rate[0] || eqvRate3 < rate[0] {
		fmt.Printf("%s -> %s -> %s: %s. %s\n", pSym1, pSym2, pSym3, "NO VOLUME", "BBS")
		return "NO VOLUME"
	}

	maxRate := rate[1] //Max Rate user defined

	if maxRate > eqvRate1 {
		maxRate = eqvRate1 //Set maxRate
	}

	if maxRate > eqvRate2 {
		maxRate = eqvRate2 //Set maxRate
	}

	if maxRate > eqvRate3 {
		maxRate = eqvRate3 //Set maxRate
	}

	// Maximal rate -20%
	maxRate -= maxRate / 100 * 20
	if maxRate < rate[0] {
		maxRate = rate[0]
	}

	//Calculate Profit
	startPrice = price1
	endPrice = price3 / price2

	profit = (endPrice - startPrice) * 100.0 / startPrice //calculate procents
	ok = fmt.Sprintf("%.2f %%", profit)

	if profit > 0.95 {

		//Check for multithreading lock
		if LockTrade == true {
			log.Printf("BBS: %s -> %s -> %s = %.2f%%...Trade is Busy. Return\n", pSym1, pSym2, pSym3, profit)
			return ok
		}

		//Lock trade operation
		MXlock.Lock()
		LockTrade = true
		MXlock.Unlock()

		log.Printf("Currency: %s, minAmount = %f, maxAmount = %f, Plan = BBS\n", pCoin, rate[0], maxRate)
		log.Printf("%s -> %s -> %s = %.2f%%\n", pSym1, pSym2, pSym3, profit)
		log.Printf("%s: Price: %.8f, QTY: %.8f = %f %s\n", pSym1, price1, qty1, eqvRate1, pCoin)
		log.Printf("%s: Price: %.8f, QTY: %.8f = %f %s\n", pSym2, price2, qty2, eqvRate2, pCoin)
		log.Printf("%s: Price: %.8f, QTY: %.8f = %f %s\n\n", pSym3, price3, qty3, eqvRate3, pCoin)

		//Calculate QTY, tax and presicion for order
		prec1 := AssetInfo[pSym1].BaseSize //Precision as BaseCoin in pSym1
		prec2 := AssetInfo[pSym2].BaseSize //Precision as BaseCoin in pSym2
		prec3 := AssetInfo[pSym3].BaseSize //Precision as BaseCoin in pSym3

		trnQtyAsk1 := trnFloat(maxRate/price1, prec1) //Step 1
		tax1 := trnQtyAsk1 / 100 * TAXRATE
		trnQtyAsk1Tax := trnFloat(trnQtyAsk1-tax1, prec2) //Step 2 for trnQtyAsk2 with tax

		trnQtyAsk2 := trnFloat(trnQtyAsk1Tax/price2, prec3) //Step 2 with tax
		//trnQtyAsk2 := trnFloat(trnQtyAsk1/price2, prec3) //Step 2
		tax2 := trnQtyAsk2 / 100 * TAXRATE
		trnQtyAsk2Tax := trnFloat(trnQtyAsk2-tax2, prec3) //Step3

		trnQtyBid3 := trnFloat(trnQtyAsk2Tax*price3, prec3) //Step 3
		tax3 := trnQtyBid3 / 100 * TAXRATE
		trnQtyBid3Tax := trnFloat(trnQtyBid3-tax3, 8)

		fee := maxRate / 100 * (TAXRATE * 3)

		//Set Market Orders for Best Price
		coin1 := AssetInfo[pSym1].BaseCoin
		coin2 := AssetInfo[pSym2].BaseCoin
		coin3 := AssetInfo[pSym3].BaseCoin

		log.Printf("%s: Buy %.8f for %.8f %s\n", pSym1, trnQtyAsk1, maxRate, pCoin)
		log.Printf("%s: Buy %.8f for %.8f %s\n", pSym2, trnQtyAsk2, trnQtyAsk1, coin1)
		log.Printf("%s: Sell %.8f for %.8f %s\n", pSym3, trnQtyAsk2Tax, trnQtyBid3Tax, pCoin)
		log.Printf("Fee = %.8f, Earnings = %.8f %s\n\n", fee, trnQtyBid3Tax-maxRate, pCoin)

		inc := profit/4 - TAXRATE //Volume for increase/decrease price

		bestRate := price1 + (price1 / 100.0 * inc) // Increase price
		prcPrice := AssetInfo[pSym1].QuotePrice
		bestRate = trnFloat(bestRate, prcPrice) //Precision as BasePriceInc in pSym1

		log.Printf("Step 1: Buy LIMIT %.8f %s, BestPrice = %.8f...\n", trnQtyAsk1, coin1, bestRate)

		if err := setOrder(pSym1, "LIMIT", "BUY", bestRate, trnQtyAsk1); err != nil {
			log.Println(err)

			if fok := strings.Contains(err.Error(), "FILL_OR_KILL_NOT_MET"); fok == true {
				log.Printf("ERROR: Orders Chains don`t complete!\n\n")

				//Unlock trade operation
				MXlock.Lock()
				LockTrade = false
				MXlock.Unlock()

				return "SET ORDER ERROR"
			}
		}

		bestRate = price2 + (price2 / 100.0 * inc) // Increase price
		prcPrice = AssetInfo[pSym2].QuotePrice
		bestRate = trnFloat(bestRate, prcPrice) //Precision as BasePriceInc in pSym2

		log.Printf("Step 2: Buy LIMIT %.8f %s, BestPrice = %.8f...\n", trnQtyAsk2, coin2, bestRate)

		if err := setOrder(pSym2, "LIMIT", "BUY", bestRate, trnQtyAsk2); err != nil {
			log.Println(err)
			//wallet := getWallet()
			//log.Println("Wallet:", wallet)
			//log.Printf("ERROR: Orders Chains don`t complete!\n\n")
			//return "SET ORDER ERROR"
		}

		bestRate = price3 - (price3 / 100.0 * inc) //Decrease price
		prcPrice = AssetInfo[pSym3].QuotePrice
		bestRate = trnFloat(bestRate, prcPrice) //Precision as BasePriceInc in pSym3

		/*
			//Check Balance
			wallet := getWallet()
			if wallet[coin3] != 0 && strContains(Coins, coin3) == false {
				trnQtyAsk2Tax = trnFloat(wallet[coin3], prec3)
			}
		*/

		log.Printf("Step 3: Sell MARKET %.8f %s, BestPrice = %.8f...\n", trnQtyAsk2Tax, coin3, bestRate)

		if err := setOrder(pSym3, "MARKET", "SELL", bestRate, trnQtyAsk2Tax); err != nil {
			log.Println(err)
			//log.Println("Wallet:", wallet)
			//log.Printf("ERROR: Orders Chains don`t complete!\n\n")
			//return "SET ORDER ERROR"
		}

		log.Printf("Order Chains Complete!\n\n\n")

		//Unlock trade operation
		MXlock.Lock()
		LockTrade = false
		MXlock.Unlock()

	}

	fmt.Printf("%s -> %s -> %s: %s. %s\n", pSym1, pSym2, pSym3, ok, "BBS")
	return ok
}

//Input [][2]string all pairs, Output [][5]string as []["BSS", "USD", "VEEUSD", "VEEBTC", "BTCUSD"]
func getTripleSymbols(pAll [][2]string) [][5]string {

	var result [][5]string
	var coin string

	for c := 0; c < len(Coins); c++ {
		coin = Coins[c]

		start, end := getStartEnd(coin, pAll)

		lenStart := len(start)
		lenEnd := len(end)
		lenAll := len(pAll)

		for s := 0; s < lenStart; s++ {
			for e := 0; e < lenEnd; e++ {
				for a := 0; a < lenAll; a++ {
					//Buy-Sell-Sell
					if (pAll[a][0] == start[s][0]) && (pAll[a][1] == end[e][0]) {
						if pAll[a][0] != coin && pAll[a][1] != coin {

							pair1 := start[s][0] + "-" + start[s][1]
							pair2 := pAll[a][0] + "-" + pAll[a][1]
							pair3 := end[e][0] + "-" + end[e][1]

							item := [5]string{"BSS", coin, pair1, pair2, pair3}
							result = append(result, item)
						}
					}
					//Buy-Buy-Sell
					if (pAll[a][1] == start[s][0]) && (pAll[a][0] == end[e][0]) {
						if pAll[a][0] != coin && pAll[a][1] != coin {

							pair1 := start[s][0] + "-" + start[s][1]
							pair2 := pAll[a][0] + "-" + pAll[a][1]
							pair3 := end[e][0] + "-" + end[e][1]

							item := [5]string{"BBS", coin, pair1, pair2, pair3}
							result = append(result, item)
						}
					}
				}
			}
		}

	}

	return result
}

//Start Bittrex Bot
func Start(pTripleSym [][5]string) {

	lenList := len(pTripleSym)
	for i := 0; i < lenList; i++ {
		plan := pTripleSym[i][0]
		coin := pTripleSym[i][1]
		pair1 := pTripleSym[i][2]
		pair2 := pTripleSym[i][3]
		pair3 := pTripleSym[i][4]

		if plan == "BSS" {
			planBSS(coin, pair1, pair2, pair3)
			time.Sleep(50 * time.Millisecond)
		}

		if plan == "BBS" {
			//go planBBS(coin, pair1, pair2, pair3)
			//time.Sleep(50 * time.Millisecond)
		}

	}

}
