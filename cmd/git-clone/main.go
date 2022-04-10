package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/exec"
	"text/tabwriter"

	"github.com/gertd/git-clone/pkg/version"
	"github.com/google/go-github/github"
	"github.com/urfave/cli"
	"golang.org/x/oauth2"
)

// command line args.
const (
	GitHost  = "host"
	GitUser  = "user"
	GitOrg   = "org"
	GitToken = "token"
	DryRun   = "dryrun"
	Verbose  = "verbose"
)

// environment variables.
const (
	GitHostEnv  = "GIT_HOST"
	GitUserEnv  = "GIT_USER"
	GitOrgEnv   = "GIT_ORG"
	GitTokenEnv = "GIT_TOKEN"
)

// command line usage.
const (
	GitHostUsage  = "GitHub Enterprise host address"
	GitUserUsage  = "GitHub user"
	GitOrgUsage   = "GitHub organization"
	GitTokenUsage = "GitHub private access token"
	DryRunUsage   = "dryrun (no-exec) mode"
	VerboseUsage  = "verbose output"
)

// command literals.
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
	app.Version = version.GetInfo().String()
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:     GitUser,
			Value:    "",
			Usage:    GitUserUsage,
			EnvVar:   GitUserEnv,
			Required: true,
		},
		cli.StringFlag{
			Name:   GitOrg,
			Value:  "",
			Usage:  GitOrgUsage,
			EnvVar: GitOrgEnv,
		},
		cli.StringFlag{
			Name:     GitToken,
			Value:    "",
			Usage:    GitTokenUsage,
			EnvVar:   GitTokenEnv,
			Required: true,
		},
		cli.StringFlag{
			Name:   GitHost,
			Value:  "",
			Usage:  GitHostUsage,
			EnvVar: GitHostEnv,
		},
		cli.BoolFlag{
			Name:  DryRun,
			Usage: DryRunUsage,
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

func gitClone(c *cli.Context) error { //nolint:funlen,gocognit
	var (
		err    error
		ctx    = context.Background()
		client *github.Client
		repos  []*github.Repository
		resp   *github.Response
	)

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', 0)

	gitToken := c.GlobalString(GitToken)
	gitOrg := c.GlobalString(GitOrg)
	gitUser := c.GlobalString(GitUser)
	gitHost := c.GlobalString(GitHost)
	dryRun := c.GlobalBool(DryRun)
	verbose := c.GlobalBool(Verbose)

	if verbose {
		fmt.Printf("user: [%s] org: [%s]\n", gitUser, gitOrg)
	}

	userinfo := url.UserPassword(gitUser, gitToken)
	userinfo.Password()

	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: gitToken},
	)
	tc := oauth2.NewClient(ctx, ts)

	if gitHost == "" {
		client = github.NewClient(tc)
	} else if client, err = github.NewEnterpriseClient(gitHost, gitHost, tc); err != nil {
		return err
	}

	page := 1

	cmdName := GitCmd

	fmt.Fprintf(w, "%s\t%s\t%s\n",
		"name",
		"operation",
		"repo",
	)

	for {
		if gitOrg == "" {
			opt := &github.RepositoryListOptions{
				Type:        "all",
				ListOptions: github.ListOptions{PerPage: 10, Page: page},
			}

			repos, resp, err = client.Repositories.List(ctx, gitUser, opt)
			if err != nil {
				return err
			}
		} else {
			opt := &github.RepositoryListByOrgOptions{
				Type:        "all",
				ListOptions: github.ListOptions{PerPage: 10, Page: page},
			}

			repos, resp, err = client.Repositories.ListByOrg(ctx, c.GlobalString(GitOrg), opt)
			if err != nil {
				return err
			}
		}

		for _, repo := range repos {
			// check if directory/.git exists
			checkPath := "../" + *repo.FullName + "/.git"

			if _, err := os.Stat(checkPath); os.IsNotExist(err) {
				fmt.Fprintf(w, "%s\t%s\t%s\n",
					*repo.FullName,
					"clone",
					*repo.CloneURL)

				if !dryRun {
					gitURL, err := url.Parse(*repo.CloneURL)
					if err != nil {
						return err
					}

					gitURL.User = userinfo

					cmdArgs := []string{GitClone, gitURL.String()}

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
				}
			} else {
				fmt.Fprintf(w, "%s\t%s\t%s\n",
					*repo.FullName,
					"exist",
					*repo.CloneURL)
			}
		}

		if resp.NextPage == 0 {
			break
		}

		page = resp.NextPage
	}
	w.Flush()

	return nil
}
