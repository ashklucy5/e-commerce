package support

import (
	"encoding/json"
	"errors"
	"testing"
)

func TestParseAdminSupportAttachments(
	t *testing.T,
) {
	t.Parallel()

	tests := []struct {
		name string
		raw  string

		wantCount int
		wantErr   bool
	}{
		{
			name: "empty",

			raw: "",

			wantCount: 0,
		},
		{
			name: "image attachment id",

			raw: `[
					{
						"type": "image",
						"kind": "image",
						"attachment_id": "11111111-1111-4111-8111-111111111111"
					}
				]`,

			wantCount: 1,
		},
		{
			name: "attachment id implies image",

			raw: `[
					{
						"attachment_id": "11111111-1111-4111-8111-111111111111"
					}
				]`,

			wantCount: 1,
		},
		{
			name: "https link",

			raw: `[
					{
						"kind": "link",
						"name": "reference",
						"url": "https://example.com/a"
					}
				]`,

			wantCount: 1,
		},
		{
			name: "base64 image rejected",

			raw: `[
					{
						"kind": "image",
						"name": "screen.jpg",
						"url": "data:image/jpeg;base64,AA=="
					}
				]`,

			wantErr: true,
		},
		{
			name: "image url rejected",

			raw: `[
					{
						"kind": "image",
						"attachment_id": "11111111-1111-4111-8111-111111111111",
						"url": "https://example.com/image.jpg"
					}
				]`,

			wantErr: true,
		},
		{
			name: "invalid image id",

			raw: `[
					{
						"kind": "image",
						"attachment_id": "not-a-uuid"
					}
				]`,

			wantErr: true,
		},
		{
			name: "javascript URL rejected",

			raw: `[
					{
						"kind": "link",
						"url": "javascript:alert(1)"
					}
				]`,

			wantErr: true,
		},
		{
			name: "duplicate attachment id rejected",

			raw: `[
					{
						"kind": "image",
						"attachment_id": "11111111-1111-4111-8111-111111111111"
					},
					{
						"kind": "image",
						"attachment_id": "11111111-1111-4111-8111-111111111111"
					}
				]`,

			wantErr: true,
		},
		{
			name: "too many attachments",

			raw: `[
					{"url":"https://example.com/1"},
					{"url":"https://example.com/2"},
					{"url":"https://example.com/3"},
					{"url":"https://example.com/4"},
					{"url":"https://example.com/5"}
				]`,

			wantErr: true,
		},
	}

	for _, test := range tests {

		test := test

		t.Run(
			test.name,
			func(
				t *testing.T,
			) {
				t.Parallel()

				result, err :=
					parseAdminSupportAttachments(
						json.RawMessage(
							test.raw,
						),
					)

				if test.wantErr {
					if !errors.Is(
						err,
						ErrInvalidMessage,
					) {
						t.Fatalf(
							"error = %v, want ErrInvalidMessage",
							err,
						)
					}

					return
				}

				if err != nil {
					t.Fatalf(
						"parse Admin support attachments: %v",
						err,
					)
				}

				if len(result) !=
					test.wantCount {

					t.Fatalf(
						"attachment count = %d, want %d",
						len(result),
						test.wantCount,
					)
				}
			},
		)
	}
}

func TestParseAdminSupportAttachmentsCanonicalizesImage(
	t *testing.T,
) {
	result, err :=
		parseAdminSupportAttachments(
			json.RawMessage(`
				[
					{
						"attachment_id": "11111111-1111-4111-8111-111111111111"
					}
				]
			`),
		)
	if err != nil {
		t.Fatalf(
			"parse Admin support attachments: %v",
			err,
		)
	}

	if len(result) != 1 {
		t.Fatalf(
			"attachment count = %d, want 1",
			len(result),
		)
	}

	if result[0].Type !=
		"image" {

		t.Fatalf(
			"type = %q, want image",
			result[0].Type,
		)
	}

	if result[0].Kind !=
		"image" {

		t.Fatalf(
			"kind = %q, want image",
			result[0].Kind,
		)
	}
}
