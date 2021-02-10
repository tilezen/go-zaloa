package fetcher

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/tilezen/go-zaloa/pkg/common"
)

type httpFetcher struct {
	baseURL string
	client  *http.Client
}

func (h httpFetcher) GetTile(ctx context.Context, t common.Tile, kind common.TileKind) (*FetchResponse, error) {
	url := fmt.Sprintf("%s/%s/%s.png", h.baseURL, kind, t)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("error building url %s: %w", url, err)
	}

	resp, err := h.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error fetching %s: %w", url, err)
	}

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response for %s: %w", url, err)
	}

	err = resp.Body.Close()
	if err != nil {
		return nil, fmt.Errorf("error closing response body for %s: %w", url, err)
	}

	responseData := &FetchResponse{
		Data: data,
		Tile: t,
	}

	log.Printf("Retrieved %s", url)

	return responseData, nil
}
