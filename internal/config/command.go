package config

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"gator/internal/database"
	"gator/internal/rss"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

type State struct {
	Conf *Config
	Db   *database.Queries
}

type Command struct {
	Name      string
	Arguments []string
}

type Commands struct {
	Cmd map[string]func(s *State, c Command) error
}

func (c *Commands) Run(s *State, cmd Command) error {
	cmdFunc, ok := c.Cmd[cmd.Name]
	if !ok {
		return fmt.Errorf("unknown command %s", cmd.Name)
	}

	return cmdFunc(s, cmd)
}

func (c *Commands) Register(name string, f func(*State, Command) error) error {
	c.Cmd[name] = f
	return nil
}

func userExists(s *State, name string) bool {
	user, _ := s.Db.GetUserByName(context.Background(), name)
	return user.ID.String() != "00000000-0000-0000-0000-000000000000"
}

func HandlerLogin(s *State, c Command) error {
	if len(c.Arguments) != 1 {
		return fmt.Errorf("%s expects one argument: <username>", c.Name)
	}

	username := c.Arguments[0]
	if !userExists(s, username) {
		return fmt.Errorf("user '%s' does not exist", username)
	}

	err := s.Conf.SetUser(username)
	if err != nil {
		return fmt.Errorf("error logging in user: %w", err)
	}
	fmt.Printf("Logged in as %s\n", username)
	return nil
}

func HandlerRegister(s *State, c Command) error {
	if len(c.Arguments) == 0 {
		return fmt.Errorf("register command requires a name")
	}

	name := c.Arguments[0]
	if userExists(s, name) {
		return fmt.Errorf("user '%s' already exists", name)
	}

	id := uuid.New()
	createdAt := time.Now()
	updatedAt := time.Now()

	arg := database.CreateUserParams{
		ID:        id,
		Name:      name,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}

	newUser, err := s.Db.CreateUser(context.Background(), arg)
	if err != nil {
		return fmt.Errorf("failed to create user in database: %w", err)
	}

	if err := s.Conf.SetUser(name); err != nil {
		return err
	}

	fmt.Printf("successfully registered %s\n", name)
	fmt.Println(newUser)

	return nil
}

func HandlerReset(s *State, c Command) error {
	if len(c.Arguments) != 0 {
		return fmt.Errorf("reset does not take any arguments")
	}

	if err := s.Db.ClearUsersTable(context.Background()); err != nil {
		return fmt.Errorf("failed to clear users table: %w", err)
	}

	fmt.Println("successfully reset database")
	return nil
}

func HandlerUsers(s *State, c Command) error {
	if len(c.Arguments) != 0 {
		return fmt.Errorf("users does not take any arguments")
	}

	users, err := s.Db.GetAllUsers(context.Background())
	if err != nil {
		return fmt.Errorf("failed to get all users from db: %w", err)
	}

	for _, user := range users {
		if user == s.Conf.CurrentUserName {
			fmt.Printf("* %s (current)\n", user)
		} else {
			fmt.Printf("* %s\n", user)
		}
	}

	return nil
}

func HandlerAgg(s *State, c Command) error {
	if len(c.Arguments) != 1 {
		return fmt.Errorf("agg requires a refresh internal in the form of <1s> <1m> <1h>")
	}

	time_between_reqs, err := time.ParseDuration(c.Arguments[0])
	if err != nil {
		return fmt.Errorf("incorrect format: %w", err)
	}

	fmt.Printf("Collecting feeds every %v\n", time_between_reqs)

	ticker := time.NewTicker(time_between_reqs)
	for ; ; <-ticker.C {
		fmt.Println("Grabbing next feed...")
		scrapeFeeds(s)
	}
}

func HandlerAddFeed(s *State, c Command, user database.User) error {
	if len(c.Arguments) != 2 {
		return fmt.Errorf("addfeed expects <feed name> <feed URL>")
	}

	feedName := c.Arguments[0]
	feedURL := c.Arguments[1]
	createdAt := time.Now()
	updatedAt := time.Now()

	args := database.CreateFeedParams{
		ID:        uuid.New(),
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
		Name:      feedName,
		Url:       feedURL,
		UserID:    user.ID,
	}

	newFeed, err := s.Db.CreateFeed(context.Background(), args)
	if err != nil {
		return fmt.Errorf("failed to create new feed: %w", err)
	}

	fmt.Printf("successfully added %s to database\n", newFeed.Name)

	followingArgs := database.CreateFeedFollowParams{
		ID:        uuid.New(),
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
		UserID:    user.ID,
		FeedID:    newFeed.ID,
	}

	if _, err := s.Db.CreateFeedFollow(context.Background(), followingArgs); err != nil {
		return fmt.Errorf("failed to follow new feed: %w", err)
	}
	return nil
}

func HandlerFeeds(s *State, c Command) error {
	if len(c.Arguments) != 0 {
		return fmt.Errorf("feeds doesn't take any arguments")
	}

	allFeeds, err := s.Db.GetAllFeeds(context.Background())
	if err != nil {
		return fmt.Errorf("error getting all feeds from db: %w", err)
	}

	fmt.Println("Current feeds:")
	for i := range allFeeds {
		name := allFeeds[i].Name
		URL := allFeeds[i].Url

		user, err := s.Db.GetUserByID(context.Background(), allFeeds[i].UserID)
		if err != nil {
			fmt.Printf("failed to grab user info from db: %s", err)
			continue
		}

		fmt.Printf("Name: %s || URL: %s || Added By: %s\n", name, URL, user.Name)
	}
	return nil
}

func HandlerFollow(s *State, c Command, user database.User) error {
	if len(c.Arguments) != 1 {
		return fmt.Errorf("follow command requires <url> argument")
	}

	url := c.Arguments[0]
	feedID, err := s.Db.GetFeedIDByURL(context.Background(), url)
	if err != nil {
		return fmt.Errorf("failed to get feed name by url: %w", err)
	}

	arg := database.CreateFeedFollowParams{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		UserID:    user.ID,
		FeedID:    feedID,
	}

	followedFeed, err := s.Db.CreateFeedFollow(context.Background(), arg)
	if err != nil {
		return fmt.Errorf("failed to follow feed: %w", err)
	}

	fmt.Printf("%s has followed %s", followedFeed.UserName, followedFeed.FeedName)
	return nil
}

func HandlerFollowing(s *State, c Command, user database.User) error {
	if len(c.Arguments) != 0 {
		return fmt.Errorf("following command doesn't accept arguments")
	}

	followedFeeds, err := s.Db.GetFeedFollowsForUser(context.Background(), user.Name)
	if err != nil {
		return fmt.Errorf("failed to get user from db: %w", err)
	}

	if len(followedFeeds) == 0 {
		fmt.Println("You aren't following any feeds")
		return nil
	}

	fmt.Println("You are following:")
	for i := range followedFeeds {
		fmt.Printf("* %s\n", followedFeeds[i].Feed)
	}

	return nil
}

func MiddlewareLoggedIn(handler func(s *State, cmd Command, user database.User) error) func(*State, Command) error {
	return func(s *State, cmd Command) error {
		user, err := s.Db.GetUserByName(context.Background(), s.Conf.CurrentUserName)
		if err != nil {
			return fmt.Errorf("failed to query user: %w", err)
		}
		return handler(s, cmd, user)
	}
}

func HandlerUnfollow(s *State, c Command, user database.User) error {
	if len(c.Arguments) != 1 {
		return fmt.Errorf("unfollow only accepts one argument: <url>")
	}

	ctx := context.Background()

	feedURL := c.Arguments[0]
	feedID, err := s.Db.GetFeedIDByURL(ctx, feedURL)
	if err != nil {
		return fmt.Errorf("failed to query url: %w", err)
	}

	arg := database.DeleteFeedFollowParams{
		UserID: user.ID,
		FeedID: feedID,
	}

	if err := s.Db.DeleteFeedFollow(ctx, arg); err != nil {
		return fmt.Errorf("failed to unfollow: %w", err)
	}

	return nil
}

func HandlerBrowse(s *State, c Command, user database.User) error {
	var limit int32
	if len(c.Arguments) > 1 {
		return fmt.Errorf("browse takes an optional number of posts to view")
	}

	if len(c.Arguments) == 1 {
		parsedInt, err := strconv.Atoi(c.Arguments[0])
		if err != nil {
			return fmt.Errorf("enter an int only as the argument")
		}
		if parsedInt <= 0 {
			return fmt.Errorf("enter a positive number only")
		}
		limit = int32(parsedInt)
	} else {
		limit = 2
	}

	arg := database.GetPostsForUserParams{
		UserID: user.ID,
		Limit:  limit,
	}
	posts, err := s.Db.GetPostsForUser(context.Background(), arg)
	if err != nil {
		return fmt.Errorf("failed to get posts: %w", err)
	}

	for i := range posts {
		title := posts[i].Title
		description := posts[i].Description

		fmt.Printf("Post %d\nTitle: %s\nDescription: %v\n\n", i, title, description)
	}
	return nil
}

func scrapeFeeds(s *State) {
	ctx := context.Background()
	nextFeed, err := s.Db.GetNextFeedToFetch(ctx)
	if err != nil {
		fmt.Printf("failed to get next feed from db: %s\n", err)
		return
	}

	now := time.Now()
	arg := database.MarkFeedFetchedParams{UpdatedAt: now, ID: nextFeed.ID}
	if err := s.Db.MarkFeedFetched(ctx, arg); err != nil {
		fmt.Printf("failed to mark feed as fetched in db: %s\n", err)
		return
	}

	feed, err := rss.FetchFeed(ctx, nextFeed.Url)
	if err != nil {
		fmt.Printf("failed to fetch feed from url: %s\n", err)
		return
	}

	numPosts := 0
	rss.CleanHTML(feed)
	for i := range feed.Channel.Item {
		postErr := createPost(s, feed.Channel.Item[i], nextFeed.ID)
		if postErr != nil {
			if isDuplicatePostURL(postErr) {
				continue
			}

			fmt.Printf("failed to create post: %v\n", postErr)
			continue
		}
		numPosts++
	}

	if numPosts == 0 {
		fmt.Println("checked feed, no new posts to add")
	} else {
		fmt.Printf("successfully added %d posts\n", numPosts)
	}
}

func createPost(s *State, post rss.RSSItem, feedID uuid.UUID) error {
	pubTime, err := http.ParseTime(post.PubDate)
	if err != nil {
		pubTime = time.Time{}
	}

	description := sql.NullString{
		String: post.Description,
		Valid:  post.Description != "",
	}

	postArgs := database.CreatePostParams{
		ID:          uuid.New(),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Title:       post.Title,
		Url:         post.Link,
		Description: description,
		PublishedAt: sql.NullTime{
			Time:  pubTime,
			Valid: !pubTime.IsZero(),
		},
		FeedID: feedID,
	}

	if err := s.Db.CreatePost(context.Background(), postArgs); err != nil {
		return err
	}

	return nil
}

func isDuplicatePostURL(err error) bool {
	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		return pqErr.Code == "23505" && pqErr.Constraint == "posts_url_key"
	}
	return false
}
