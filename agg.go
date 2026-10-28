package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/wNRG/blog_aggregator/internal/database"

)

func parsePublishedAt(value string) sql.NullTime {
	value = strings.TrimSpace(value)
	if value == "" {
		return sql.NullTime{}
	}

	layouts := []string{
		time.RFC1123Z,
		time.RFC1123,
		time.RFC822Z,
		time.RFC822,
		time.RFC3339,
		time.RFC3339Nano,
	}

	for _, layout := range layouts {
		publishedAt, err := time.Parse(layout, value)
		if err == nil {
			return sql.NullTime{
				Time: publishedAt, 
				Valid: true,
			}
		}
	}

	return sql.NullTime{}
}

func scrapeFeeds(s *state) error {
	ctx := context.Background()
	
	feed, err := s.db.GetNextFeedToFetch(ctx)
	if err != nil {
		return fmt.Errorf("get next feed to fetch: %w", err)
	}

	fmt.Printf("Fetching feed: %s\n", feed.Name)

	if err := s.db.MarkFeedFetched(ctx, feed.ID); err != nil {
		return fmt.Errorf("mark feed fetched: %w", err)
	}

	rssFeed, err := fetchFeed(ctx, feed.Url)
	if err != nil {
		return fmt.Errorf("fetch feed %q: %w", feed.Url, err)
	}

	for _, item := range rssFeed.Channel.Items {
		now := time.Now().UTC()

		err := s.db.CreatePost(ctx, database.CreatePostParams{
			ID: uuid.New(),
			CreatedAt: now,
			UpdatedAt: now,
			Title: item.Title,
			Url: item.Link,
			Description: item.Description,
			PublishedAt: parsePublishedAt(item.PubDate),
			FeedID: feed.ID,
		})

		if err != nil {
			fmt.Fprintf(os.Stderr, "save post %q: %v\n", item.Title, err)
		}
	}

	return nil
}
