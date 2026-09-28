package search

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	platformhttpclient "project.local/commerce-api/internal/platform/httpclient"
)

const voyageMultimodalEndpoint = "https://api.voyageai.com/v1/multimodalembeddings"

type Embedder interface {
	Model() string

	EmbedQuery(
		ctx context.Context,
		text string,
	) ([]float32, error)

	EmbedDocuments(
		ctx context.Context,
		texts []string,
	) ([][]float32, error)
}

type ImageQueryEmbedder interface {
	Model() string

	EmbedImageQuery(
		ctx context.Context,
		mediaType string,
		data []byte,
	) ([]float32, error)
}

type MultimodalDocument struct {
	Text     string
	ImageURL string
}

type MultimodalDocumentEmbedder interface {
	Model() string

	EmbedMultimodalDocuments(
		ctx context.Context,
		documents []MultimodalDocument,
	) ([][]float32, error)
}

type voyageEmbedder struct {
	apiKey string
	model  string
	client *platformhttpclient.Client
}

type voyageContent struct {
	Type string `json:"type"`

	Text string `json:"text,omitempty"`

	ImageURL string `json:"image_url,omitempty"`

	ImageBase64 string `json:"image_base64,omitempty"`
}

type voyageInput struct {
	Content []voyageContent `json:"content"`
}

type voyageEmbeddingRequest struct {
	Inputs []voyageInput `json:"inputs"`

	Model string `json:"model"`

	InputType string `json:"input_type"`

	Truncation bool `json:"truncation"`
}

type voyageEmbeddingData struct {
	Embedding []float32 `json:"embedding"`
	Index     int       `json:"index"`
}

type voyageEmbeddingResponse struct {
	Object string `json:"object"`

	Data []voyageEmbeddingData `json:"data"`

	Model string `json:"model"`
}

func NewVoyageEmbedderFromEnv() (
	Embedder,
	error,
) {
	cfg, err :=
		loadSemanticConfigFromEnv()
	if err != nil {
		return nil, err
	}

	if !cfg.Enabled {
		return nil,
			fmt.Errorf(
				"semantic search is disabled; set SEARCH_SEMANTIC_ENABLED=true",
			)
	}

	return newVoyageEmbedder(
		cfg,
	), nil
}

func newVoyageEmbedder(
	cfg semanticConfig,
) Embedder {
	clientConfig :=
		platformhttpclient.DefaultConfig()

	clientConfig.Timeout =
		cfg.Timeout

	clientConfig.MaxResponseBytes =
		16 << 20

	return &voyageEmbedder{
		apiKey: cfg.APIKey,

		model: cfg.Model,

		client: platformhttpclient.New(
			clientConfig,
		),
	}
}

func (v *voyageEmbedder) Model() string {
	return v.model
}

func (v *voyageEmbedder) EmbedQuery(
	ctx context.Context,
	text string,
) ([]float32, error) {
	text =
		strings.TrimSpace(
			text,
		)

	if text == "" {
		return nil,
			fmt.Errorf(
				"embedding query is blank",
			)
	}

	result, err :=
		v.embedInputs(
			ctx,
			[]voyageInput{
				{
					Content: []voyageContent{
						{
							Type: "text",
							Text: text,
						},
					},
				},
			},
			"query",
		)
	if err != nil {
		return nil, err
	}

	if len(result) != 1 {
		return nil,
			fmt.Errorf(
				"expected one query embedding, got %d",
				len(result),
			)
	}

	return result[0],
		nil
}

func (v *voyageEmbedder) EmbedImageQuery(
	ctx context.Context,
	mediaType string,
	data []byte,
) ([]float32, error) {
	mediaType =
		strings.ToLower(
			strings.TrimSpace(
				mediaType,
			),
		)

	if !isSupportedSearchImageMediaType(
		mediaType,
	) {
		return nil,
			ErrUnsupportedImageType
	}

	if len(data) == 0 {
		return nil,
			ErrInvalidImage
	}

	if int64(len(data)) >
		maxSearchImageBytes {
		return nil,
			ErrImageTooLarge
	}

	dataURL :=
		"data:" +
			mediaType +
			";base64," +
			base64.StdEncoding.EncodeToString(
				data,
			)

	result, err :=
		v.embedInputs(
			ctx,
			[]voyageInput{
				{
					Content: []voyageContent{
						{
							Type: "image_base64",

							ImageBase64: dataURL,
						},
					},
				},
			},
			"query",
		)
	if err != nil {
		return nil, err
	}

	if len(result) != 1 {
		return nil,
			fmt.Errorf(
				"expected one image query embedding, got %d",
				len(result),
			)
	}

	return result[0],
		nil
}

func (v *voyageEmbedder) EmbedDocuments(
	ctx context.Context,
	texts []string,
) ([][]float32, error) {
	if len(texts) == 0 {
		return make(
			[][]float32,
			0,
		), nil
	}

	inputs :=
		make(
			[]voyageInput,
			0,
			len(texts),
		)

	for _, text := range texts {
		text =
			strings.TrimSpace(
				text,
			)

		if text == "" {
			return nil,
				fmt.Errorf(
					"embedding input contains blank text",
				)
		}

		inputs =
			append(
				inputs,
				voyageInput{
					Content: []voyageContent{
						{
							Type: "text",

							Text: text,
						},
					},
				},
			)
	}

	return v.embedInputs(
		ctx,
		inputs,
		"document",
	)
}

func (v *voyageEmbedder) EmbedMultimodalDocuments(
	ctx context.Context,
	documents []MultimodalDocument,
) ([][]float32, error) {
	if len(documents) == 0 {
		return make(
			[][]float32,
			0,
		), nil
	}

	inputs :=
		make(
			[]voyageInput,
			0,
			len(documents),
		)

	for _, document := range documents {
		text :=
			strings.TrimSpace(
				document.Text,
			)

		if text == "" {
			return nil,
				fmt.Errorf(
					"embedding document contains blank text",
				)
		}

		content :=
			[]voyageContent{
				{
					Type: "text",

					Text: text,
				},
			}

		if imageURL :=
			normalizedRemoteImageURL(
				document.ImageURL,
			); imageURL != "" {

			content =
				append(
					content,
					voyageContent{
						Type: "image_url",

						ImageURL: imageURL,
					},
				)
		}

		inputs =
			append(
				inputs,
				voyageInput{
					Content: content,
				},
			)
	}

	return v.embedInputs(
		ctx,
		inputs,
		"document",
	)
}

func (v *voyageEmbedder) embedInputs(
	ctx context.Context,
	inputs []voyageInput,
	inputType string,
) ([][]float32, error) {
	if len(inputs) == 0 {
		return make(
			[][]float32,
			0,
		), nil
	}

	if len(inputs) > 1000 {
		return nil,
			fmt.Errorf(
				"embedding batch exceeds 1000 inputs",
			)
	}

	payload :=
		voyageEmbeddingRequest{
			Inputs: inputs,

			Model: v.model,

			InputType: inputType,

			Truncation: true,
		}

	body, err :=
		json.Marshal(
			payload,
		)
	if err != nil {
		return nil,
			fmt.Errorf(
				"encode embedding request: %w",
				err,
			)
	}

	request, err :=
		http.NewRequestWithContext(
			ctx,
			http.MethodPost,
			voyageMultimodalEndpoint,
			bytes.NewReader(
				body,
			),
		)
	if err != nil {
		return nil,
			fmt.Errorf(
				"create embedding request: %w",
				err,
			)
	}

	request.Header.Set(
		"Authorization",
		"Bearer "+v.apiKey,
	)

	request.Header.Set(
		"Content-Type",
		"application/json",
	)

	// Voyage embedding POST is computational and does not create
	// durable provider-side state, so retrying it is safe.
	request =
		platformhttpclient.AllowRetry(
			request,
		)

	response, err :=
		v.client.Do(
			request,
		)
	if err != nil {
		return nil,
			fmt.Errorf(
				"call embedding provider: %w",
				err,
			)
	}

	defer response.Body.Close()

	responseBody, err :=
		v.client.ReadAll(
			response,
		)
	if err != nil {
		return nil,
			fmt.Errorf(
				"read embedding response: %w",
				err,
			)
	}

	if response.StatusCode <
		http.StatusOK ||
		response.StatusCode >=
			http.StatusMultipleChoices {

		return nil,
			fmt.Errorf(
				"embedding provider returned HTTP %d: %s",
				response.StatusCode,
				boundedProviderMessage(
					responseBody,
					4096,
				),
			)
	}

	var decoded voyageEmbeddingResponse

	if err :=
		json.Unmarshal(
			responseBody,
			&decoded,
		); err != nil {

		return nil,
			fmt.Errorf(
				"decode embedding response: %w",
				err,
			)
	}

	if len(decoded.Data) !=
		len(inputs) {

		return nil,
			fmt.Errorf(
				"embedding provider returned %d vectors for %d inputs",
				len(decoded.Data),
				len(inputs),
			)
	}

	result :=
		make(
			[][]float32,
			len(inputs),
		)

	seen :=
		make(
			[]bool,
			len(inputs),
		)

	for _, item := range decoded.Data {
		if item.Index < 0 ||
			item.Index >= len(inputs) {

			return nil,
				fmt.Errorf(
					"embedding provider returned invalid index %d",
					item.Index,
				)
		}

		if seen[item.Index] {
			return nil,
				fmt.Errorf(
					"embedding provider returned duplicate index %d",
					item.Index,
				)
		}

		if err :=
			validateEmbedding(
				item.Embedding,
			); err != nil {

			return nil,
				fmt.Errorf(
					"embedding index %d: %w",
					item.Index,
					err,
				)
		}

		result[item.Index] =
			item.Embedding

		seen[item.Index] =
			true
	}

	for index, found := range seen {
		if !found {
			return nil,
				fmt.Errorf(
					"embedding provider omitted index %d",
					index,
				)
		}
	}

	return result, nil
}

func boundedProviderMessage(
	body []byte,
	limit int,
) string {
	message :=
		strings.TrimSpace(
			string(
				body,
			),
		)

	if limit <= 0 ||
		len(message) <= limit {

		return message
	}

	return message[:limit]
}
