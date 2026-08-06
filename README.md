# Gator

A command-line RSS feed aggregator written in Go. Register a user, follow feeds,
and run a long-lived scraper that pulls new posts on an interval and stores them
in PostgreSQL. Then browse what came in from the terminal.

Built as part of the [boot.dev](https://boot.dev) course on SQL and Go.

## Requirements

- Go 1.26+
- PostgreSQL
- [goose](https://github.com/pressly/goose) to run the migrations
- [sqlc](https://sqlc.dev) only if you change anything in `sql/queries`

## Setup

1. **Create the database.**

   ```sh
   createdb gator
   ```

2. **Run the migrations** against it:

   ```sh
   goose -dir sql/schema postgres "postgres://username:@localhost:5432/gator?sslmode=disable" up
   ```

3. **Install gator:**

   ```sh
   go install github.com/PokeMasterCP/gator@latest
   ```

4. **Create `.gatorconfig.json` in your home directory** with your connection
   string. Gator rewrites this file to track who is logged in, so it needs to be
   writable:

   ```json
   { "db_url": "postgres://username:@localhost:5432/gator?sslmode=disable" }
   ```

## Getting started

```sh
gator register alice                                  # create a user and log in
gator addfeed "Boot.dev Blog" https://blog.boot.dev/index.xml
gator agg 15s                                         # scrape every 15 seconds
```

`agg` runs until you stop it with Ctrl-C, so leave it going in one terminal and
use another to browse:

```sh
gator browse 5
```

## Commands

| Command | Description |
| --- | --- |
| `register <name>` | Create a user and log in as them |
| `login <name>` | Switch to an existing user |
| `users` | List registered users, marking the current one |
| `addfeed <name> <url>` | Add a feed and follow it (requires login) |
| `feeds` | List every feed in the database |
| `follow <url>` | Follow a feed someone else already added (requires login) |
| `following` | List the feeds you follow (requires login) |
| `unfollow <url>` | Stop following a feed (requires login) |
| `browse [limit]` | Show your most recent posts, default 2 (requires login) |
| `agg <interval>` | Run the scraper on a loop, e.g. `agg 15s`, `agg 1m` |
| `reset` | **Destructive** — wipes all users and their data |

## How it works

**Config and "login".** There is no authentication; `.gatorconfig.json` in your
home directory stores the database URL and the current user's name. `login` and
`register` just write that name back to the file.

**Middleware.** Commands that need a user are wrapped in `MiddlewareLoggedIn`,
which looks the current user up once and passes the record through to the
handler. That keeps the "who am I" lookup and its error handling in one place
instead of repeated at the top of seven handlers.

**Scraping.** `agg` takes a duration and uses a `time.Ticker` to repeat. Each
tick it asks Postgres for the feed whose `last_fetched_at` is oldest, fetches
and parses its XML, and inserts each item. Duplicate posts are expected on every
pass, so a unique-violation on the post URL is detected and ignored rather than
treated as a failure.

**Follows** are a join table between users and feeds, which is what lets several
users share a feed while each seeing their own timeline in `browse`.

**Queries** are written as plain SQL in `sql/queries` and compiled by sqlc into
type-safe Go in `internal/database` — that generated code is committed and
should not be edited by hand.

## Project layout

```
main.go                 wires up the database, registers commands, dispatches
internal/config/        config file handling, command registry, and all handlers
internal/rss/           fetching and parsing RSS XML
internal/database/      sqlc-generated queries (generated — do not edit)
sql/schema/             goose migrations
sql/queries/            sqlc query definitions
```

## Development

After editing anything in `sql/queries`:

```sh
sqlc generate
```
