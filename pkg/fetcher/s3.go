package fetcher

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"path"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"

	"github.com/tilezen/go-zaloa/pkg/common"
)

type s3tileFetcher struct {
	s3Bucket      string
	requesterPays bool
	s3            s3iface.S3API
}

func (s s3tileFetcher) GetTile(ctx context.Context, t common.Tile, kind common.TileKind, version common.TileVersion) (*FetchResponse, error) {
	s3Key := path.Clean(fmt.Sprintf("%s/%s/%s.png", version, kind, t))

	input := &s3.GetObjectInput{
		Bucket: aws.String(s.s3Bucket),
		Key:    aws.String(s3Key),
	}

	if s.requesterPays {
		input.RequestPayer = aws.String("requester")
	}

	resp, err := s.s3.GetObjectWithContext(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("error fetching Tile s3://%s/%s: %w", s.s3Bucket, s3Key, err)
	}

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading data for Tile s3://%s/%s: %w", s.s3Bucket, s3Key, err)
	}

	err = resp.Body.Close()
	if err != nil {
		return nil, fmt.Errorf("error closing response for Tile s3://%s/%s: %w", s.s3Bucket, s3Key, err)
	}

	responseData := &FetchResponse{
		Data: data,
		Tile: t,
	}

	log.Printf("Retrieved s3://%s/%s", s.s3Bucket, s3Key)

	return responseData, nil
}
