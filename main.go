package main

import (
	"database/sql"
	"gator/internal/config"
	"gator/internal/database"
	"log"
	"os"

	_ "github.com/lib/pq"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("not enough arguments provided")
	}

	conf, err := config.ReadConfig()
	if err != nil {
		log.Fatal("error reading config: %w", err)
	}

	state := config.State{Conf: &conf}
	db, err := sql.Open("postgres", conf.DbURL)
	if err != nil {
		log.Fatal("error connecting to database: %w", err)
	}

	dbQueries := database.New(db)
	state.Db = dbQueries

	allCommands := config.Commands{
		Cmd: make(map[string]func(*config.State, config.Command) error),
	}
	allCommands.Register("login", config.HandlerLogin)
	allCommands.Register("register", config.HandlerRegister)
	allCommands.Register("reset", config.HandlerReset)
	allCommands.Register("users", config.HandlerUsers)
	allCommands.Register("agg", config.HandlerAgg)
	allCommands.Register("addfeed", config.HandlerAddFeed)
	allCommands.Register("feeds", config.HandlerFeeds)

	commandName := os.Args[1]
	args := os.Args[2:]

	command := config.Command{Name: commandName, Arguments: args}
	if err := allCommands.Run(&state, command); err != nil {
		log.Fatal(err)
	}
}
