package catalogimport

type BatchSummary struct {
	ID                string `json:"id"`
	Status            string `json:"status"`
	TotalRows         int    `json:"total_rows"`
	ValidRows         int    `json:"valid_rows"`
	FailedRows        int    `json:"failed_rows"`
	CreatedProducts   int    `json:"created_products"`
	UpdatedProducts   int    `json:"updated_products"`
	CreatedVariants   int    `json:"created_variants"`
	UpdatedVariants   int    `json:"updated_variants"`
	CreatedCategories int    `json:"created_categories"`
}

type StageResult struct {
	Batch  BatchSummary      `json:"batch"`
	Errors []ValidationError `json:"errors"`
}
