package catalogmediawrite

type CreateImageInput struct {
	ProductID  string
	VariantID  string
	StorageKey string
	AltText    string
	SortOrder  int
	IsPrimary  bool
}

type ImageResult struct {
	ID        string  `json:"id"`
	ProductID string  `json:"product_id"`
	VariantID *string `json:"variant_id,omitempty"`
	URL       string  `json:"url"`
	AltText   *string `json:"alt_text,omitempty"`
	SortOrder int     `json:"sort_order"`
	IsPrimary bool    `json:"is_primary"`
}

type Create360FrameInput struct {
	ProductID  string
	VariantID  string
	StorageKey string
	FrameIndex int
}

type Frame360Result struct {
	ID         string  `json:"id"`
	ProductID  string  `json:"product_id"`
	VariantID  *string `json:"variant_id,omitempty"`
	StorageKey string  `json:"storage_key"`
	URL        string  `json:"url"`
	FrameIndex int     `json:"frame_index"`
}

type Upsert3DModelInput struct {
	ProductID  string
	VariantID  string
	StorageKey string
	PosterURL  string
}

type Model3DResult struct {
	ID         string  `json:"id"`
	ProductID  string  `json:"product_id"`
	VariantID  *string `json:"variant_id,omitempty"`
	StorageKey string  `json:"storage_key"`
	URL        string  `json:"url"`
	PosterURL  *string `json:"poster_url,omitempty"`
}
