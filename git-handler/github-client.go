package githandler

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	//"strings"

	"github.com/google/go-github/v31/github"
	"github.com/gookit/color"
	"golang.org/x/oauth2"
)

// GithubClient incapsulate github client api
type GithubClient struct {
	ctx    context.Context
	client *github.Client
}

// ConnectGithub connects to gitlab
// my connect token is: remote-api-token
func ConnectGithub(provider, secret string) *GithubClient {
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: secret},
	)
	tc := oauth2.NewClient(ctx, ts)

	client := github.NewClient(tc)
	// client.SetBaseURL("https://git.mydomain.com/api/v4")

	return &GithubClient{ctx, client}
}

// ProviderName return docker registry provider name
func (c *GithubClient) ProviderName() string {
	return "gitHUB"
}

// LoadAssets load tag of docker image for project
func (c *GithubClient) LoadAssets(groupName, projectName, mode string) ([]byte, error) {
	organization, _, err := c.client.Organizations.Get(c.ctx, groupName)
	if err != nil {
		return nil, fmt.Errorf("find organization `%s` error: %v", groupName, err)
	}

	login := organization.GetLogin()
	if len(login) == 0 {
		return nil, fmt.Errorf("organization `%s` does not have login", groupName)
	}

	repository, _, err := c.client.Repositories.Get(c.ctx, login, projectName)
	if err != nil {
		return nil, fmt.Errorf("find project `%s` error: %v", projectName, err)
	}

	releases, _, err := c.client.Repositories.ListReleases(c.ctx, login, repository.GetName(), &github.ListOptions{
		Page: 0,
		PerPage: 3,
	})
	if err != nil {
		return nil, fmt.Errorf("list releases of project `%s` error: %v", projectName, err)
	}
	for _, rel := range releases {
		color.FgGray.Print("   release tag: ")
		color.FgCyan.Printf("%s", rel.GetTagName())
		color.FgGray.Printf(" from %s ", rel.GetPublishedAt())
		color.FgGray.Println(" -- gitHUB")
		/*
		if !strings.HasSuffix(rel.GetTagName(), mode) {
			color.FgGray.Println("   wrong mode(suffix); must be: " + mode)
			continue
		}
		*/
		for _, asset := range rel.Assets {
			assetName := asset.GetName()
			assetContentType := asset.GetContentType()
			color.FgGray.Print("   asset: ")
			color.FgCyan.Printf("%s", assetName)
			color.FgGray.Printf(" content-type: %s ", assetContentType)
			println()
			if assetContentType == "application/zip" {
				reader, redirect, err := c.client.Repositories.DownloadReleaseAsset(c.ctx, login, repository.GetName(), asset.GetID(), http.DefaultClient)
				if err != nil {
					return nil, fmt.Errorf("download asset. s of project `%s` error: %v", projectName, err)
				} else if redirect != "" {
					color.FgGray.Printf("   follow redirect %s to download asset\n", redirect)
				} else {
					defer reader.Close()

					body, err := ioutil.ReadAll(reader)
					if err != nil {
						return nil, fmt.Errorf("read assets of project `%s` error: %v", projectName, err)
					}
					// fmt.Printf("size of body: %v\n", len(body))
					return body, nil
				}
			}
		}

		return nil, nil
	}

	return nil, nil
}
