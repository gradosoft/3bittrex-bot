package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"os"
	"strings"
	"time"
)

//import "encoding/base64"

//strContains check occurrence
func strContains(slice []string, item string) bool {

	set := make(map[string]struct{}, len(slice))
	for _, s := range slice {
		set[s] = struct{}{}
	}
	_, ok := set[item]
	return ok
	//Example
	//s := []string{"a", "b"}
	//s1 := "a"
	//fmt.Println(strContains(s, s1))
}

//trnFloat trim float number
func trnFloat(val float64, prec int) float64 {

	rounder := math.Floor(val * math.Pow(10, float64(prec)))

	return rounder / math.Pow(10, float64(prec))
}

//qtyDecimalPlaces return qty of places after comma
func qtyDecimalPlaces(num float64) int {

	str := fmt.Sprintf("%g", num) //Necessary digits only, trim zero

	if strings.Contains(str, ".") { //If exists part after "."
		parts := strings.Split(str, ".")
		return len(parts[1])
	}

	return 0
}

func getStartEnd(coin string, all [][2]string) ([][2]string, [][2]string) { //Start and End chains

	l := len(all)

	start := [][2]string{}
	end := [][2]string{}

	for i := 0; i < l; i++ {

		if all[i][1] == coin || all[i][0] == coin {

			start = append(start, all[i])
		}

		if all[i][1] == coin {

			end = append(end, all[i])
		}

	}

	return start, end
}

//getAmountMinMax read amount.json
func getAmountMinMax(pFile string) map[string][2]float64 {

	amount := make(map[string][2]float64)
	configFile, openErr := os.Open(pFile)
	if openErr != nil {
		log.Fatal(openErr)
	}
	defer configFile.Close()

	jsonParser := json.NewDecoder(configFile)
	jsonParser.Decode(&amount)

	return amount
}

//getBlackList read blacklist.json
func getBlackList(loadFile string) []string {

	var blacklist []string
	configFile, openErr := os.Open(loadFile)
	if openErr != nil {
		log.Fatal(openErr)
	}
	defer configFile.Close()

	jsonParser := json.NewDecoder(configFile)
	jsonParser.Decode(&blacklist)

	return blacklist
}

func getAllSymbols(pAssetInfo map[string]*MarketInfo) [][2]string {

	var all [][2]string

	for _, p := range pAssetInfo {
		symbol := p.BaseCoin + p.QuotedCoin

		if strContains(Blacklist, symbol) == false { //If Symbol is not in blacklist
			pair := [2]string{p.BaseCoin, p.QuotedCoin}
			all = append(all, pair)
		}
	}

	return all
}

//Current time
func currentTimestamp() int64 {

	return int64(time.Nanosecond) * time.Now().UnixNano() / int64(time.Millisecond) //- 2000
}

//Signature
func getSignature(str string) string {

	key := []byte(APISECRET)
	h := hmac.New(sha512.New, key)
	h.Write([]byte(str))
	//return base64.StdEncoding.EncodeToString(h.Sum(nil))
	return hex.EncodeToString(h.Sum(nil))
}

// getSha256 creates a sha256 hash for given []byte --Kraken
func getSha256(input []byte) []byte {
	sha := sha256.New()
	sha.Write(input)
	return sha.Sum(nil)
}

// getHMacSha512 creates a hmac hash with sha512 --Kraken
func getHMacSha512(message, secret []byte) []byte {
	mac := hmac.New(sha512.New, secret)
	mac.Write(message)
	return mac.Sum(nil)
}

// See https://www.kraken.com/help/api#general-usage for more information --Kraken
func createSignature(urlPath string, values string, nonce string) string {
	secret, _ := base64.StdEncoding.DecodeString(APISECRET)
	shaSum := getSha256([]byte(nonce + values))
	macSum := getHMacSha512(append([]byte(urlPath), shaSum...), secret)
	return base64.StdEncoding.EncodeToString(macSum)
}
