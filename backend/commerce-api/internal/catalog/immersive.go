package catalog

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

var (
	ErrInvalidImmersiveVariantID = errors.New(
		"invalid immersive variant ID",
	)

	ErrImmersiveVariantNotFound = errors.New(
		"immersive variant not found",
	)
)

type ProductImmersiveMedia struct {
	Spin360 *ProductSpin360Media `json:"spin_360,omitempty"`

	Model3D *ProductModel3DMedia `json:"model_3d,omitempty"`
}

type ProductSpin360Media struct {
	Available bool `json:"available"`

	FrameCount int `json:"frame_count,omitempty"`

	PosterURL *string `json:"poster_url,omitempty"`

	Variants []ProductSpin360Variant `json:"variants,omitempty"`
}

type ProductSpin360Variant struct {
	VariantID string `json:"variant_id"`

	FrameCount int `json:"frame_count"`

	PosterURL string `json:"poster_url"`
}

type ProductModel3DMedia struct {
	Available bool `json:"available"`

	GLBURL *string `json:"glb_url,omitempty"`

	PosterURL *string `json:"poster_url,omitempty"`

	Variants []ProductModel3DVariant `json:"variants,omitempty"`
}

type ProductModel3DVariant struct {
	VariantID string `json:"variant_id"`

	GLBURL string `json:"glb_url"`

	PosterURL *string `json:"poster_url,omitempty"`
}

type Product360Frame struct {
	Frame int `json:"frame"`

	URL string `json:"url"`
}

type Product360FrameSet struct {
	VariantID *string `json:"variant_id,omitempty"`

	Frames []Product360Frame `json:"frames"`
}

type immersiveRepository interface {
	FindProductImmersiveMedia(
		ctx context.Context,
		productID string,
	) (*ProductImmersiveMedia, error)

	FindActiveProduct360FramesBySlug(
		ctx context.Context,
		slug string,
		variantID string,
	) (Product360FrameSet, error)
}

func (r *Repository) FindProductImmersiveMedia(
	ctx context.Context,
	productID string,
) (*ProductImmersiveMedia, error) {
	result :=
		&ProductImmersiveMedia{}

	spinRows, err :=
		r.db.Query(
			ctx,
			`
				SELECT
					variant_id::text,
					COUNT(*)::int,
					(array_agg(
						url
						ORDER BY frame_index ASC
					))[1]
				FROM product_360_frames
				WHERE product_id = $1
				GROUP BY variant_id
				ORDER BY variant_id NULLS FIRST
			`,
			productID,
		)
	if err != nil {
		return nil,
			fmt.Errorf(
				"query product 360 summaries: %w",
				err,
			)
	}

	spin :=
		&ProductSpin360Media{}

	for spinRows.Next() {
		var variantID pgtype.Text
		var frameCount int
		var posterURL string

		if err :=
			spinRows.Scan(
				&variantID,
				&frameCount,
				&posterURL,
			); err != nil {

			spinRows.Close()

			return nil,
				fmt.Errorf(
					"scan product 360 summary: %w",
					err,
				)
		}

		if variantID.Valid {
			spin.Variants =
				append(
					spin.Variants,
					ProductSpin360Variant{
						VariantID: variantID.String,

						FrameCount: frameCount,

						PosterURL: posterURL,
					},
				)

			continue
		}

		poster :=
			posterURL

		spin.FrameCount =
			frameCount

		spin.PosterURL =
			&poster
	}

	if err :=
		spinRows.Err(); err != nil {

		spinRows.Close()

		return nil,
			fmt.Errorf(
				"iterate product 360 summaries: %w",
				err,
			)
	}

	spinRows.Close()

	spin.Available =
		spin.FrameCount > 0 ||
			len(
				spin.Variants,
			) > 0

	if spin.Available {
		result.Spin360 =
			spin
	}

	modelRows, err :=
		r.db.Query(
			ctx,
			`
				SELECT
					variant_id::text,
					url,
					poster_url
				FROM product_3d_models
				WHERE product_id = $1
				ORDER BY variant_id NULLS FIRST
			`,
			productID,
		)
	if err != nil {
		return nil,
			fmt.Errorf(
				"query product 3D models: %w",
				err,
			)
	}

	model :=
		&ProductModel3DMedia{}

	for modelRows.Next() {
		var variantID pgtype.Text
		var modelURL string
		var posterURL pgtype.Text

		if err :=
			modelRows.Scan(
				&variantID,
				&modelURL,
				&posterURL,
			); err != nil {

			modelRows.Close()

			return nil,
				fmt.Errorf(
					"scan product 3D model: %w",
					err,
				)
		}

		if variantID.Valid {
			var poster *string

			if posterURL.Valid {
				value :=
					posterURL.String

				poster =
					&value
			}

			model.Variants =
				append(
					model.Variants,
					ProductModel3DVariant{
						VariantID: variantID.String,

						GLBURL: modelURL,

						PosterURL: poster,
					},
				)

			continue
		}

		urlValue :=
			modelURL

		model.GLBURL =
			&urlValue

		if posterURL.Valid {
			value :=
				posterURL.String

			model.PosterURL =
				&value
		}
	}

	if err :=
		modelRows.Err(); err != nil {

		modelRows.Close()

		return nil,
			fmt.Errorf(
				"iterate product 3D models: %w",
				err,
			)
	}

	modelRows.Close()

	model.Available =
		model.GLBURL != nil ||
			len(
				model.Variants,
			) > 0

	if model.Available {
		result.Model3D =
			model
	}

	if result.Spin360 == nil &&
		result.Model3D == nil {

		return nil,
			nil
	}

	return result,
		nil
}

func (r *Repository) FindActiveProduct360FramesBySlug(
	ctx context.Context,
	slug string,
	variantID string,
) (Product360FrameSet, error) {
	var productID string

	err :=
		r.db.QueryRow(
			ctx,
			`
				SELECT
					p.id::text
				FROM products p

				JOIN categories c
					ON c.id = p.category_id

				WHERE
					p.slug = $1
					AND p.status = 'active'
					AND c.is_active = true

				LIMIT 1
			`,
			slug,
		).Scan(
			&productID,
		)

	if err != nil {
		return Product360FrameSet{},
			err
	}

	if variantID != "" {
		var belongs bool

		err =
			r.db.QueryRow(
				ctx,
				`
					SELECT EXISTS (
						SELECT 1
						FROM product_variants
						WHERE
							id = $1::uuid
							AND product_id = $2::uuid
							AND is_active = true
					)
				`,
				variantID,
				productID,
			).Scan(
				&belongs,
			)

		if err != nil {
			return Product360FrameSet{},
				fmt.Errorf(
					"verify product variant: %w",
					err,
				)
		}

		if !belongs {
			return Product360FrameSet{},
				ErrImmersiveVariantNotFound
		}

		frames, err :=
			r.list360Frames(
				ctx,
				productID,
				variantID,
			)

		if err != nil {
			return Product360FrameSet{},
				err
		}

		if len(frames) > 0 {
			value :=
				variantID

			return Product360FrameSet{
				VariantID: &value,

				Frames: frames,
			}, nil
		}
	}

	frames, err :=
		r.list360Frames(
			ctx,
			productID,
			"",
		)

	if err != nil {
		return Product360FrameSet{},
			err
	}

	return Product360FrameSet{
		Frames: frames,
	}, nil
}

func (r *Repository) list360Frames(
	ctx context.Context,
	productID string,
	variantID string,
) ([]Product360Frame, error) {
	var rows pgx.Rows
	var err error

	if variantID == "" {
		rows, err =
			r.db.Query(
				ctx,
				`
					SELECT
						frame_index,
						url
					FROM product_360_frames
					WHERE
						product_id = $1::uuid
						AND variant_id IS NULL
					ORDER BY
						frame_index ASC,
						id ASC
				`,
				productID,
			)
	} else {
		rows, err =
			r.db.Query(
				ctx,
				`
					SELECT
						frame_index,
						url
					FROM product_360_frames
					WHERE
						product_id = $1::uuid
						AND variant_id = $2::uuid
					ORDER BY
						frame_index ASC,
						id ASC
				`,
				productID,
				variantID,
			)
	}

	if err != nil {
		return nil,
			fmt.Errorf(
				"query product 360 frames: %w",
				err,
			)
	}

	defer rows.Close()

	frames :=
		make(
			[]Product360Frame,
			0,
		)

	for rows.Next() {
		var frame Product360Frame

		if err :=
			rows.Scan(
				&frame.Frame,
				&frame.URL,
			); err != nil {

			return nil,
				fmt.Errorf(
					"scan product 360 frame: %w",
					err,
				)
		}

		frames =
			append(
				frames,
				frame,
			)
	}

	if err :=
		rows.Err(); err != nil {

		return nil,
			fmt.Errorf(
				"iterate product 360 frames: %w",
				err,
			)
	}

	return frames,
		nil
}

func (s *Service) attachImmersiveMedia(
	ctx context.Context,
	product *ProductDetail,
) error {
	repository, ok :=
		s.repository.(immersiveRepository)

	if !ok {
		return nil
	}

	media, err :=
		repository.
			FindProductImmersiveMedia(
				ctx,
				product.ID,
			)

	if err != nil {
		return err
	}

	product.ImmersiveMedia =
		media

	return nil
}

func (s *Service) GetActiveProduct360FramesBySlug(
	ctx context.Context,
	slug string,
	variantID string,
) (Product360FrameSet, error) {
	slug =
		strings.TrimSpace(
			slug,
		)

	variantID =
		strings.TrimSpace(
			variantID,
		)

	if slug == "" {
		return Product360FrameSet{},
			ErrInvalidProductSlug
	}

	if variantID != "" {
		var parsed pgtype.UUID

		if err :=
			parsed.Scan(
				variantID,
			); err != nil ||
			!parsed.Valid {

			return Product360FrameSet{},
				ErrInvalidImmersiveVariantID
		}
	}

	repository, ok :=
		s.repository.(immersiveRepository)

	if !ok {
		return Product360FrameSet{},
			fmt.Errorf(
				"immersive catalog repository is unavailable",
			)
	}

	result, err :=
		repository.
			FindActiveProduct360FramesBySlug(
				ctx,
				slug,
				variantID,
			)

	if err != nil {
		if errors.Is(
			err,
			pgx.ErrNoRows,
		) {
			return Product360FrameSet{},
				ErrProductNotFound
		}

		return Product360FrameSet{},
			err
	}

	if result.Frames == nil {
		result.Frames =
			make(
				[]Product360Frame,
				0,
			)
	}

	return result,
		nil
}

func (h *Handler) GetProduct360Frames(
	c *gin.Context,
) {
	result, err :=
		h.service.
			GetActiveProduct360FramesBySlug(
				c.Request.Context(),

				c.Param(
					"slug",
				),

				c.Query(
					"variant_id",
				),
			)

	if err != nil {
		switch {

		case errors.Is(
			err,
			ErrInvalidProductSlug,
		),
			errors.Is(
				err,
				ErrInvalidImmersiveVariantID,
			):

			c.JSON(
				http.StatusBadRequest,
				gin.H{
					"error": gin.H{
						"code": "INVALID_IMMERSIVE_MEDIA_REQUEST",

						"message": "Invalid immersive media request",
					},
				},
			)

		case errors.Is(
			err,
			ErrProductNotFound,
		):

			c.JSON(
				http.StatusNotFound,
				gin.H{
					"error": gin.H{
						"code": "PRODUCT_NOT_FOUND",

						"message": "Product not found",
					},
				},
			)

		case errors.Is(
			err,
			ErrImmersiveVariantNotFound,
		):

			c.JSON(
				http.StatusNotFound,
				gin.H{
					"error": gin.H{
						"code": "PRODUCT_VARIANT_NOT_FOUND",

						"message": "Product variant not found",
					},
				},
			)

		default:
			c.JSON(
				http.StatusInternalServerError,
				gin.H{
					"error": gin.H{
						"code": "INTERNAL_ERROR",

						"message": "Unable to load product 360 media",
					},
				},
			)
		}

		return
	}

	c.JSON(
		http.StatusOK,
		gin.H{
			"data": result,
		},
	)
}
