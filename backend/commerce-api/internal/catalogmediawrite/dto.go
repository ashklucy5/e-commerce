package catalogmediawrite

type CreateImageRequest struct {
	VariantID  string `json:"variant_id"`
	StorageKey string `json:"storage_key"`
	AltText    string `json:"alt_text"`
	SortOrder  int    `json:"sort_order"`
	IsPrimary  bool   `json:"is_primary"`
}

type Create360FrameRequest struct {
	VariantID  string `json:"variant_id"`
	StorageKey string `json:"storage_key"`
	FrameIndex int    `json:"frame_index"`
}

type Upsert3DModelRequest struct {
	VariantID  string `json:"variant_id"`
	StorageKey string `json:"storage_key"`
	PosterURL  string `json:"poster_url"`
}
