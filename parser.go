package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
	readability "github.com/go-shiori/go-readability"
)

type Article struct {
	URL       string    `json:"url"`
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	ScrapedAt time.Time `json:"scraped_at"`
}

func ParseAndSave(url string) error {
	article, err := readability.FromURL(url, 30*time.Second)
	if err != nil {
		return err
	}

	var articles []Article
	data, _ := os.ReadFile("articles.json")
	json.Unmarshal(data, &articles)

	articles = append(articles, Article{
		URL:       url,
		Title:     article.Title,
		Content:   article.TextContent,
		ScrapedAt: time.Now(),
	})

	data, _ = json.MarshalIndent(articles, "", "  ")
	os.WriteFile("articles.json", data, 0644)
	fmt.Printf("✓ Saved article: %s (Total: %d)\n", article.Title, len(articles))
	return nil
}