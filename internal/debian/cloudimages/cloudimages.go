// Copyright 2026 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package cloudimages

import (
	"bufio"
	"context"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"hash"
	"net/http"
	"strings"

	"github.com/ironcore-dev/os-images/internal/debian"
	"github.com/ironcore-dev/os-images/internal/httpfs"
)

const DefaultBaseURL = "https://cloud.debian.org/images/cloud"

type Repository struct {
	httpFS *httpfs.HTTPFS
}

func NewRepository(httpFS *httpfs.HTTPFS) *Repository {
	return &Repository{
		httpFS: httpFS,
	}
}

func DefaultRepository() *Repository {
	return &Repository{
		httpFS: httpfs.New(DefaultBaseURL, http.DefaultClient),
	}
}

// filename returns the URL of a file in the latest directory of the codename.
func (r *Repository) filename(codename debian.Codename, version Version, filename string) string {
	return fmt.Sprintf("%s/%s/%s", codename, version, filename)
}

func (r *Repository) OpenFile(ctx context.Context, codename debian.Codename, version Version, filename string) (httpfs.File, error) {
	path := r.filename(codename, version, filename)

	return r.httpFS.Open(ctx, path)
}

const SHA512SumsFilename = "SHA512SUMS"

func (r *Repository) Checksums(ctx context.Context, codename debian.Codename, version Version) (*Checksums, error) {
	f, err := r.OpenFile(ctx, codename, version, SHA512SumsFilename)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()

	hashByFilename := make(map[string][]byte)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		// Format: "<hash>  <filename>" (two spaces between hash and filename)
		parts := strings.SplitN(line, "  ", 2)
		if len(parts) != 2 {
			continue
		}

		expectedHash, err := hex.DecodeString(strings.TrimSpace(parts[0]))
		if err != nil {
			continue
		}

		filename := strings.TrimSpace(parts[1])
		hashByFilename[filename] = expectedHash
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return &Checksums{
		newHash:        sha512.New,
		hashByFilename: hashByFilename,
	}, nil
}

type Checksums struct {
	newHash        func() hash.Hash
	hashByFilename map[string][]byte
}

func (c *Checksums) NewHash() hash.Hash {
	return c.newHash()
}

func (c *Checksums) Checksum(filename string) ([]byte, error) {
	expectedHash, ok := c.hashByFilename[filename]
	if !ok {
		return nil, fmt.Errorf("no checksums found for %q", filename)
	}

	return expectedHash, nil
}
