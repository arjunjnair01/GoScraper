package main

import (
	"fmt"
	"regexp"
	"strings"
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
	getNews()
	getReddit()
}
