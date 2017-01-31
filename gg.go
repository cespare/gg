package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
)

var commands = map[string]func(client *github.Client, args []string){
	"repos": listRepos,
}

func usage() {
	fmt.Fprintf(
		os.Stderr,
		"usage: %s command\nwhere command is one of\n",
		os.Args[0],
	)
	var cmds []string
	for cmd := range commands {
		cmds = append(cmds, cmd)
	}
	sort.Strings(cmds)
	for _, cmd := range cmds {
		fmt.Fprintf(os.Stderr, "\t%s\n", cmd)
	}
	fmt.Fprintf(
		os.Stderr,
		"Run %s command -h to see more about a particular command.\n",
		os.Args[0],
	)
	os.Exit(1)
}

func main() {
	log.SetFlags(0)
	if len(os.Args) < 2 || os.Args[1] == "help" || os.Args[1] == "-h" {
		usage()
	}
	f, ok := commands[os.Args[1]]
	if !ok {
		log.Fatalf("No command %q", os.Args[1])
	}
	token, err := loadToken()
	if err != nil {
		log.Fatalln("Error loading gg token:", err)
	}
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	tc := oauth2.NewClient(oauth2.NoContext, ts)
	client := github.NewClient(tc)
	f(client, os.Args[2:])
}

func listRepos(client *github.Client, args []string) {
	fs := flag.NewFlagSet("gg", flag.ExitOnError)
	user := fs.String("u", "", "Username (if different from credentials)")
	usage := func() {
		fmt.Fprintln(os.Stderr, `usage: repos [flags]
where flags are:`)
		fs.PrintDefaults()
	}
	fs.Usage = usage
	fs.Parse(args)
	if fs.NArg() > 0 {
		usage()
		os.Exit(1)
	}

	opt := &github.RepositoryListOptions{Type: "owner"}
	tw := tabwriter.NewWriter(os.Stdout, 0, 1, 4, ' ', 0)
	for {
		repos, resp, err := client.Repositories.List(*user, opt)
		if err != nil {
			log.Fatal(err)
		}
		for _, repo := range repos {
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
