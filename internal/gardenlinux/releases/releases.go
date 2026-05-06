package releases

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/google/go-github/v85/github"
)

const (
	gardenlinuxOwner = "gardenlinux"
	gardenlinuxRepo  = "gardenlinux"
)

type Releases struct {
	owner                 string
	repo                  string
	client                *github.Client
	followRedirectsClient *http.Client
}

func NewReleases(
	owner, repo string,
	client *github.Client,
	followRedirectsClient *http.Client,
) *Releases {
	return &Releases{
		owner:                 owner,
		repo:                  repo,
		client:                client,
		followRedirectsClient: followRedirectsClient,
	}
}

func DefaultReleases() *Releases {
	githubClient := github.NewClient(nil)
	if githubToken := os.Getenv("GITHUB_TOKEN"); githubToken != "" {
		githubClient = githubClient.WithAuthToken(githubToken)
	}

	return NewReleases(
		gardenlinuxOwner,
		gardenlinuxRepo,
		githubClient,
		http.DefaultClient,
	)
}

type Asset struct {
	releases *Releases
	asset    *github.ReleaseAsset
}

func (a *Asset) Name() string {
	return a.asset.GetName()
}

func (a *Asset) Base() string {
	return strings.TrimSuffix(a.asset.GetName(), ".tar.xz")
}

func (a *Asset) Open(ctx context.Context) (io.ReadCloser, error) {
	rc, _, err := a.releases.client.Repositories.DownloadReleaseAsset(ctx,
		a.releases.owner,
		a.releases.repo,
		a.asset.GetID(),
		a.releases.followRedirectsClient,
	)
	return rc, err
}

func (r *Releases) GetAsset(ctx context.Context, flavor, arch, tag string) (*Asset, error) {
	release, _, err := r.client.Repositories.GetReleaseByTag(ctx, r.owner, r.repo, tag)
	if err != nil {
		return nil, err
	}

	for _, asset := range release.Assets {
		assetName := asset.GetName()
		if !strings.HasSuffix(assetName, ".tar.xz") || strings.HasSuffix(assetName, ".logs.tar.xz") {
			continue
		}

		assetPrefix := fmt.Sprintf("%s-%s-", flavor, arch)
		if !strings.HasPrefix(assetName, assetPrefix) {
			continue
		}

		return &Asset{
			releases: r,
			asset:    asset,
		}, nil
	}
	return nil, fmt.Errorf("no asset found for %s/%s", flavor, arch)
}
