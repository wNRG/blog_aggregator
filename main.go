package main

import ( 
	"context"
	"database/sql"
	"fmt"
	"os"
	"time"
	"strconv"
	
	"github.com/google/uuid"
	"github.com/wNRG/blog_aggregator/internal/database"
	"github.com/wNRG/blog_aggregator/internal/config"
	_ "github.com/lib/pq"
)

type state struct {
	db *database.Queries
	cfg *config.Config
}

type command struct {
	name string
	args []string
}

type commands struct {
	handlers map[string] func(*state, command) error
}

func (c *commands) run(s *state, cmd command) error {
	handler, ok := c.handlers[cmd.name]
	if !ok {
		return fmt.Errorf("Unknown command: %s", cmd.name)
	}

	return handler(s, cmd)
}

func (c *commands) register(name string, f func(*state, command) error) {
	c.handlers[name] = f
}

func handlerLogin(s *state, cmd command) error {
	if len(cmd.args) == 0 {
		return fmt.Errorf("ERROR: missing argument username")
	}

	if len(cmd.args) != 1 {
		return fmt.Errorf("ERROR: expects one username")
	}

	_, err := s.db.GetUser(context.Background(), cmd.args[0])
	if err != nil {
		return fmt.Errorf("user %s does not exist", cmd.args[0])
	}

	if err := s.cfg.SetUser(cmd.args[0]); err != nil {
		return err
	}

	fmt.Printf("User set to %s\n", cmd.args[0])
	return nil
}

func handlerRegister(s* state, cmd command) error {
	if len(cmd.args) == 0 {
		return fmt.Errorf("missing argument username")
	}

	if len(cmd.args) != 1 {
		return fmt.Errorf("expects one username")
	}

	now := time.Now()

	user, err := s.db.CreateUser(context.Background(), database.CreateUserParams{
		ID: uuid.New(),
		CreatedAt: now,
		UpdatedAt: now,
		Name: cmd.args[0],
	})
	if err != nil {
		return fmt.Errorf("could not create user: %w", err)
	}

	if err := s.cfg.SetUser(cmd.args[0]); err != nil {
		return err
	}

	fmt.Printf("User %s created successfully!\n", user.Name)
	fmt.Printf("%+v\n", user)

	return nil
}

func handlerReset(s *state, cmd command) error {
	err := s.db.DeleteAllUsers(context.Background())
	if err != nil {
		return fmt.Errorf("failed to reset database: %w", err)	
	}

	fmt.Println("Database has been reset successfully")
	return nil
}

func handlerUsers(s *state, cmd command) error {
	users, err := s.db.GetUsers(context.Background())
	if err != nil {
		return err
	}

	for _, user := range users {
		if user.Name == s.cfg.CurrentUserName {
			fmt.Printf("* %s (current)\n", user.Name)
		} else {
			fmt.Printf("* %s\n", user.Name)
		}
	}

	return nil
}

func handlerAgg(s *state, cmd command) error {
	if len(cmd.args) != 1 {
		return fmt.Errorf("usage: agg <time_between_reqs>")
	}

	timeBetweenRequests, err := time.ParseDuration(cmd.args[0])
	if err != nil {
		return fmt.Errorf("invalid time_between_reqs %q: %w",cmd.args[0], err)
	}

	if timeBetweenRequests <= 0 {
		return fmt.Errorf("time_between_reqs must be greater than zero")
	}

	fmt.Printf("Collecting feeds every %s\n", timeBetweenRequests)

	ticker := time.NewTicker(timeBetweenRequests)
	defer ticker.Stop()

	for {
		if err := scrapeFeeds(s); err != nil {
			fmt.Fprintf(os.Stderr, "scrape feeds: %v\n", err)
		}
		<-ticker.C
	}


}


func handlerAddFeed(s *state, cmd command, user database.User) error {
	if len(cmd.args) != 2 {
		return fmt.Errorf("usage: addfeed <name> <url>")
	} 

	ctx := context.Background()
	now := time.Now().UTC()

	feed, err := s.db.CreateFeed(ctx, database.CreateFeedParams{
		ID: uuid.New(),
		CreatedAt: now,
		UpdatedAt: now,
		Name: cmd.args[0],
		Url: cmd.args[1],
		UserID: user.ID,
	})
	if err != nil {
		return fmt.Errorf("create feed: %w", err)
	}

	follow, err := s.db.CreateFeedFollow(ctx, database.CreateFeedFollowParams{
		ID: uuid.New(),
		CreatedAt: now,
		UpdatedAt: now,
		UserID: user.ID,
		FeedID: feed.ID,
	})
	if err != nil {
		return fmt.Errorf("follow newly added feed: %w", err)
	}

	fmt.Printf("Feed created: %+v\n", feed)
	fmt.Printf("%s is now following %s\n", follow.UserName, follow.FeedName)
	return nil
}

func handlerFeeds(s *state, cmd command) error {
	if len(cmd.args) != 0 {
		return fmt.Errorf("usage: feeds")
	}

	feeds, err := s.db.GetFeeds(context.Background())
	if err != nil {
		return err
	}

	for _, feed := range feeds {
		fmt.Printf("* %s\n URL: %s\n Added by: %s\n",
				feed.FeedName,
				feed.FeedUrl,
				feed.UserName,
			)
	}

	return nil
}

func handlerFollow(s *state, cmd command, user database.User) error {
	if len(cmd.args) != 1 {
		return fmt.Errorf ("usage: follow <url>")
	}

	ctx := context.Background()
	feed, err := s.db.GetFeedByURL(ctx, cmd.args[0])

	if err != nil {
		return fmt.Errorf("find feed by URL: %w",err)
	}

	now := time.Now().UTC()
	follow, err := s.db.CreateFeedFollow(ctx, database.CreateFeedFollowParams{
		ID: uuid.New(),
		CreatedAt: now,
		UpdatedAt: now,
		UserID: user.ID,
		FeedID: feed.ID,
	})

	if err != nil {
		return fmt.Errorf("create feed follow: %w", err)
	}

	fmt.Printf("%s is now following %s\n", follow.UserName, follow.FeedName)
	return nil
}

func handlerFollowing(s *state, cmd command, user database.User) error {
	if len(cmd.args) != 0 {
		return fmt.Errorf ("usage: following")
	}

	ctx := context.Background()
	follows, err := s.db.GetFeedFollowsForUser(ctx, user.ID)

	if err != nil {
		return fmt.Errorf("get followed feeds: %w", err)
	}

	for _, follow := range follows {
		fmt.Println(follow.FeedName)
	}

	return nil
}

func handlerBrowse(s *state, cmd command, user database.User) error {
	if len(cmd.args) > 1 {
		return fmt.Errorf("usage: browse [limit]")
	}

	limit := int32(2)
	if len(cmd.args) == 1 {
		n, err := strconv.Atoi(cmd.args[0])
		if err != nil || n < 1 {
			return fmt.Errorf("limit must be a positive integer")
		}
		limit = int32(n)
	}

	posts, err := s.db.GetPostsForUser(context.Background(), database.GetPostsForUserParams{
		UserID: user.ID,
		Limit: limit,
	})
	if err != nil {
		return fmt.Errorf("get posts: %w", err)
	}

	for _, post := range posts {
		fmt.Printf("Title: %s\nURL: %s\nPublished: %s\nDescription: %s\n\n",
			post.Title,
			post.Url,
			post.PublishedAt.Time,
			post.Description,
		)
	}

	return nil
}

func handlerUnfollow(s *state, cmd command, user database.User) error {
	if len(cmd.args) != 1 {
		return fmt.Errorf("usage: unfollow <url>")
	}

	ctx := context.Background()
	
	feed, err := s.db.GetFeedByURL(ctx, cmd.args[0])
	if err != nil {
		return fmt.Errorf("find feed by URL: %w", err)
	}

	err = s.db.DeleteFeedFollow(ctx, database.DeleteFeedFollowParams{
		UserID: user.ID,
		FeedID: feed.ID,
	})
	if err != nil {
		return fmt.Errorf("unfollow feed: %w", err)
	}

	fmt.Printf("%s unfollowed %s\n", user.Name, feed.Name)
	return nil
}
func main() {

	args := os.Args
	if len(args) < 2 {
		fmt.Println("not enough arguments")
		os.Exit(1)
	}

	cfg, err := config.Read()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	db, err := sql.Open("postgres", cfg.DBURL)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	defer db.Close()

	dbQueries := database.New(db)

	s:= &state {
		db: dbQueries,
		cfg: &cfg,
	}

	cmds := &commands{
		handlers: make(map[string]func(*state, command) error),
	}

	cmds.register("login", handlerLogin)
	cmds.register("register", handlerRegister)
	cmds.register("reset", handlerReset)
	cmds.register("users", handlerUsers)
	cmds.register("agg", handlerAgg)
	cmds.register("addfeed", middlewareLoggedIn(handlerAddFeed))
	cmds.register("feeds", handlerFeeds)
	cmds.register("follow", middlewareLoggedIn(handlerFollow))
	cmds.register("following", middlewareLoggedIn(handlerFollowing))
	cmds.register("unfollow", middlewareLoggedIn(handlerUnfollow))
	cmds.register("browse", middlewareLoggedIn(handlerBrowse))

	cmd := command{
		name: args[1],
		args: args[2:],
	}

	if err := cmds.run(s, cmd); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
