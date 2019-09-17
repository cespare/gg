package main

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/cespare/subcmd"
	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
)

var commands = []subcmd.Command{
	{
		Name:        "repos",
		Description: "list repositories",
		Do:          listRepos,
	},
}

func main() {
	log.SetFlags(0)
	subcmd.Run(commands)
}

func listRepos(args []string) {
	var (
		fs              = flag.NewFlagSet("gg repos", flag.ExitOnError)
		user            = fs.String("u", "", "Username (if different from credentials)")
		public          = fs.Bool("public", false, "Only include public repos")
		private         = fs.Bool("private", false, "Only include private repos")
		includeForks    = fs.Bool("includeforks", false, "Include forked repos")
		includeArchived = fs.Bool("includearchived", false, "Include archived repos")
		sortBy          = fs.String("sortby", "name", "Sort by `field`: one of name, created, updated, pushed")
	)
	fs.Parse(args)

	if fs.NArg() > 0 {
		fs.Usage()
		os.Exit(1)
	}
	if *public && *private {
		log.Fatal("Only one of -public and -private may be given")
	}
	switch *sortBy {
	case "name":
		*sortBy = "full_name"
	case "created", "updated", "pushed":
	default:
		log.Fatalf("Unknown -sortby option %q", *sortBy)
	}

	client, err := makeGHClient()
	if err != nil {
		log.Fatalln("Cannot create GitHub client:", err)
	}

	opt := &github.RepositoryListOptions{
		Affiliation: "owner",
		Sort:        *sortBy,
	}
	if *public {
		opt.Visibility = "public"
	}
	if *private {
		opt.Visibility = "private"
	}
	tw := tabwriter.NewWriter(os.Stdout, 0, 1, 4, ' ', 0)
	for {
		repos, resp, err := client.Repositories.List(context.Background(), *user, opt)
		if err != nil {
			log.Fatal(err)
		}
		for _, repo := range repos {
			if repo.Fork != nil && *repo.Fork && !*includeForks {
				continue
			}
			if repo.Archived != nil && *repo.Archived && !*includeArchived {
				continue
			}
			var desc string
			if repo.Description != nil {
				desc = *repo.Description
			}
			fmt.Fprintf(tw, "%s\t%s\n", *repo.Name, desc)
		}
		if resp.NextPage == 0 {
			break
		}
		opt.ListOptions.Page = resp.NextPage
	}
	tw.Flush()
}

func makeGHClient() (*github.Client, error) {
	token, err := loadToken()
	if err != nil {
		return nil, fmt.Errorf("Error loading GitHub token: %w", err)
	}
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	tc := oauth2.NewClient(oauth2.NoContext, ts)
	return github.NewClient(tc), nil
}

func loadToken() (string, error) {
	path, ok := os.LookupEnv("GG_TOKEN")
	if !ok {
		path = os.ExpandEnv("$HOME/gg.token")
	}
	contents, err := ioutil.ReadFile(path)
	if err != nil {
		return "", err
	}
	token := strings.Split(string(contents), "\n")[0]
	return token, nil
}
