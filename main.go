package main

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gocolly/colly"

	"github.com/gocolly/colly/extensions"
)

func main() {
	amazonItemLookup("B00TU3BTUI")
}

type amazonBook struct {
	ASIN          string
	Title         string
	Author        string
	CoverImageURL string
	KindlePrice   float32
	HardPrice     float32
	PaperPrice    float32
}

var amazonRequestChannel chan string
var amazonBookResultChannel chan amazonBook

var rxPrice = regexp.MustCompile(`\$(\d+(\.\d+)?)`)

func amazonItemLookup(ASIN string) (*amazonBook, error) {

	book := &amazonBook{}

	c := colly.NewCollector(
		colly.Async(true),
	)

	c.Limit(&colly.LimitRule{
		RandomDelay: 2 * time.Second,
		Parallelism: 4,
	})

	extensions.RandomUserAgent(c)

	c.OnHTML("#dp-container", func(e *colly.HTMLElement) {
		book.Title = e.ChildText("#title span")
		book.Author = e.ChildText(".contributorNameID")
		book.CoverImageURL = e.ChildAttr(".frontImage", "src")

		e.ForEach(".swatchElement", func(_ int, x *colly.HTMLElement) {
			//type of media (kindle, hardcover, paperback, audiobook, etc)
			mediastring := x.ChildText(".a-button-text span")
			mediastring = mediastring[0:strings.Index(mediastring, "\n")]

			//read the price
			pricestring := x.ChildText(".a-color-price")
			if len(pricestring) == 0 {
				pricestring = x.ChildText(".a-color-secondary")
			}
			pricematch := rxPrice.FindStringSubmatch(pricestring)

			if len(pricematch) > 1 {
				price, err := strconv.ParseFloat(pricematch[1], 32)

				if err == nil {
					switch mediastring {
					case "Kindle":
						book.KindlePrice = float32(price)

					case "Hardcover":
						book.HardPrice = float32(price)

					case "Paperback":
						book.PaperPrice = float32(price)

					}
				}
			}
		})
	})

	// Set error handler
	c.OnError(func(r *colly.Response, err error) {
		fmt.Println("Request URL:", r.Request.URL, "failed with response:", r, "\nError:", err)
	})

	c.Visit("https://www.amazon.com/dp/" + ASIN)

	c.Wait()

	return book, nil
}
