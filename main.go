package main

import (
	"bufio"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/exec"

	"github.com/codegangsta/cli"
	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
)

// command line args
const (
	GitUser  = "user"
	GitOrg   = "org"
	GitToken = "token"
	Verbose  = "verbose"
)

// environment variables
const (
	GitUserEnv  = "GIT_USER"
	GitOrgEnv   = "GIT_ORG"
	GitTokenEnv = "GIT_TOKEN"
)

// command line usage
const (
	GitUserUsage  = "GitHub user id"
	GitOrgUsage   = "GitHub organization"
	GitTokenUsage = "GitHub private access token"
	VerboseUsage  = "verbose output"
)

// command literals
const (
	AppName  = "git-clone"
	AppUsage = "clone all GitHub repos not in current directory"
	GitCmd   = "git"
	GitClone = "clone"
)

func main() {

	app := cli.NewApp()
	app.Name = AppName
	app.Usage = AppUsage
	app.Action = gitClone
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   GitUser,
			Value:  "",
			Usage:  GitUserUsage,
			EnvVar: GitUserEnv,
		},
		cli.StringFlag{
			Name:   GitOrg,
			Value:  "",
			Usage:  GitOrgUsage,
			EnvVar: GitOrgEnv,
		},
		cli.StringFlag{
			Name:   GitToken,
			Value:  "",
			Usage:  GitTokenUsage,
			EnvVar: GitTokenEnv,
		},
		cli.BoolFlag{
			Name:  Verbose,
			Usage: VerboseUsage,
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err.Error())
	}

	os.Exit(1)
}

func gitClone(c *cli.Context) error {

	gitToken := c.GlobalString(GitToken)
	gitOrg := c.GlobalString(GitOrg)
	gitUser := c.GlobalString(GitUser)
	verbose := c.GlobalBool(Verbose)

	if verbose {
		fmt.Printf("user: [%s] org: [%s]\n", gitUser, gitOrg)
	}

	userinfo := url.UserPassword(gitUser, gitToken)

	userinfo.Password()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: gitToken},
	)
	tc := oauth2.NewClient(oauth2.NoContext, ts)

	client := github.NewClient(tc)

	page := 1

	cmdName := GitCmd

	var repos []*github.Repository
	var resp *github.Response
	var err error

	for {

		if len(gitOrg) == 0 {

			opt := &github.RepositoryListOptions{
				Type:        "all",
				ListOptions: github.ListOptions{PerPage: 10, Page: page},
			}

			repos, resp, err = client.Repositories.List(gitUser, opt)
			if err != nil {
				return err
			}

		} else {
			opt := &github.RepositoryListByOrgOptions{
				Type:        "all",
				ListOptions: github.ListOptions{PerPage: 10, Page: page},
			}

			repos, resp, err = client.Repositories.ListByOrg(c.GlobalString(GitOrg), opt)
			if err != nil {
				return err
			}
		}

		for _, repo := range repos {

			fmt.Printf("%s ", *repo.FullName)

			// check if directory/.git exists
			checkPath := "../" + *repo.FullName + "/.git"

			if _, err := os.Stat(checkPath); os.IsNotExist(err) {

				fmt.Printf("does not exist, cloning [%s]\n", *repo.CloneURL)

				url, err := url.Parse(*repo.CloneURL)
				if err != nil {
					return err
				}
				url.User = userinfo

				cmdArgs := []string{GitClone, url.String()}

				cmd := exec.Command(cmdName, cmdArgs...)
				cmdReader, err := cmd.StdoutPipe()
				if err != nil {
					fmt.Fprintln(os.Stderr, "Error: creating StdoutPipe for Cmd", err)
					return err
				}

				scanner := bufio.NewScanner(cmdReader)
				go func() {
					for scanner.Scan() {
						fmt.Printf("%s\n", scanner.Text())
					}
				}()

				err = cmd.Start()
				if err != nil {
					fmt.Fprintln(os.Stderr, "Error: starting Cmd", err)
					return err
				}

				err = cmd.Wait()
				if err != nil {
					fmt.Fprintln(os.Stderr, "Error: waiting for Cmd", err)
				}

			} else {
				fmt.Printf("exists\n")
			}
		}

		if resp.NextPage == 0 {
			break
		}

		page = resp.NextPage
	}
	return nil
}
