package catalogwrite

import (
	"context"
	"fmt"
	"regexp"
	"strings"
)

var productCodePrefixPattern = regexp.MustCompile(
	`^[A-Z0-9]{3}-[A-Z0-9]{3}$`,
)

func (s *Service) CategoryTree(
	ctx context.Context,
) ([]AdminCategory, error) {
	items, err := s.repository.ListAdminCategories(
		ctx,
	)
	if err != nil {
		return nil,
			fmt.Errorf(
				"list Admin category tree: %w",
				err,
			)
	}

	return buildAdminCategoryTree(items), nil
}

func (s *Service) CreateCategory(
	ctx context.Context,
	input CreateCategoryInput,
) (AdminCategory, error) {
	normalizeCreateCategoryInput(
		&input,
	)

	if err := validateCreateCategoryInput(
		input,
	); err != nil {
		return AdminCategory{}, err
	}

	return s.repository.CreateCategory(
		ctx,
		input,
	)
}

func normalizeCreateCategoryInput(
	input *CreateCategoryInput,
) {
	input.Name = strings.TrimSpace(
		input.Name,
	)

	input.Slug = strings.TrimSpace(
		input.Slug,
	)

	input.Description = strings.TrimSpace(
		input.Description,
	)

	input.ProductCodePrefix = strings.ToUpper(
		strings.TrimSpace(
			input.ProductCodePrefix,
		),
	)

	if input.ParentID != nil {
		parentID := strings.TrimSpace(
			*input.ParentID,
		)

		if parentID == "" {
			input.ParentID = nil
		} else {
			input.ParentID = &parentID
		}
	}
}

func validateCreateCategoryInput(
	input CreateCategoryInput,
) error {
	if input.Name == "" {
		return ErrCategoryNameRequired
	}

	if input.SortOrder < 0 {
		return ErrInvalidCategorySortOrder
	}

	if input.ProductCodePrefix != "" &&
		!productCodePrefixPattern.MatchString(
			input.ProductCodePrefix,
		) {
		return ErrInvalidProductCodePrefix
	}

	return nil
}

func buildAdminCategoryTree(
	items []AdminCategory,
) []AdminCategory {
	byParent := make(
		map[string][]AdminCategory,
	)

	roots := make(
		[]AdminCategory,
		0,
	)

	for _, item := range items {
		item.Children = nil

		if item.ParentID == nil {
			roots = append(
				roots,
				item,
			)

			continue
		}

		byParent[*item.ParentID] = append(
			byParent[*item.ParentID],
			item,
		)
	}

	var attachChildren func(
		AdminCategory,
	) AdminCategory

	attachChildren = func(
		item AdminCategory,
	) AdminCategory {
		children := byParent[item.ID]

		if len(children) == 0 {
			return item
		}

		item.Children = make(
			[]AdminCategory,
			0,
			len(children),
		)

		for _, child := range children {
			item.Children = append(
				item.Children,
				attachChildren(child),
			)
		}

		return item
	}

	for index := range roots {
		roots[index] = attachChildren(
			roots[index],
		)
	}

	return roots
}
