package delivery

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"

	"project.local/commerce-api/internal/platform/pagination"
)

const (
	AdminShipmentStatusNotReady    = "not_ready"
	AdminShipmentStatusNotPrepared = "not_prepared"
)

type AdminShipmentFilter struct {
	Query        string
	Status       string
	DeliveryMode string
}

type AdminShipmentListItem struct {
	ShipmentID string `json:"shipment_id,omitempty"`
	OrderID    string `json:"order_id"`

	OrderNumber   string `json:"order_number"`
	OrderStatus   string `json:"order_status"`
	PaymentStatus string `json:"payment_status"`
	PaymentMethod string `json:"payment_method"`

	CustomerName  string `json:"customer_name"`
	CustomerPhone string `json:"customer_phone"`
	CustomerEmail string `json:"customer_email,omitempty"`

	Currency    string `json:"currency"`
	TotalAmount int64  `json:"total_amount"`

	ShippingCity string `json:"shipping_city"`
	ShippingArea string `json:"shipping_area"`

	DeliveryMethod string `json:"delivery_method"`
	DeliveryMode   string `json:"delivery_mode,omitempty"`

	ProviderCode       string `json:"provider_code,omitempty"`
	ProviderShipmentID string `json:"provider_shipment_id,omitempty"`
	ProviderStatus     string `json:"provider_status,omitempty"`
	CourierName        string `json:"courier_name,omitempty"`
	CourierReference   string `json:"courier_reference,omitempty"`
	RiderReference     string `json:"rider_reference,omitempty"`
	TrackingNumber     string `json:"tracking_number,omitempty"`
	TrackingURL        string `json:"tracking_url,omitempty"`

	ShipmentStatus string `json:"shipment_status"`

	WarehouseID   string `json:"warehouse_id,omitempty"`
	WarehouseCode string `json:"warehouse_code,omitempty"`
	WarehouseName string `json:"warehouse_name,omitempty"`

	LatestEventCode    string     `json:"latest_event_code,omitempty"`
	LatestEventStatus  string     `json:"latest_event_status,omitempty"`
	LatestEventMessage string     `json:"latest_event_message,omitempty"`
	LatestEventAt      *time.Time `json:"latest_event_at,omitempty"`

	ShippedAt              *time.Time `json:"shipped_at,omitempty"`
	AwaitingConfirmationAt *time.Time `json:"awaiting_confirmation_at,omitempty"`
	DeliveredAt            *time.Time `json:"delivered_at,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type AdminShipmentListResult struct {
	Items []AdminShipmentListItem
	Meta  pagination.Meta
}

type AdminShipmentSummary struct {
	NeedsShipment        int64 `json:"needs_shipment"`
	Pending              int64 `json:"pending"`
	Shipped              int64 `json:"shipped"`
	AwaitingConfirmation int64 `json:"awaiting_confirmation"`
	Delivered            int64 `json:"delivered"`
	Cancelled            int64 `json:"cancelled"`
}

func normalizeAdminShipmentFilter(
	filter AdminShipmentFilter,
) (AdminShipmentFilter, error) {
	filter.Query = strings.TrimSpace(filter.Query)
	filter.Status = strings.ToLower(strings.TrimSpace(filter.Status))
	filter.DeliveryMode = strings.ToLower(strings.TrimSpace(filter.DeliveryMode))

	if utf8.RuneCountInString(filter.Query) > 100 {
		return AdminShipmentFilter{}, ErrInvalidInput
	}

	switch filter.Status {
	case "",
		AdminShipmentStatusNotReady,
		AdminShipmentStatusNotPrepared,
		ShipmentStatusPending,
		ShipmentStatusShipped,
		ShipmentStatusAwaitingConfirmation,
		ShipmentStatusDelivered,
		ShipmentStatusCancelled:
	default:
		return AdminShipmentFilter{}, ErrInvalidInput
	}

	if filter.DeliveryMode != "" && !validDeliveryMode(filter.DeliveryMode) {
		return AdminShipmentFilter{}, ErrInvalidInput
	}

	return filter, nil
}

func (s *Service) ListAdminShipments(
	ctx context.Context,
	params pagination.Params,
	filter AdminShipmentFilter,
) (AdminShipmentListResult, error) {
	filter, err := normalizeAdminShipmentFilter(filter)
	if err != nil {
		return AdminShipmentListResult{}, err
	}

	return s.repository.ListAdminShipments(ctx, params, filter)
}

func (s *Service) AdminShipmentSummary(
	ctx context.Context,
) (AdminShipmentSummary, error) {
	return s.repository.AdminShipmentSummary(ctx)
}

func (r *Repository) ListAdminShipments(
	ctx context.Context,
	params pagination.Params,
	filter AdminShipmentFilter,
) (AdminShipmentListResult, error) {
	rows, err := r.db.Query(
		ctx,
		`
			SELECT
				COALESCE(s.id::text, ''),
				o.id::text,
				o.order_number,
				o.status,
				o.payment_status,
				o.payment_method,
				o.customer_name,
				o.customer_phone,
				COALESCE(o.customer_email, ''),
				o.currency,
				o.total_amount,
				o.shipping_city,
				o.shipping_area,
				o.delivery_method,
				COALESCE(s.delivery_mode, ''),
				COALESCE(s.provider_code, ''),
				COALESCE(s.provider_shipment_id, ''),
				COALESCE(s.provider_status, ''),
				COALESCE(s.courier_name, ''),
				COALESCE(s.courier_reference, ''),
				COALESCE(s.rider_reference, ''),
				COALESCE(s.tracking_number, ''),
				COALESCE(s.tracking_url, ''),
				CASE
					WHEN s.id IS NULL AND o.status = 'processing' THEN 'not_prepared'
					WHEN s.id IS NULL THEN 'not_ready'
					ELSE s.status
				END,
				COALESCE(s.origin_warehouse_id::text, ''),
				COALESCE(w.code, ''),
				COALESCE(w.name, ''),
				COALESCE(latest.event_code, ''),
				COALESCE(latest.status, ''),
				COALESCE(NULLIF(latest.public_message, ''), latest.message, ''),
				latest.occurred_at,
				s.shipped_at,
				s.awaiting_confirmation_at,
				s.delivered_at,
				COALESCE(s.created_at, o.created_at),
				GREATEST(o.updated_at, COALESCE(s.updated_at, o.updated_at)),
				COUNT(*) OVER()::bigint
			FROM orders o
			LEFT JOIN shipments s
				ON s.order_id = o.id
			LEFT JOIN warehouses w
				ON w.id = s.origin_warehouse_id
			LEFT JOIN LATERAL (
				SELECT
					e.event_code,
					COALESCE(e.status, '') AS status,
					COALESCE(e.public_message, '') AS public_message,
					COALESCE(e.message, '') AS message,
					e.occurred_at
				FROM delivery_tracking_events e
				WHERE e.order_id = o.id
				ORDER BY e.occurred_at DESC, e.created_at DESC, e.id DESC
				LIMIT 1
			) latest ON true
			WHERE
				(
					$1 <> ''
					OR s.id IS NOT NULL
					OR o.status = 'processing'
				)
				AND (
					$2 = ''
					OR CASE
						WHEN s.id IS NULL AND o.status = 'processing' THEN 'not_prepared'
						WHEN s.id IS NULL THEN 'not_ready'
						ELSE s.status
					END = $2
				)
				AND ($3 = '' OR s.delivery_mode = $3)
				AND (
					$1 = ''
					OR o.order_number = upper($1)
					OR o.customer_phone = $1
					OR COALESCE(s.tracking_number, '') = $1
					OR COALESCE(s.provider_shipment_id, '') = $1
					OR COALESCE(s.courier_reference, '') = $1
					OR COALESCE(s.rider_reference, '') = $1
					OR COALESCE(s.id::text, '') = $1
					OR lower(o.customer_name) LIKE '%' || lower($1) || '%'
					OR lower(COALESCE(s.courier_name, '')) LIKE '%' || lower($1) || '%'
					OR lower(COALESCE(s.provider_code, '')) = lower($1)
				)
			ORDER BY
				CASE WHEN s.id IS NULL AND o.status = 'processing' THEN 0 ELSE 1 END,
				COALESCE(s.created_at, o.created_at) DESC,
				o.id DESC
			LIMIT $4 OFFSET $5
		`,
		filter.Query,
		filter.Status,
		filter.DeliveryMode,
		params.Limit,
		params.Offset(),
	)
	if err != nil {
		return AdminShipmentListResult{}, fmt.Errorf("list admin shipments: %w", err)
	}
	defer rows.Close()

	items := make([]AdminShipmentListItem, 0, params.Limit)
	var total int64

	for rows.Next() {
		var item AdminShipmentListItem
		var latestEventAt pgtype.Timestamptz
		var shippedAt pgtype.Timestamptz
		var awaitingConfirmationAt pgtype.Timestamptz
		var deliveredAt pgtype.Timestamptz

		if err := rows.Scan(
			&item.ShipmentID,
			&item.OrderID,
			&item.OrderNumber,
			&item.OrderStatus,
			&item.PaymentStatus,
			&item.PaymentMethod,
			&item.CustomerName,
			&item.CustomerPhone,
			&item.CustomerEmail,
			&item.Currency,
			&item.TotalAmount,
			&item.ShippingCity,
			&item.ShippingArea,
			&item.DeliveryMethod,
			&item.DeliveryMode,
			&item.ProviderCode,
			&item.ProviderShipmentID,
			&item.ProviderStatus,
			&item.CourierName,
			&item.CourierReference,
			&item.RiderReference,
			&item.TrackingNumber,
			&item.TrackingURL,
			&item.ShipmentStatus,
			&item.WarehouseID,
			&item.WarehouseCode,
			&item.WarehouseName,
			&item.LatestEventCode,
			&item.LatestEventStatus,
			&item.LatestEventMessage,
			&latestEventAt,
			&shippedAt,
			&awaitingConfirmationAt,
			&deliveredAt,
			&item.CreatedAt,
			&item.UpdatedAt,
			&total,
		); err != nil {
			return AdminShipmentListResult{}, fmt.Errorf("scan admin shipment: %w", err)
		}

		item.LatestEventAt = adminShipmentTimePointer(latestEventAt)
		item.ShippedAt = adminShipmentTimePointer(shippedAt)
		item.AwaitingConfirmationAt = adminShipmentTimePointer(awaitingConfirmationAt)
		item.DeliveredAt = adminShipmentTimePointer(deliveredAt)

		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return AdminShipmentListResult{}, fmt.Errorf("iterate admin shipments: %w", err)
	}

	return AdminShipmentListResult{
		Items: items,
		Meta:  pagination.NewMeta(params, total),
	}, nil
}

func (r *Repository) AdminShipmentSummary(
	ctx context.Context,
) (AdminShipmentSummary, error) {
	var result AdminShipmentSummary

	err := r.db.QueryRow(
		ctx,
		`
			SELECT
				COUNT(*) FILTER (
					WHERE o.status = 'processing' AND s.id IS NULL
				)::bigint,
				COUNT(*) FILTER (WHERE s.status = 'pending')::bigint,
				COUNT(*) FILTER (WHERE s.status = 'shipped')::bigint,
				COUNT(*) FILTER (WHERE s.status = 'awaiting_confirmation')::bigint,
				COUNT(*) FILTER (WHERE s.status = 'delivered')::bigint,
				COUNT(*) FILTER (WHERE s.status = 'cancelled')::bigint
			FROM orders o
			LEFT JOIN shipments s ON s.order_id = o.id
			WHERE s.id IS NOT NULL OR o.status = 'processing'
		`,
	).Scan(
		&result.NeedsShipment,
		&result.Pending,
		&result.Shipped,
		&result.AwaitingConfirmation,
		&result.Delivered,
		&result.Cancelled,
	)
	if err != nil {
		return AdminShipmentSummary{}, fmt.Errorf("load admin shipment summary: %w", err)
	}

	return result, nil
}

func adminShipmentTimePointer(value pgtype.Timestamptz) *time.Time {
	if !value.Valid {
		return nil
	}

	result := value.Time.UTC()
	return &result
}

func (h *AdminHandler) ListShipments(c *gin.Context) {
	params, err := pagination.Parse(c.Query("page"), c.Query("limit"))
	if err != nil {
		writeDeliveryError(c, ErrInvalidInput)
		return
	}

	result, err := h.service.ListAdminShipments(
		c.Request.Context(),
		params,
		AdminShipmentFilter{
			Query:        c.Query("q"),
			Status:       c.Query("status"),
			DeliveryMode: c.Query("delivery_mode"),
		},
	)
	if err != nil {
		writeDeliveryError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": result.Items,
		"meta": result.Meta,
	})
}

func (h *AdminHandler) ShipmentSummary(c *gin.Context) {
	result, err := h.service.AdminShipmentSummary(c.Request.Context())
	if err != nil {
		writeDeliveryError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": result})
}
