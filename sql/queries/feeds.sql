-- name: CreateFeed :one
INSERT INTO feeds(id, created_at, updated_at, name, url, user_id)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: GetFeeds :many
SELECT
	feeds.name AS feed_name,
	feeds.url AS feed_url,
	users.name AS user_name
FROM feeds
JOIN users ON feeds.user_id = users.id
ORDER BY feeds.name;

-- name: GetFeedByURL :one
SELECT * FROM feeds
WHERE url = $1;

-- name: CreateFeedFollow :one
WITH inserted AS (
	INSERT INTO feed_follows (id, created_at, updated_at, user_id, feed_id)
	VALUES ($1, $2, $3, $4, $5)
	RETURNING *
)
SELECT
	inserted.*,
	users.name AS user_name,
	feeds.name AS feed_name
FROM inserted
JOIN users ON inserted.user_id = users.id
JOIN feeds ON inserted.feed_id = feeds.id;

-- name: GetFeedFollowsForUser :many
SELECT
	feed_follows.*,
	users.name AS user_name,
	feeds.name AS feed_name
FROM feed_follows
JOIN users ON feed_follows.user_id = users.id
JOIN feeds ON feed_follows.feed_id = feeds.id
WHERE feed_follows.user_id = $1
ORDER BY feeds.name;

-- name: DeleteFeedFollow :exec
DELETE FROM feed_follows
WHERE user_id = $1 AND feed_id = $2;

-- name: MarkFeedFetched :exec
UPDATE feeds
SET last_fetched_at = NOW(),
	updated_at = NOW()
WHERE id = $1;

-- name: GetNextFeedToFetch :one
SELECT * FROM feeds
ORDER BY last_fetched_at ASC NULLS FIRST
LIMIT 1;

