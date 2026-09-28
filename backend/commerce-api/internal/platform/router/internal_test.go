package router

import "testing"

func TestValidRuntimeTickAuthorization(
	t *testing.T,
) {
	const token = "0123456789abcdef0123456789abcdef"

	tests :=
		[]struct {
			name string

			header string

			expected bool
		}{
			{
				name: "valid",

				header: "Bearer " + token,

				expected: true,
			},
			{
				name: "bearer case insensitive",

				header: "bearer " + token,

				expected: true,
			},
			{
				name: "missing",

				header: "",

				expected: false,
			},
			{
				name: "wrong scheme",

				header: "Basic " + token,

				expected: false,
			},
			{
				name: "wrong token",

				header: "Bearer wrong-token",

				expected: false,
			},
			{
				name: "missing expected token",

				header: "Bearer " + token,

				expected: false,
			},
		}

	for _, test := range tests {

		t.Run(
			test.name,
			func(
				t *testing.T,
			) {
				expectedToken :=
					token

				if test.name ==
					"missing expected token" {

					expectedToken =
						""
				}

				actual :=
					validRuntimeTickAuthorization(
						test.header,
						expectedToken,
					)

				if actual !=
					test.expected {

					t.Fatalf(
						"expected %v, got %v",
						test.expected,
						actual,
					)
				}
			},
		)
	}
}
