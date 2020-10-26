package common

import (
	"fmt"
	"math"
	"strconv"
)

type Tile struct {
	Z, X, Y uint
}

func (t Tile) IsValid() bool {
	if t.Z < 0 || t.X < 0 || t.Y < 0 {
		return false
	}

	xyLimit := uint(math.Pow(2, float64(t.Z)))
	if t.X >= xyLimit || t.Y >= xyLimit {
		return false
	}

	return true
}

func (t Tile) String() string {
	return fmt.Sprintf("%d/%d/%d", t.Z, t.X, t.Y)
}

func ParseTile(zStr string, xStr string, yStr string) (*Tile, error) {
	z, err := strconv.ParseUint(zStr, 10, 32)
	if err != nil {
		return nil, fmt.Errorf("error parsing z: %w", err)
	}

	x, err := strconv.ParseUint(xStr, 10, 32)
	if err != nil {
		return nil, fmt.Errorf("error parsing x: %w", err)
	}

	y, err := strconv.ParseUint(yStr, 10, 32)
	if err != nil {
		return nil, fmt.Errorf("error parsing y: %w", err)
	}

	t := &Tile{Z: uint(z), X: uint(x), Y: uint(y)}

	if !t.IsValid() {
		return nil, fmt.Errorf("invalid Tile coordinate: %s", t)
	}

	return t, nil
}
