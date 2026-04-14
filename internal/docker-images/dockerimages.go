// Copyright 2026 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package dockerimages

import (
	"archive/tar"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/docker/docker/api/types/build"
	"github.com/docker/docker/client"
	"github.com/docker/docker/errdefs"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/ironcore-dev/os-images/internal/ulog"
)

func HashedTagOf(name string, dockerfile []byte) string {
	shasum := sha256.Sum256(dockerfile)
	return fmt.Sprintf("docker.io/library/%s:%s", name, hex.EncodeToString(shasum[:]))
}

func BuildHashed(ctx context.Context, c *client.Client, name string, dockerfile []byte) (tag string, err error) {
	log := ulog.FromContext(ctx)

	tag = HashedTagOf(name, dockerfile)

	_, err = c.ImageInspect(ctx, tag)
	if err != nil {
		if _, ok := err.(errdefs.ErrNotFound); !ok {
			return "", fmt.Errorf("inspecting image %q: %v", tag, err)
		}
	}
	if err == nil {
		return tag, nil
	}

	buildContext, err := buildSingleDockerfileBuildContextTar(dockerfile)
	if err != nil {
		return "", err
	}

	group := log.Group(fmt.Sprintf("docker build %s", name), "Tag", tag)
	defer func() {
		if err != nil {
			group.Fail(err, "Error building docker image")
		} else {
			group.Success("Built image")
		}
	}()

	res, err := c.ImageBuild(ctx, bytes.NewReader(buildContext), build.ImageBuildOptions{
		PullParent: true,
		Remove:     true,
		Tags:       []string{tag},
		Version:    build.BuilderV1,
	})
	if err != nil {
		return "", fmt.Errorf("building image: %w", err)
	}
	defer func() { _ = res.Body.Close() }()

	decoder := json.NewDecoder(res.Body)
	for {
		var msg jsonmessage.JSONMessage
		if err := decoder.Decode(&msg); err != nil {
			if err == io.EOF {
				break
			}
			return "", fmt.Errorf("reading build output: %w", err)
		}

		if msg.Error != nil {
			return "", fmt.Errorf("building image: %s", msg.Error.Message)
		}

		switch {
		case msg.Stream != "":
			group.Info("Message", "Stream", strings.TrimSpace(msg.Stream))
		case msg.Status != "":
			group.Info("Message", "Status", strings.TrimSpace(msg.Status))
		case msg.Aux != nil:
		}
	}

	return tag, nil
}

func buildSingleDockerfileBuildContextTar(dockerfile []byte) ([]byte, error) {
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	if err := tw.WriteHeader(&tar.Header{
		Name: "Dockerfile",
		Size: int64(len(dockerfile)),
		Mode: 0644,
	}); err != nil {
		_ = tw.Close()
		return nil, err
	}
	if _, err := tw.Write(dockerfile); err != nil {
		_ = tw.Close()
		return nil, err
	}
	if err := tw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

type ClientCache struct {
	mu            sync.Mutex
	imageByClient map[*client.Client]string
	buildImage    func(ctx context.Context, client *client.Client) (string, error)
}

func NewClientCache(buildImage func(ctx context.Context, client *client.Client) (string, error)) *ClientCache {
	return &ClientCache{
		buildImage: buildImage,
	}
}

func (c *ClientCache) GetOrBuild(ctx context.Context, cl *client.Client) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if img, ok := c.imageByClient[cl]; ok {
		return img, nil
	}

	img, err := c.buildImage(ctx, cl)
	if err != nil {
		return "", fmt.Errorf("building image: %w", err)
	}

	if c.imageByClient == nil {
		c.imageByClient = make(map[*client.Client]string)
	}
	c.imageByClient[cl] = img

	return img, nil
}
