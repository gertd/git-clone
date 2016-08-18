package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/exec"

	"github.com/codegangsta/cli"
	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
)

// command line args
const (
	GitOrg   = "org"
	GitToken = "token"
)

// environment variables
const (
	GitOrgEnv   = "GIT_ORG"
	GitTokenEnv = "GIT_TOKEN"
)

// command line usage
const (
	GitOrgUsage   = "GitHub organization"
	GitTokenUsage = "GitHub private access token"
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
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err.Error())
	}

	os.Exit(1)
}

func gitClone(c *cli.Context) error {

	fmt.Printf("token %s\n", c.GlobalString(GitToken))
	fmt.Printf("org %s\n", c.GlobalString(GitOrg))

	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: c.GlobalString(GitToken)},
	)
	tc := oauth2.NewClient(oauth2.NoContext, ts)

	client := github.NewClient(tc)

	page := 1

	cmdName := GitCmd

	for {

		opt := &github.RepositoryListByOrgOptions{
			Type:        "all",
			ListOptions: github.ListOptions{PerPage: 10, Page: page},
		}

		repos, resp, err := client.Repositories.ListByOrg(c.GlobalString(GitOrg), opt)
		if err != nil {
			return err
		}

		for _, v := range repos {

			fmt.Printf("%s ", *v.FullName)

			// check if directory/.git exists
			checkPath := "../" + *v.FullName + "/.git"

			if _, err := os.Stat(checkPath); os.IsNotExist(err) {

				fmt.Printf("does not exist, cloning [%s]\n", *v.CloneURL)

				cmdArgs := []string{GitClone, *v.CloneURL}

				cmd := exec.Command(cmdName, cmdArgs...)
				cmdReader, err := cmd.StdoutPipe()
				if err != nil {
					fmt.Fprintln(os.Stderr, "Error: creating StdoutPipe for Cmd", err)
					return err
				}

				scanner := bufio.NewScanner(cmdReader)
				go func() {
					for scanner.Scan() {
						fmt.Printf("git out | %s\n", scanner.Text())
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
					// return err
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
