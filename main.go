package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/gocolly/colly/v2"
)

type ScanTime struct {
	Time string `json:"time"`
}

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

// func to convert the time to RFC1123Z
func convertToRFC1123Z(timeStr string) (time.Time, error) {
	formats := []string{
		time.RFC1123,     // "Mon, 02 Jan 2006 15:04:05 MST"
		time.RFC1123Z,    // "Mon, 02 Jan 2006 15:04:05 -0700"
		time.RFC3339,     // "2006-01-02T15:04:05Z07:00"
		time.RFC822,      // "02-Jan-06 15:04 MST"
		time.RFC822Z,     // "02-Jan-06 15:04 -0700"
		time.RFC3339Nano, // "2006-01-02T15:04:05.999999999Z07:00"
		time.ANSIC,       // "Mon Jan _2 15:04:05 2006"
		"Mon, 2 Jan 2006 15:04:05 -0700",
		"2006-01-02 15:04:05",
	}

	for _, format := range formats {
		x, err := time.Parse(format, timeStr)
		if err == nil {
			return x, nil
		}
	}
	return time.Time{}, errors.New("error parsing")
}

func main() {

	//reading the last scan time from file
	file, err := os.Open("lastScan.json")
	if err != nil {
		fmt.Println("Error opening file")
		return
	}

	byteData, _ := io.ReadAll(file)
	var prev ScanTime
	json.Unmarshal(byteData, &prev)
	fmt.Printf("PREV TIME: %s\n\n", prev.Time)

	//creating collector
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

	prevTime, err := time.Parse(time.RFC1123Z, prev.Time)
	if err != nil {
		fmt.Printf("Error creating prevTime")
	}

	c.OnXML("//item", func(e *colly.XMLElement) {
		title := e.ChildText("//title")
		sctime := e.ChildText("pubDate")
		currTime, err := convertToRFC1123Z((sctime))
		if err != nil {
			fmt.Printf("Error creating currTime")
		}

		if currTime.After(prevTime) {
			if contains(title) {
				fmt.Printf("\n\n")
				fmt.Println(title)
				desc := e.ChildText("description")
				fmt.Printf("Desc: %s\n", desc)
				fmt.Printf("Time : %s\n", sctime)
				//calling the parse and save function
				link := e.ChildText("link")
				if link != "" {
					fmt.Printf("Processing URL: %s\n", link)
					err := ParseAndSave(link)
					if err != nil {
						fmt.Printf("Error parsing article: %v\n", err)
					}
				}
			}
		}

	})

	c.OnScraped(func(r *colly.Response) {
		scantime := &ScanTime{
			Time: time.Now().Format(time.RFC1123Z),
		}
		data, _ := json.MarshalIndent(scantime, "", "")
		err := os.WriteFile("lastScan.json", data, 0644)
		if err != nil {
			fmt.Println("Error creating a file")
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
