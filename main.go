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
	amazonItemLookup("B07QDNZ1V2")
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

var amazonRequestChannel chan string
var amazonBookResultChannel chan amazonBook

var rxPrice = regexp.MustCompile(`\$(\d+(\.\d+)?)`)
var rxRating = regexp.MustCompile(`([\d.]+) `)

func amazonItemLookup(ASIN string) (*amazonBook, error) {

	book := &amazonBook{}
	var err error

	c := colly.NewCollector(
		colly.Async(true),
	)

	c.Limit(&colly.LimitRule{
		RandomDelay: 2 * time.Second,
		Parallelism: 4,
	})

	extensions.RandomUserAgent(c)

	c.OnHTML("#dp-container", func(e *colly.HTMLElement) {
		book.Title = e.ChildText("#title .a-size-extra-large")
		book.Author = e.ChildText(".contributorNameID")
		book.ASIN = e.ChildAttr(".contributorNameID", "data-asin")
		book.CoverImageURL = e.ChildAttr(".frontImage", "src")

		ratingmatch := rxRating.FindStringSubmatch(e.ChildText(".a-icon .a-icon-alt"))
		if len(ratingmatch) > 1 {
			rating, err := strconv.ParseFloat(ratingmatch[1], 32)
			if err == nil {
				book.Rating = float32(rating)
			}
		}

		kindlematch := rxPrice.FindStringSubmatch(e.ChildText(".kindle-price .a-color-price"))
		if len(kindlematch) > 1 {
			kindleprice, err := strconv.ParseFloat(kindlematch[1], 32)
			if err == nil {
				book.KindlePrice = float32(kindleprice)
			}
		}

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
	})

	// Set error handler
	c.OnError(func(r *colly.Response, e error) {
		err = e
		fmt.Println("Request URL:", r.Request.URL, "failed with response:", r, "\nError:", err)
	})

	c.Visit("https://www.amazon.com/dp/" + ASIN)

	c.Wait()

	return book, err
}
