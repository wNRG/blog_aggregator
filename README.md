# Gator

Gator is a Go command-line RSS feed aggregator backed by PostgreSQL. It can manage users and feed subscriptions, continuously fetch RSS feeds, save posts, and display posts from feeds a user follows.

## Current progress

Implemented or in progress:

- User registration, login, and listing
- Adding feeds and listing available feeds
- Following, unfollowing, and listing followed feeds
- Database migrations for feeds, feed-fetch tracking, and posts
- Continuous feed scraping with a configurable request interval
- Saving posts and skipping duplicate post URLs
- Browsing posts from feeds followed by the current user

This project is still under development. The RSS date parser supports several common date formats; feeds using other formats may be saved without a publication date.

*Still using `go run .` for development.*

## Requirements

You’ll need Go and PostgreSQL installed. Goose is needed to apply the database migrations.

## Install

Install the project with Go:

```sh
go install github.com/wNRG/blog\_aggregator@latest
