package category

// BuildTree converts a flat category list into
// a parent -> children hierarchy.
func BuildTree(
	categories []Category,
) []CategoryNode {
	childrenByParent := make(
		map[string][]Category,
	)

	roots := make(
		[]Category,
		0,
	)

	for _, item := range categories {
		if item.ParentID == nil {
			roots = append(
				roots,
				item,
			)

			continue
		}

		childrenByParent[*item.ParentID] = append(
			childrenByParent[*item.ParentID],
			item,
		)
	}

	var buildNode func(Category) CategoryNode

	buildNode = func(
		item Category,
	) CategoryNode {
		children := make(
			[]CategoryNode,
			0,
		)

		for _, child := range childrenByParent[item.ID] {
			children = append(
				children,
				buildNode(child),
			)
		}

		return CategoryNode{
			ID:          item.ID,
			ParentID:    item.ParentID,
			Name:        item.Name,
			Slug:        item.Slug,
			Description: item.Description,
			ImageURL:    item.ImageURL,
			IconURL:     item.IconURL,
			SortOrder:   item.SortOrder,
			Children:    children,
		}
	}

	tree := make(
		[]CategoryNode,
		0,
		len(roots),
	)

	for _, root := range roots {
		tree = append(
			tree,
			buildNode(root),
		)
	}

	return tree
}
