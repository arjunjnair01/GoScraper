package main

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/gocolly/colly/v2"
)

// storing diff possible names of the top 10 nifty 50 companies as a reg exp
var companyRegex = regexp.MustCompile(
	fmt.Sprintf(`(?i)\b(%s)\b`, strings.Join([]string{
		"Reliance Industries", "Reliance", "RIL", "Tata Consultancy Services",
		"TCS", "Tata Consultancy", "HDFC Bank", "HDFC", "ICICI Bank", "ICICI",
		"Bharti Airtel", "Airtel", "State Bank of India", "SBI", "Infosys",
		"Life Insurance Corporation", "LIC", "Hindustan Unilever", "HUL", "ITC",
	}, "|")),
)

// function to match the input title of news to the selected companies using regex
func contains(title string) bool {
	return companyRegex.MatchString(title)
}

func main() {

	c := colly.NewCollector(
		colly.AllowedDomains(
			"www.thehindubusinessline.com",
			"www.cnbctv18.com",
			"www.investing.com",
			"www.moneycontrol.com",
			"economictimes.indiatimes.com",
		),
		colly.MaxDepth(2),
	)
	c.UserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/104.0.0.0 Safari/537.36"
	c.OnError(func(e *colly.Response, err error) {
		fmt.Printf("Error")
	})
	c.OnRequest(func(e *colly.Request) {
		fmt.Printf("\n\nVisiting url: %s \n", e.URL.String())
	})

	c.OnXML("//item", func(e *colly.XMLElement) {
		title := e.ChildText("//title")
		if contains(title) {
			fmt.Printf("\n\n")
			fmt.Println(title)
			desc := e.ChildText("description")
			fmt.Printf("Desc: %s\n", desc)
			time := e.ChildText("pubDate")
			fmt.Printf("Time : %s\n", time)
		}
	})

	urls := []string{
		"https://www.investing.com/rss/news_356.rss",
		"https://economictimes.indiatimes.com/markets/rssfeeds/1977021501.cms",
		"https://www.cnbctv18.com/commonfeeds/v1/cne/rss/india.xml",
		"https://www.cnbctv18.com/commonfeeds/v1/cne/rss/economy.xml",
		"https://www.cnbctv18.com/commonfeeds/v1/cne/rss/market.xml",
		"https://www.cnbctv18.com/commonfeeds/v1/cne/rss/business.xml",
		"https://www.thehindubusinessline.com/markets/feeder/default.rss",
		"https://economictimes.indiatimes.com/markets/stocks/rssfeeds/2146842.cms",
	}

	for _, url := range urls {
		c.Visit(url)
	}
	c.Wait()
	fmt.Println("Finish")
}
