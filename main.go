package main

import (
	"math/rand"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

func main() {
	rand.Seed(time.Now().UnixNano())
	amazonItemLookup("B07SKPNNKG")
}

type amazonBook struct {
	ASIN          string
	Title         string
	Author        string
	CoverImageURL string
	Rating        float32
	KindlePrice   float32
	HardPrice     float32
	PaperPrice    float32
}

var rxPrice = regexp.MustCompile(`\$(\d+(\.\d+)?)`)
var rxRating = regexp.MustCompile(`([\d.]+) `)

var userAgentStrings = [...]string{
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/42.0.2311.135 Safari/537.36 Edge/12.246",
	"Mozilla/5.0 (X11; CrOS x86_64 8172.45.0) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/51.0.2704.64 Safari/537.36",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_11_2) AppleWebKit/601.3.9 (KHTML, like Gecko) Version/9.0.2 Safari/601.3.9",
	"Mozilla/5.0 (Windows NT 6.1; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/47.0.2526.111 Safari/537.36",
	"Mozilla/5.0 (X11; Ubuntu; Linux x86_64; rv:15.0) Gecko/20100101 Firefox/15.0.1"}

var weblock = make(chan struct{}, 1)

// get a page from the web
func getPage(URL string) (*http.Response, error) {
	//get a lock on web
	weblock <- struct{}{}
	defer func() {
		go func() {
			time.Sleep(time.Duration(rand.Intn(5)) * time.Second) //introduce random delay
			<-weblock                                             //release lock so next operation can call web
		}()
	}()

	//client with timeout
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	//header
	request, err := http.NewRequest("GET", URL, nil)
	if err != nil {
		return nil, err
	}

	//set random user agent
	request.Header.Set("User-Agent", userAgentStrings[rand.Intn(len(userAgentStrings))])

	//return web response
	return client.Do(request)
}

func amazonItemLookup(ASIN string) (*amazonBook, error) {

	//get the item page
	response, err := getPage("https://www.amazon.com/dp/" + ASIN)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	//parse the document
	doc, err := goquery.NewDocumentFromReader(response.Body)
	if err != nil {
		return nil, err
	}

	return getBookDetails(doc)
}

func getBookDetails(doc *goquery.Document) (*amazonBook, error) {
	//make a book
	book := &amazonBook{}

	e := doc.Find("#dp-container").First()

	book.Title = strings.TrimSpace(e.Find("#title .a-size-extra-large").Text())
	book.Author = e.Find(".contributorNameID").First().Text()
	book.ASIN = e.Find(".contributorNameID").AttrOr("data-asin", "")
	book.CoverImageURL = e.Find(".frontImage").AttrOr("src", "")

	ratingmatch := rxRating.FindStringSubmatch(e.Find(".a-icon .a-icon-alt").Text())
	if len(ratingmatch) > 1 {
		rating, err := strconv.ParseFloat(ratingmatch[1], 32)
		if err == nil {
			book.Rating = float32(rating)
		}
	}

	kindlematch := rxPrice.FindStringSubmatch(e.Find(".kindle-price .a-color-price").Text())
	if len(kindlematch) > 1 {
		kindleprice, err := strconv.ParseFloat(kindlematch[1], 32)
		if err == nil {
			book.KindlePrice = float32(kindleprice)
		}
	}

	e.Find(".swatchElement").Each(func(_ int, x *goquery.Selection) {
		//type of media (kindle, hardcover, paperback, audiobook, etc)
		mediastring := x.Find(".a-button-text span").Text()
		mediastring = mediastring[0:strings.Index(mediastring, "\n")]

		//read the price
		pricestring := x.Find(".a-color-price").Text()
		if len(pricestring) == 0 {
			pricestring = x.Find(".a-color-secondary").Text()
		}
		pricematch := rxPrice.FindStringSubmatch(pricestring)

		if len(pricematch) > 1 {
			price, err := strconv.ParseFloat(pricematch[1], 32)

			if err == nil {
				switch mediastring {
				case "Kindle":
					if price > 0 {
						book.KindlePrice = float32(price)
					}

				case "Hardcover":
					book.HardPrice = float32(price)

				case "Paperback":
					book.PaperPrice = float32(price)

				}
			}
		}
	})
	return book, nil
}
