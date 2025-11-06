package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/joho/godotenv"
	"github.com/vartanbeno/go-reddit/v2/reddit"
)

const idFile = "last_seen_posts.json"

// loadIDs reads the saved post IDs from our JSON file.
func loadIDs(filename string) map[string]string {
	// This map will hold subreddit -> latest_post_ID
	ids := make(map[string]string)

	data, err := os.ReadFile(filename)
	if err != nil {
		// If the file doesn't exist, that's fine. It's the first run.
		if !os.IsNotExist(err) {
			log.Printf("WARN: Failed to read %s: %v", filename, err)
		}
		return ids
	}

	// Unmarshal the JSON data into our map
	if err := json.Unmarshal(data, &ids); err != nil {
		log.Printf("WARN: Failed to parse %s: %v. Starting fresh.", filename, err)
		// Return an empty map if JSON is corrupt
		return make(map[string]string)
	}

	log.Println("Successfully loaded last seen post IDs.")
	return ids
}

// saveIDs writes the new latest post IDs to our JSON file.
func saveIDs(filename string, ids map[string]string) {
	// Marshal the map into a pretty-printed JSON
	data, err := json.MarshalIndent(ids, "", "  ")
	if err != nil {
		log.Printf("ERROR: Failed to marshal IDs for saving: %v", err)
		return
	}

	// Write the data to the file
	if err := os.WriteFile(filename, data, 0644); err != nil {
		log.Printf("ERROR: Failed to write IDs to %s: %v", filename, err)
	}
}

func getReddit() {
	fmt.Println("Getting reddit")

	err := godotenv.Load()
	if err != nil {
		fmt.Println("Error loading the env file")
	}

	client_id := os.Getenv("Client_id")
	secret := os.Getenv("Client_secret")
	user := os.Getenv("User_name")
	password := os.Getenv("Password")

	targetSubreddits := []string{
		"IndiaInvestments",
		"IndianStockMarket",
		"IndianStreetBets",
		"stocks",
		"business",
		"news",
	}

	// --- 1. Load the last seen post IDs ---
	lastSeenIDs := loadIDs(idFile)

	// This map will be safely populated by our goroutines
	newLatestIDs := make(map[string]string)
	var idMutex sync.Mutex // A mutex to protect the map

	// --- 4. Authenticate with Reddit ---
	credentials := reddit.Credentials{
		ID:       client_id,
		Secret:   secret,
		Username: user,
		Password: password,
	}
	client, err := reddit.NewClient(credentials)
	if err != nil {
		log.Fatalf("Failed to create Reddit client: %v", err)
	}

	// --- 5. Set up Concurrency ---
	var wg sync.WaitGroup
	resultsChan := make(chan *reddit.Post)
	ctx := context.Background()

	log.Println("Starting concurrent monitoring of new posts...")

	// --- 6. Launch Goroutines (One per subreddit) ---
	for _, sub := range targetSubreddits {
		wg.Add(1)
		// Get the last seen ID for this *specific* subreddit
		lastID := lastSeenIDs[sub]

		// Launch a worker with the lastID and the map to update
		go monitorSubreddit(ctx, client, sub, &wg, resultsChan, lastID, &newLatestIDs, &idMutex)
	}

	// Start a goroutine to close the channel once all workers are done
	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	// --- 7. Collect & Deduplicate Results ---
	finalPosts := make(map[string]*reddit.Post)

	for post := range resultsChan {
		if _, exists := finalPosts[post.ID]; !exists {
			finalPosts[post.ID] = post
		}
	}

	// --- NEW: Save the new latest IDs ---
	// This happens *after* the channel is closed and all goroutines are done.
	log.Println("Scan processing complete. Saving new latest IDs...")
	saveIDs(idFile, newLatestIDs)

	// --- 8. Print Final Results ---
	log.Println("Displaying matched posts:")
	if len(finalPosts) == 0 {
		fmt.Println("No matching posts found in 'new' feeds.")
	}

	for _, post := range finalPosts {
		// ... (rest of your printing logic is unchanged) ...
		matches := companyRegex.FindStringSubmatch(post.Title)
		companyMatched := "N/A"
		if len(matches) > 1 {
			companyMatched = matches[1] // Get the first capture group
		}

		postType := "Discussion"
		sourceURL := ""
		if !post.IsSelfPost { // 'IsSelf' = false means it's a link post
			postType = "Link"
			sourceURL = fmt.Sprintf(" (Source: %s)", post.URL)
		}

		fmt.Printf("\n[Company: %s] (Found in r/%s)\n", companyMatched, post.SubredditName)
		fmt.Printf("  [%s] %s%s\n", postType, post.Title, sourceURL)
		fmt.Printf("  Reddit URL: https://www.reddit.com%s\n", post.Permalink)

		if post.IsSelfPost && post.Body != "" {
			fmt.Printf("  Body: %s\n", post.Body)
		}
	}

	fmt.Println("\n\nFinished Reddit")
}

/**
 * monitorSubreddit is our "worker" function.
 * It fetches the latest posts from a single subreddit and applies
 * the regex filter.
 */
func monitorSubreddit(ctx context.Context, client *reddit.Client, subredditName string, wg *sync.WaitGroup,
	ch chan<- *reddit.Post, lastSeenID string,
	newLatestIDs *map[string]string, idMutex *sync.Mutex) {
	// Ensure WaitGroup is marked as 'done' when this function exits
	defer wg.Done()

	// Set options to get the 50 newest posts
	listingOptions := &reddit.ListOptions{
		Limit: 50,
		// REMOVE THIS LINE: Before: lastSeenID,
	}

	posts, _, err := client.Subreddit.NewPosts(ctx, subredditName, listingOptions)
	if err != nil {
		log.Printf("ERROR fetching new posts from r/%s: %v", subredditName, err)
		// If we error, lock and save the *old* ID so we don't lose our place
		idMutex.Lock()
		if lastSeenID != "" {
			(*newLatestIDs)[subredditName] = lastSeenID
		}
		idMutex.Unlock()
		return
	}

	// --- Update the map with the latest ID for this subreddit ---
	idMutex.Lock() // Lock the mutex before writing to the shared map
	if len(posts) > 0 {
		// If we got new posts, store the ID of the newest one (at index 0)
		(*newLatestIDs)[subredditName] = posts[0].ID
	} else if lastSeenID != "" {
		// No new posts were found, so we keep the same (old) ID
		(*newLatestIDs)[subredditName] = lastSeenID
	}
	idMutex.Unlock() // Unlock the mutex

	// --- This is where your logic is applied ---
	for _, post := range posts {
		// --- THIS IS THE FIX ---
		// If we've reached the post that was newest on the last run,
		// stop processing. We have seen all posts from this point on.
		if lastSeenID != "" && post.ID == lastSeenID {
			break
		}
		// --- END OF FIX ---

		// Apply your regex function to the post title
		if contains(post.Title) {
			// If it matches, send the post back to the main channel
			ch <- post
		}
	}
}
