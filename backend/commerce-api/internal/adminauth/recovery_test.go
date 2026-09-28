package adminauth

import (
	"testing"
)

func TestRecoveryCodes(
	t *testing.T,
) {
	codes, err :=
		GenerateRecoveryCodes(
			DefaultRecoveryCodeCount,
		)
	if err != nil {
		t.Fatalf(
			"generate recovery codes: %v",
			err,
		)
	}

	if len(codes) !=
		DefaultRecoveryCodeCount {

		t.Fatalf(
			"expected %d recovery codes, got %d",
			DefaultRecoveryCodeCount,
			len(codes),
		)
	}

	seen :=
		make(
			map[string]struct{},
		)

	for _, code := range codes {

		if _, exists :=
			seen[code]; exists {

			t.Fatalf(
				"duplicate recovery code %q",
				code,
			)
		}

		seen[code] = struct{}{}

		hash, err :=
			HashRecoveryCode(
				code,
			)
		if err != nil {
			t.Fatalf(
				"hash recovery code: %v",
				err,
			)
		}

		if len(hash) != 64 {
			t.Fatalf(
				"expected SHA-256 hex hash length 64, got %d",
				len(hash),
			)
		}
	}
}

func TestRecoveryCodeNormalization(
	t *testing.T,
) {
	const code = "ABCD-EFGH-IJKL-MNOP-QRST"

	first, err :=
		HashRecoveryCode(
			code,
		)
	if err != nil {
		t.Fatalf(
			"hash formatted code: %v",
			err,
		)
	}

	second, err :=
		HashRecoveryCode(
			"abcdefghijklmnopqrst",
		)
	if err != nil {
		t.Fatalf(
			"hash normalized code: %v",
			err,
		)
	}

	if first != second {
		t.Fatal(
			"equivalent recovery-code forms must produce the same hash",
		)
	}
}
