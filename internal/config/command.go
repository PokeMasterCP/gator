package config

import (
	"context"
	"fmt"
	"gator/internal/database"
	"gator/internal/rss"
	"time"

	"github.com/google/uuid"
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
	feed, err := rss.FetchFeed(context.Background(), "https://www.wagslane.dev/index.xml")
	if err != nil {
		return err
	}

	rss.CleanHTML(feed)
	fmt.Println(*feed)
	return nil
}

func HandlerAddFeed(s *State, c Command) error {
	if len(c.Arguments) != 2 {
		return fmt.Errorf("addfeed expects <feed name> <feed URL>")
	}

	currentUser, err := s.Db.GetUserByName(context.Background(), s.Conf.CurrentUserName)
	if err != nil {
		return fmt.Errorf("failed to grab %s from database: %w", s.Conf.CurrentUserName, err)
	}

	feedName := c.Arguments[0]
	feedURL := c.Arguments[1]
	args := database.CreateFeedParams{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Name:      feedName,
		Url:       feedURL,
		UserID:    currentUser.ID,
	}

	newFeed, err := s.Db.CreateFeed(context.Background(), args)
	if err != nil {
		return fmt.Errorf("failed to create new feed: %w", err)
	}

	fmt.Printf("successfully added %s to database\n", newFeed.Name)

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
			fmt.Printf("failed to grab user %s from db: %w")
			continue
		}

		fmt.Printf("Name: %s || URL: %s || Added By: %s\n", name, URL, user.Name)
	}
	return nil
}
