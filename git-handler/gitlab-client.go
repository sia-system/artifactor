package githandler

import (
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"

	"github.com/gookit/color"
	gitlab "github.com/xanzy/go-gitlab"
)

// TODO: if provider is docker.pkg.github.com then use github
// for provier: registry.gitlab.com
// Connect to gitlab
// my connect token is: remote-api-token

// GitlabClient incapsulate gitlab client api
type GitlabClient struct {
	httpclient *http.Client
	client     *gitlab.Client
}

// ConnectGitlab connects to gitlab
func ConnectGitlab(provider, secret string) *GitlabClient {
	// fmt.Printf("GITLAB api secret token: %s and custom endpoint: %s\n", secret, provider)

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	httpclient := &http.Client{Transport: tr}

	git, err := gitlab.NewClient(secret,
		gitlab.WithBaseURL("https://"+provider+"/api/v4"),
		gitlab.WithHTTPClient(httpclient),
	)
	if err != nil {
		panic(err.Error())
	}

	return &GitlabClient{httpclient, git}
}

// ProviderName return docker registry provider name
func (c *GitlabClient) ProviderName() string {
	return "gitLAB"
}

// LoadAssets load tag of docker image for project
func (c *GitlabClient) LoadAssets(groupName, projectName, mode string) ([]byte, error) {
	groups, _, err := c.client.Groups.SearchGroup(groupName)
	if err != nil {
		return nil, fmt.Errorf("find group `%s` error: %v", groupName, err)
	}

	for _, g := range groups {
		owned := true
		opt := &gitlab.ListGroupProjectsOptions{
			Owned:  &owned,
			Search: &projectName,
		}

		projects, _, err := c.client.Groups.ListGroupProjects(g.ID, opt)
		if err != nil {
			return nil, fmt.Errorf("find project `%s` error: %v", projectName, err)
		}

		for _, p := range projects {
			if p.Path != projectName {
				continue
			}

			color.FgGray.Print("    project: ")
			println(p.PathWithNamespace)

			opt := &gitlab.ListReleasesOptions{
				Page:    0,
				PerPage: 3,
			}

			releases, _, err := c.client.Releases.ListReleases(p.ID, opt)
			if err != nil {
				return nil, fmt.Errorf("list release error: %v", err)
			}
			for _, rel := range releases {
				color.FgGray.Print("   release tag: ")
				color.FgCyan.Printf("%s", rel.TagName)
				color.FgGray.Printf(" from %s ", rel.CreatedAt)
				color.FgGray.Println(" -- gitLAB")
				/*
				if !strings.HasSuffix(rel.TagName, mode) {
					color.FgGray.Println("     wrong mode(suffix); must be: " + mode)
					continue
				}
				*/
				for _, link := range rel.Assets.Links {
					color.FgGray.Print("     link: ")
					color.FgCyan.Printf("%s", link.Name)
					color.FgGray.Printf(" from %s ", link.URL)
					if link.External {
						color.FgGray.Print(" EX ")
					}
					println()

					if !link.External && strings.HasPrefix(link.Name, "Deployments") {
						color.FgGray.Println("     download assets")

						// extract job-id from url
						jobsIndex := strings.Index(link.URL, "/jobs/")
						artifactsIndex := strings.Index(link.URL, "/artifacts/")
						if jobsIndex < 10 || artifactsIndex < 15 {
							return nil, fmt.Errorf("Invalid artifact url of project `%s`", projectName)
						}

						jobID, err := strconv.Atoi(link.URL[jobsIndex+6 : artifactsIndex])
						if err != nil {
							return nil, fmt.Errorf("Invalid job id of project `%s` error: %v", projectName, err)
						}

						reader, _, err := c.client.Jobs.GetJobArtifacts(p.ID, jobID, nil)
						if err != nil {
							return nil, fmt.Errorf("Download assets of project `%s` error: %v", projectName, err)
						}

						body, err := ioutil.ReadAll(reader)
						if err != nil {
							return nil, fmt.Errorf("read assets of project `%s` error: %v", projectName, err)
						}

						return body, nil
					}
				}
				return nil, nil
			}

		}
	}

	return nil, nil
}
