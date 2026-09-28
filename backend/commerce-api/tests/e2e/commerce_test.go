package blackbox_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"
)

const defaultBaseURL = "http://127.0.0.1:8081"

type apiClient struct {
	baseURL string

	client *http.Client
}

type apiResponse struct {
	Status int

	Body []byte

	Header http.Header
}

func TestBackendBlackBox(
	t *testing.T,
) {
	if os.Getenv(
		"RUN_BLACKBOX_E2E",
	) != "1" {

		t.Skip(
			"set RUN_BLACKBOX_E2E=1 to run against a live commerce API",
		)
	}

	baseURL :=
		strings.TrimRight(
			strings.TrimSpace(
				os.Getenv(
					"E2E_BASE_URL",
				),
			),
			"/",
		)

	if baseURL == "" {
		baseURL =
			defaultBaseURL
	}

	api :=
		apiClient{
			baseURL: baseURL,

			client: &http.Client{
				Timeout: 20 *
					time.Second,
			},
		}

	t.Run(
		"health ready",
		func(
			t *testing.T,
		) {
			response :=
				api.do(
					t,
					http.MethodGet,
					"/health/ready",
					nil,
					nil,
				)

			assertStatus(
				t,
				response,
				http.StatusOK,
			)
		},
	)

	var productSlug string
	var productName string
	var variantID string
	var quantity int

	t.Run(
		"anonymous storefront browsing",
		func(
			t *testing.T,
		) {
			categories :=
				api.do(
					t,
					http.MethodGet,
					"/api/v1/categories",
					nil,
					nil,
				)

			assertStatus(
				t,
				categories,
				http.StatusOK,
			)

			products :=
				api.do(
					t,
					http.MethodGet,
					"/api/v1/products?page=1&limit=20&in_stock=true&sort=newest",
					nil,
					nil,
				)

			assertStatus(
				t,
				products,
				http.StatusOK,
			)

			var productEnvelope struct {
				Data []struct {
					Slug string `json:"slug"`

					Name string `json:"name"`

					InStock bool `json:"in_stock"`
				} `json:"data"`
			}

			decodeJSON(
				t,
				products.Body,
				&productEnvelope,
			)

			if len(
				productEnvelope.Data,
			) == 0 {

				t.Fatal(
					"black-box E2E requires at least one active in-stock product",
				)
			}

			for _, item := range productEnvelope.Data {

				if item.InStock {
					productSlug =
						item.Slug

					productName =
						item.Name

					break
				}
			}

			if productSlug == "" {
				t.Fatal(
					"black-box E2E could not find an in-stock product",
				)
			}

			detail :=
				api.do(
					t,
					http.MethodGet,
					"/api/v1/products/"+
						url.PathEscape(
							productSlug,
						),
					nil,
					nil,
				)

			assertStatus(
				t,
				detail,
				http.StatusOK,
			)

			var detailEnvelope struct {
				Data struct {
					Variants []struct {
						ID string `json:"id"`

						MinimumOrderQuantity int `json:"minimum_order_quantity"`

						AvailableQuantity int `json:"available_quantity"`

						InStock bool `json:"in_stock"`
					} `json:"variants"`
				} `json:"data"`
			}

			decodeJSON(
				t,
				detail.Body,
				&detailEnvelope,
			)

			for _, variant := range detailEnvelope.Data.Variants {

				if variant.InStock &&
					variant.MinimumOrderQuantity >
						0 &&
					variant.AvailableQuantity >=
						variant.MinimumOrderQuantity {

					variantID =
						variant.ID

					quantity =
						variant.MinimumOrderQuantity

					break
				}
			}

			if variantID == "" {
				t.Fatal(
					"black-box E2E requires an active variant with stock >= MOQ",
				)
			}

			query :=
				url.QueryEscape(
					productName,
				)

			searchResponse :=
				api.do(
					t,
					http.MethodGet,
					"/api/v1/search?q="+
						query+
						"&page=1&limit=10",
					nil,
					nil,
				)

			assertStatus(
				t,
				searchResponse,
				http.StatusOK,
			)

			browseSearch :=
				api.do(
					t,
					http.MethodGet,
					"/api/v1/search/products?q="+
						query+
						"&page=1&limit=10&in_stock=true&sort=price_asc",
					nil,
					nil,
				)

			assertStatus(
				t,
				browseSearch,
				http.StatusOK,
			)

			var searchEnvelope struct {
				Data []any `json:"data"`

				Facets map[string]any `json:"facets"`
			}

			decodeJSON(
				t,
				browseSearch.Body,
				&searchEnvelope,
			)

			if searchEnvelope.Facets == nil {
				t.Fatal(
					"filterable search response did not include facets",
				)
			}

			promotions :=
				api.do(
					t,
					http.MethodGet,
					"/api/v1/promotions/active?currency=BDT",
					nil,
					nil,
				)

			assertStatus(
				t,
				promotions,
				http.StatusOK,
			)
		},
	)

	t.Run(
		"account routes require login",
		func(
			t *testing.T,
		) {
			orders :=
				api.do(
					t,
					http.MethodGet,
					"/api/v1/orders",
					nil,
					nil,
				)

			assertStatus(
				t,
				orders,
				http.StatusUnauthorized,
			)

			returns :=
				api.do(
					t,
					http.MethodGet,
					"/api/v1/returns",
					nil,
					nil,
				)

			assertStatus(
				t,
				returns,
				http.StatusUnauthorized,
			)
		},
	)

	var customerToken string

	t.Run(
		"customer register login and account history",
		func(
			t *testing.T,
		) {
			unique :=
				time.Now().
					UnixNano()

			phone :=
				fmt.Sprintf(
					"+88017%08d",
					unique%
						100000000,
				)

			email :=
				fmt.Sprintf(
					"commerce-e2e-%d@example.test",
					unique,
				)

			password :=
				"E2E-Test-2026!"

			register :=
				api.do(
					t,
					http.MethodPost,
					"/api/v1/auth/register",
					map[string]any{
						"phone": phone,

						"email": email,

						"password": password,

						"full_name": "Commerce E2E",
					},
					nil,
				)

			assertStatus(
				t,
				register,
				http.StatusCreated,
			)

			login :=
				api.do(
					t,
					http.MethodPost,
					"/api/v1/auth/login",
					map[string]any{
						"phone": phone,

						"password": password,
					},
					nil,
				)

			assertStatus(
				t,
				login,
				http.StatusOK,
			)

			var authEnvelope struct {
				Data struct {
					Tokens struct {
						AccessToken string `json:"access_token"`
					} `json:"tokens"`
				} `json:"data"`
			}

			decodeJSON(
				t,
				login.Body,
				&authEnvelope,
			)

			customerToken =
				authEnvelope.
					Data.
					Tokens.
					AccessToken

			if customerToken == "" {
				t.Fatal(
					"login response did not include access token",
				)
			}

			headers :=
				map[string]string{
					"Authorization": "Bearer " +
						customerToken,
				}

			orders :=
				api.do(
					t,
					http.MethodGet,
					"/api/v1/orders?page=1&limit=20",
					nil,
					headers,
				)

			assertStatus(
				t,
				orders,
				http.StatusOK,
			)

			returns :=
				api.do(
					t,
					http.MethodGet,
					"/api/v1/returns?page=1&limit=20",
					nil,
					headers,
				)

			assertStatus(
				t,
				returns,
				http.StatusOK,
			)
		},
	)

	var guestCheckoutKey string
	var guestOrderID string

	t.Run(
		"guest cart checkout and order",
		func(
			t *testing.T,
		) {
			cartResponse :=
				api.do(
					t,
					http.MethodPost,
					"/api/v1/carts",
					map[string]any{},
					nil,
				)

			assertStatus(
				t,
				cartResponse,
				http.StatusCreated,
			)

			var cartEnvelope struct {
				Data struct {
					CartKey string `json:"cart_key"`
				} `json:"data"`
			}

			decodeJSON(
				t,
				cartResponse.Body,
				&cartEnvelope,
			)

			if cartEnvelope.Data.CartKey == "" {
				t.Fatal(
					"cart response did not include cart_key",
				)
			}

			addItem :=
				api.do(
					t,
					http.MethodPost,
					"/api/v1/carts/"+
						url.PathEscape(
							cartEnvelope.
								Data.
								CartKey,
						)+
						"/items",
					map[string]any{
						"variant_id": variantID,

						"quantity": quantity,
					},
					nil,
				)

			assertStatus(
				t,
				addItem,
				http.StatusOK,
			)

			checkoutResponse :=
				api.do(
					t,
					http.MethodPost,
					"/api/v1/checkouts",
					map[string]any{
						"cart_key": cartEnvelope.
							Data.
							CartKey,
					},
					nil,
				)

			assertStatus(
				t,
				checkoutResponse,
				http.StatusOK,
			)

			var checkoutEnvelope struct {
				Data struct {
					CheckoutKey string `json:"checkout_key"`
				} `json:"data"`
			}

			decodeJSON(
				t,
				checkoutResponse.Body,
				&checkoutEnvelope,
			)

			guestCheckoutKey =
				checkoutEnvelope.
					Data.
					CheckoutKey

			if guestCheckoutKey == "" {
				t.Fatal(
					"checkout response did not include checkout_key",
				)
			}

			deliveryResponse :=
				api.do(
					t,
					http.MethodGet,
					"/api/v1/checkouts/"+
						url.PathEscape(
							guestCheckoutKey,
						)+
						"/delivery-methods",
					nil,
					nil,
				)

			assertStatus(
				t,
				deliveryResponse,
				http.StatusOK,
			)

			var deliveryEnvelope struct {
				Data []struct {
					Code string `json:"code"`
				} `json:"data"`
			}

			decodeJSON(
				t,
				deliveryResponse.Body,
				&deliveryEnvelope,
			)

			if len(
				deliveryEnvelope.Data,
			) == 0 {

				t.Fatal(
					"checkout has no enabled delivery method",
				)
			}

			paymentResponse :=
				api.do(
					t,
					http.MethodGet,
					"/api/v1/checkout-options/payment-methods?checkout_key="+
						url.QueryEscape(
							guestCheckoutKey,
						),
					nil,
					nil,
				)

			assertStatus(
				t,
				paymentResponse,
				http.StatusOK,
			)

			var paymentEnvelope struct {
				Data []struct {
					Code string `json:"code"`
				} `json:"data"`
			}

			decodeJSON(
				t,
				paymentResponse.Body,
				&paymentEnvelope,
			)

			paymentMethod :=
				""

			for _, option := range paymentEnvelope.Data {

				if option.Code ==
					"cod" {

					paymentMethod =
						option.Code

					break
				}
			}

			if paymentMethod == "" {
				t.Fatal(
					"black-box E2E requires COD enabled to avoid leaving pending payment reservations",
				)
			}

			updateResponse :=
				api.do(
					t,
					http.MethodPatch,
					"/api/v1/checkouts/"+
						url.PathEscape(
							guestCheckoutKey,
						),
					map[string]any{
						"customer_name": "Guest E2E",

						"customer_phone": "+8801700000000",

						"customer_email": "guest-e2e@example.test",

						"shipping_address_line1": "E2E Test Address",

						"shipping_city": "Dhaka",

						"shipping_area": "Dhanmondi",

						"delivery_method": deliveryEnvelope.
							Data[0].
							Code,

						"payment_method": paymentMethod,
					},
					nil,
				)

			assertStatus(
				t,
				updateResponse,
				http.StatusOK,
			)

			placeOrder :=
				api.do(
					t,
					http.MethodPost,
					"/api/v1/checkouts/"+
						url.PathEscape(
							guestCheckoutKey,
						)+
						"/place-order",
					map[string]any{},
					nil,
				)

			assertOneOfStatus(
				t,
				placeOrder,
				http.StatusCreated,
				http.StatusOK,
			)

			var orderEnvelope struct {
				Data struct {
					ID string `json:"id"`
				} `json:"data"`
			}

			decodeJSON(
				t,
				placeOrder.Body,
				&orderEnvelope,
			)

			guestOrderID =
				orderEnvelope.
					Data.
					ID

			if guestOrderID == "" {
				t.Fatal(
					"place-order response did not include order id",
				)
			}
		},
	)

	t.Run(
		"guest order and tracking ownership",
		func(
			t *testing.T,
		) {
			withoutKey :=
				api.do(
					t,
					http.MethodGet,
					"/api/v1/orders/"+
						url.PathEscape(
							guestOrderID,
						),
					nil,
					nil,
				)

			assertStatus(
				t,
				withoutKey,
				http.StatusNotFound,
			)

			wrongHeaders :=
				map[string]string{
					"X-Checkout-Key": "chk_wrong_blackbox_key",
				}

			wrongOrder :=
				api.do(
					t,
					http.MethodGet,
					"/api/v1/orders/"+
						url.PathEscape(
							guestOrderID,
						),
					nil,
					wrongHeaders,
				)

			assertStatus(
				t,
				wrongOrder,
				http.StatusNotFound,
			)

			guestHeaders :=
				map[string]string{
					"X-Checkout-Key": guestCheckoutKey,
				}

			correctOrder :=
				api.do(
					t,
					http.MethodGet,
					"/api/v1/orders/"+
						url.PathEscape(
							guestOrderID,
						),
					nil,
					guestHeaders,
				)

			assertStatus(
				t,
				correctOrder,
				http.StatusOK,
			)

			wrongTracking :=
				api.do(
					t,
					http.MethodGet,
					"/api/v1/orders/"+
						url.PathEscape(
							guestOrderID,
						)+
						"/tracking",
					nil,
					wrongHeaders,
				)

			assertStatus(
				t,
				wrongTracking,
				http.StatusNotFound,
			)

			if errorCode(
				wrongTracking.Body,
			) != "ORDER_NOT_FOUND" {

				t.Fatalf(
					"wrong guest tracking key must be rejected as ORDER_NOT_FOUND; body=%s",
					string(
						wrongTracking.Body,
					),
				)
			}

			correctTracking :=
				api.do(
					t,
					http.MethodGet,
					"/api/v1/orders/"+
						url.PathEscape(
							guestOrderID,
						)+
						"/tracking",
					nil,
					guestHeaders,
				)

			switch correctTracking.Status {
			case http.StatusOK:
				// Shipment exists and ownership succeeded.

			case http.StatusNotFound:

				if errorCode(
					correctTracking.Body,
				) != "SHIPMENT_NOT_FOUND" {

					t.Fatalf(
						"correct guest key did not pass ownership boundary; body=%s",
						string(
							correctTracking.Body,
						),
					)
				}

			default:

				t.Fatalf(
					"correct guest tracking key returned HTTP %d: %s",
					correctTracking.Status,
					string(
						correctTracking.Body,
					),
				)
			}
		},
	)

	t.Run(
		"admin boundary rejects anonymous request",
		func(
			t *testing.T,
		) {
			response :=
				api.do(
					t,
					http.MethodGet,
					"/api/v1/admin/dashboard",
					nil,
					nil,
				)

			assertOneOfStatus(
				t,
				response,
				http.StatusUnauthorized,
				http.StatusForbidden,
			)
		},
	)

	if customerToken != "" {
		t.Run(
			"customer logout",
			func(
				t *testing.T,
			) {
				response :=
					api.do(
						t,
						http.MethodPost,
						"/api/v1/auth/logout",
						map[string]any{},
						map[string]string{
							"Authorization": "Bearer " +
								customerToken,
						},
					)

				assertStatus(
					t,
					response,
					http.StatusNoContent,
				)
			},
		)
	}
}

func (a apiClient) do(
	t *testing.T,
	method string,
	path string,
	body any,
	headers map[string]string,
) apiResponse {
	t.Helper()

	var reader io.Reader

	if body != nil {
		encoded, err :=
			json.Marshal(
				body,
			)
		if err != nil {
			t.Fatalf(
				"encode request %s %s: %v",
				method,
				path,
				err,
			)
		}

		reader =
			bytes.NewReader(
				encoded,
			)
	}

	request, err :=
		http.NewRequest(
			method,
			a.baseURL+
				path,
			reader,
		)
	if err != nil {
		t.Fatalf(
			"build request %s %s: %v",
			method,
			path,
			err,
		)
	}

	if body != nil {
		request.Header.Set(
			"Content-Type",
			"application/json",
		)
	}

	for key, value := range headers {

		request.Header.Set(
			key,
			value,
		)
	}

	response, err :=
		a.client.Do(
			request,
		)
	if err != nil {
		t.Fatalf(
			"request %s %s failed: %v",
			method,
			path,
			err,
		)
	}

	defer response.Body.Close()

	data, err :=
		io.ReadAll(
			io.LimitReader(
				response.Body,
				4*1024*1024,
			),
		)
	if err != nil {
		t.Fatalf(
			"read response %s %s: %v",
			method,
			path,
			err,
		)
	}

	return apiResponse{
		Status: response.StatusCode,

		Body: data,

		Header: response.Header.Clone(),
	}
}

func decodeJSON(
	t *testing.T,
	data []byte,
	target any,
) {
	t.Helper()

	if err :=
		json.Unmarshal(
			data,
			target,
		); err != nil {

		t.Fatalf(
			"decode JSON response: %v\nbody=%s",
			err,
			string(
				data,
			),
		)
	}
}

func assertStatus(
	t *testing.T,
	response apiResponse,
	expected int,
) {
	t.Helper()

	if response.Status !=
		expected {

		t.Fatalf(
			"expected HTTP %d, got %d: %s",
			expected,
			response.Status,
			string(
				response.Body,
			),
		)
	}
}

func assertOneOfStatus(
	t *testing.T,
	response apiResponse,
	expected ...int,
) {
	t.Helper()

	for _, status := range expected {

		if response.Status ==
			status {

			return
		}
	}

	t.Fatalf(
		"unexpected HTTP %d; expected one of %v: %s",
		response.Status,
		expected,
		string(
			response.Body,
		),
	)
}

func errorCode(
	data []byte,
) string {
	var envelope struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}

	if json.Unmarshal(
		data,
		&envelope,
	) != nil {

		return ""
	}

	return envelope.Error.Code
}
