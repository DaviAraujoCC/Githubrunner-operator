package github

import (
	"context"

	"github.com/google/go-github/v39/github"
	"golang.org/x/oauth2"
)

type Client struct {
	*github.Client
	OrgName string
}

func NewClient(token, orgName string) (*Client, error) {

	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)

	return &Client{client, orgName}, nil
}

func (c *Client) ListOrganizationRunners() ([]*github.Runner, error) {

	opts := github.ListOptions{PerPage: 50}
	var runners []*github.Runner
	for {
		list, res, err := c.Actions.ListOrganizationRunners(context.Background(), c.OrgName, &opts)
		if err != nil {
			return nil, err
		}
		runners = append(runners, list.Runners...)
		if res.NextPage == 0 {
			break
		}
	}
	return runners, nil
}
