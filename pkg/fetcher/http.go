package fetcher

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"

	"github.com/tilezen/go-zaloa/pkg/common"
)

type httpFetcher struct {
	baseURL string
	client  *http.Client
}

func (h httpFetcher) GetTile(ctx context.Context, t common.Tile, kind common.TileKind, version common.TileVersion) (*FetchResponse, error) {
	u, err := url.JoinPath(h.baseURL, string(version), string(kind), t.String()+".png")
	if err != nil {
		return nil, fmt.Errorf("error joining url: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
	if err != nil {
		return nil, fmt.Errorf("error building url %s: %w", u, err)
	}

	resp, err := h.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error fetching %s: %w", u, err)
	}

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response for %s: %w", u, err)
	}

	err = resp.Body.Close()
	if err != nil {
		return nil, fmt.Errorf("error closing response body for %s: %w", u, err)
	}

	responseData := &FetchResponse{
		Data: data,
		Tile: t,
	}

	log.Printf("Retrieved %s", u)

	return responseData, nil
}
