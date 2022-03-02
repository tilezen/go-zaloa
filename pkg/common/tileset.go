package common

type TileKind string
type TileEncoding string
type TileVersion string

const (
	TileType_TERRARIUM = TileKind("terrarium")
	TileType_NORMAL    = TileKind("normal")
	TileEncoding_PNG   = TileEncoding("png")
	TileEncoding_WEBP  = TileEncoding("webp")
	// TileVersion_V1 is the enum for v1 tiles. The string is empty because v1 tiles sit at the root of the S3 bucket.
	TileVersion_V1     = TileVersion("")
	TileVersion_V2     = TileVersion("v2")
)
