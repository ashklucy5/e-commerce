package catalogimport

import (
	"path"
	"strings"
)

const catalogProductImageUploadPrefix = "public/products/images/uploads/"

func (s *Service) resolveImageFiles(
	result *ParseResult,
	assets map[string]string,
) {
	if result == nil {
		return
	}

	lookup :=
		buildImageAssetLookup(
			assets,
		)

	for index := range result.Workbook.Images {

		row :=
			&result.Workbook.
				Images[index]

		imageFile :=
			strings.TrimSpace(
				row.ImageFile,
			)

		/*
			URL-backed imports already
			work and require no resolution.
		*/
		if imageFile == "" {
			continue
		}

		reference :=
			normalizeImageAssetReference(
				imageFile,
			)

		if reference == "" {
			addError(
				result,
				row.Source,
				"image_file",
				"INVALID_IMAGE_ASSET_REFERENCE",
				"image filename is invalid",
			)

			continue
		}

		storageKey :=
			lookup[reference]

		/*
			The generated ZIP format uses a
			flat images/ directory, but tolerate
			either:

			  image.webp
			  images/image.webp
		*/
		if storageKey == "" {
			storageKey =
				lookup[strings.ToLower(
					path.Base(
						reference,
					),
				)]
		}

		if storageKey == "" {
			addError(
				result,
				row.Source,
				"image_file",
				"MISSING_IMAGE_ASSET",
				"referenced image file was not uploaded with the catalog import",
			)

			continue
		}

		storageKey =
			strings.TrimSpace(
				storageKey,
			)

		/*
			Never allow a workbook to turn
			arbitrary/private storage objects
			into storefront URLs.
		*/
		if !strings.HasPrefix(
			storageKey,
			catalogProductImageUploadPrefix,
		) {
			addError(
				result,
				row.Source,
				"image_file",
				"INVALID_IMAGE_STORAGE_KEY",
				"catalog image must reference a product image upload",
			)

			continue
		}

		if s.storage == nil ||
			strings.EqualFold(
				strings.TrimSpace(
					s.storage.ProviderName(),
				),
				"disabled",
			) {

			addError(
				result,
				row.Source,
				"image_file",
				"IMAGE_STORAGE_UNAVAILABLE",
				"product image storage is unavailable",
			)

			continue
		}

		publicURL, err :=
			s.storage.PublicURL(
				storageKey,
			)

		if err != nil {
			addError(
				result,
				row.Source,
				"image_file",
				"INVALID_IMAGE_STORAGE_KEY",
				"uploaded product image could not be resolved to a public URL",
			)

			continue
		}

		/*
			From this point forward the old
			import pipeline can remain exactly
			the same.

			The action planner and Apply code
			see a normal URL-backed image.
		*/
		row.ImageURL =
			strings.TrimSpace(
				publicURL,
			)

		row.ImageFile = ""
	}
}

func buildImageAssetLookup(
	assets map[string]string,
) map[string]string {
	result :=
		make(
			map[string]string,
			len(assets)*2,
		)

	for reference, storageKey := range assets {

		normalized :=
			normalizeImageAssetReference(
				reference,
			)

		if normalized == "" {
			continue
		}

		storageKey =
			strings.TrimSpace(
				storageKey,
			)

		if storageKey == "" {
			continue
		}

		result[normalized] =
			storageKey

		base :=
			strings.ToLower(
				path.Base(
					normalized,
				),
			)

		if base != "" {
			if _, exists :=
				result[base]; !exists {

				result[base] =
					storageKey
			}
		}
	}

	return result
}

func normalizeImageAssetReference(
	value string,
) string {
	value =
		strings.TrimSpace(
			value,
		)

	value =
		strings.ReplaceAll(
			value,
			"\\",
			"/",
		)

	value =
		strings.TrimPrefix(
			value,
			"./",
		)

	value =
		strings.TrimLeft(
			value,
			"/",
		)

	if value == "" {
		return ""
	}

	cleaned :=
		path.Clean(
			value,
		)

	if cleaned == "." ||
		cleaned == ".." ||
		strings.HasPrefix(
			cleaned,
			"../",
		) ||
		strings.Contains(
			cleaned,
			"/../",
		) {

		return ""
	}

	return strings.ToLower(
		cleaned,
	)
}
