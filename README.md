# Gator
### A command line RSS feed scraper utility
---
## Requirements
* Go
* PostgreSQL

## Installation
1. Install PostgreSQL
2. Set up the database tables using Goose (preferred) and the schemas in `/sql/schema` or manually
3. Install Gator using Go
```
go install github.com/PokeMasterCP/gator@latest
```
3. Create `.gatorconfig.json` in your home directory and populate it with the PostgreSQL URL
```
{"db_url":"postgres://username:@localhost:5432/gator?sslmode=disable"}
```

## How to use
1. Register your username with `gator register <username>`
2. Add a feed to follow with `gator addfeed <feed name> <feed URL>`
3. Start gator service which will pull latest posts from feeds in the specified interval `gator agg 15s`

## Command List
|Command|Description|
|-|-|
|login|Logs in as the specified user, requires user as argument `gator login username`|
|register|Registers the user in the PostgreSQL database, requires user as argument `gator register username`|
|reset|**DANGEROUS**, will clear database|
|users|Will list currently registered users|
|agg|Starts gator service, requires time interval for to refresh `gator agg 15s`|
|addfeed|Adds a feed to the database and follows it for your user account, requires feed name and URL `gator addfeed example http://example.com`|
|feeds|Lists all the feeds in the database|
|follow|Follows a feed in the database, requires a feed URL as an argument `gator follow feedURL`|
|following|Lists all of your followed feeds|
|unfollow|Unfollows a feed, requires a feed URL as an argument `gator unfollow feedURL`|
|browse|Lists the specified number of posts, will default to 2 if no number is provided|
