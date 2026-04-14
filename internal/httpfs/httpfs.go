// Copyright 2026 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package httpfs

import (
	"context"
	"fmt"
	"io"
	"net/http"
)

type HTTPFS struct {
	baseURL    string
	httpClient *http.Client
}

func New(baseURL string, httpClient *http.Client) *HTTPFS {
	return &HTTPFS{
		baseURL:    baseURL,
		httpClient: httpClient,
	}
}

func Default(baseURL string) *HTTPFS {
	return New(baseURL, http.DefaultClient)
}

func (r *HTTPFS) fileURL(path string) string {
	return fmt.Sprintf("%s/%s", r.baseURL, path)
}

func (r *HTTPFS) Open(ctx context.Context, filename string) (File, error) {
	url := r.fileURL(filename)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error doing request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		data, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		return nil, fmt.Errorf("http error %d for %s (%s)", resp.StatusCode, url, string(data))
	}

	return &file{
		name:       filename,
		ReadCloser: resp.Body,
		size:       resp.ContentLength,
	}, nil
}

type file struct {
	name string
	io.ReadCloser
	size int64
}

func (f *file) Name() string {
	return f.name
}

func (f *file) Size() int64 {
	return f.size
}

type File interface {
	io.ReadCloser
	Name() string
	Size() int64
}
