package wishlist

import "project.local/commerce-api/internal/platform/pagination"

type ListResult struct {
	Items []Item
	Meta  pagination.Meta
}
