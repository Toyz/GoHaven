package GoHaven

type WallHaven struct {
	// TODO: Add jar info here
}

type SearchInfo struct {
	CurrentPage int
	TotalPages  int
	End         bool
	Results     []SearchResult
}

type SearchResult struct {
	ImageID   ID
	Thumbnail string
	Purity    string
	Category  string
	Width     int
	Height    int
	Favorites int
	Link      string
}

type Color struct {
	HEX  string
	RGB  string
	Link string
}

type Uploader struct {
	Name           string
	Profile        string
	ProfilePicture string
}

type Tag struct {
	TagID  int
	Name   string
	Purity string
	Link   string
}

type ImageDetail struct {
	ImageID    ID
	URL        string
	UploadedOn string `json:,omitempty`
	Category   string
	Purity     string
	Views      int
	Favorites  int
	Link       string
	Uploader   Uploader
	Tags       []Tag
	Colors     []Color
}
