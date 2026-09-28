package adminauth

import (
	"encoding/base32"
	"net/url"
	"testing"
	"time"
)

func TestGenerateTOTPSecret(
	t *testing.T,
) {
	secret, err :=
		GenerateTOTPSecret()
	if err != nil {
		t.Fatalf(
			"generate secret: %v",
			err,
		)
	}

	decoded, err :=
		base32.StdEncoding.
			WithPadding(
				base32.NoPadding,
			).
			DecodeString(
				secret,
			)
	if err != nil {
		t.Fatalf(
			"decode secret: %v",
			err,
		)
	}

	if len(decoded) !=
		DefaultTOTPSecretBytes {

		t.Fatalf(
			"expected %d bytes, got %d",
			DefaultTOTPSecretBytes,
			len(decoded),
		)
	}
}

// RFC 6238 Appendix B:
//
// Secret:
// 12345678901234567890
//
// At Unix time 59 with SHA1, 8 digits and 30-second period,
// the expected TOTP is 94287082.
func TestTOTPMatchesRFC6238Vector(
	t *testing.T,
) {
	secret :=
		base32.StdEncoding.
			WithPadding(
				base32.NoPadding,
			).
			EncodeToString(
				[]byte(
					"12345678901234567890",
				),
			)

	cfg :=
		TOTPConfig{
			Algorithm: "SHA1",

			Digits: 8,

			PeriodSeconds: 30,

			Skew: 0,
		}

	code,
		step,
		err :=
		GenerateTOTP(
			secret,
			time.Unix(
				59,
				0,
			),
			cfg,
		)
	if err != nil {
		t.Fatalf(
			"generate TOTP: %v",
			err,
		)
	}

	if code !=
		"94287082" {

		t.Fatalf(
			"expected 94287082, got %s",
			code,
		)
	}

	if step != 1 {
		t.Fatalf(
			"expected step 1, got %d",
			step,
		)
	}
}

func TestVerifyTOTPAndReplayProtection(
	t *testing.T,
) {
	secret, err :=
		GenerateTOTPSecret()
	if err != nil {
		t.Fatalf(
			"generate secret: %v",
			err,
		)
	}

	cfg :=
		DefaultTOTPConfig()

	now :=
		time.Unix(
			1_725_000_000,
			0,
		)

	code,
		_,
		err :=
		GenerateTOTP(
			secret,
			now,
			cfg,
		)
	if err != nil {
		t.Fatalf(
			"generate code: %v",
			err,
		)
	}

	verification, err :=
		VerifyTOTP(
			secret,
			code,
			now,
			nil,
			cfg,
		)
	if err != nil {
		t.Fatalf(
			"verify TOTP: %v",
			err,
		)
	}

	if !verification.Valid {
		t.Fatal(
			"expected TOTP to verify",
		)
	}

	lastAccepted :=
		verification.Step

	replayed, err :=
		VerifyTOTP(
			secret,
			code,
			now,
			&lastAccepted,
			cfg,
		)
	if err != nil {
		t.Fatalf(
			"verify replayed TOTP: %v",
			err,
		)
	}

	if replayed.Valid {
		t.Fatal(
			"replayed TOTP unexpectedly verified",
		)
	}
}

func TestVerifyTOTPRejectsWrongCode(
	t *testing.T,
) {
	secret, err :=
		GenerateTOTPSecret()
	if err != nil {
		t.Fatalf(
			"generate secret: %v",
			err,
		)
	}

	result, err :=
		VerifyTOTP(
			secret,
			"000000",
			time.Unix(
				1_725_000_000,
				0,
			),
			nil,
			DefaultTOTPConfig(),
		)
	if err != nil {
		t.Fatalf(
			"verify TOTP: %v",
			err,
		)
	}

	if result.Valid {
		t.Fatal(
			"wrong TOTP unexpectedly verified",
		)
	}
}

func TestBuildTOTPEnrollmentURI(
	t *testing.T,
) {
	secret, err :=
		GenerateTOTPSecret()
	if err != nil {
		t.Fatalf(
			"generate secret: %v",
			err,
		)
	}

	value, err :=
		BuildTOTPEnrollmentURI(
			"Commerce Admin",
			"admin.test@example.com",
			secret,
			DefaultTOTPConfig(),
		)
	if err != nil {
		t.Fatalf(
			"build enrollment URI: %v",
			err,
		)
	}

	parsed, err :=
		url.Parse(
			value,
		)
	if err != nil {
		t.Fatalf(
			"parse enrollment URI: %v",
			err,
		)
	}

	if parsed.Scheme !=
		"otpauth" {

		t.Fatalf(
			"expected otpauth scheme, got %q",
			parsed.Scheme,
		)
	}

	if parsed.Host !=
		"totp" {

		t.Fatalf(
			"expected totp host, got %q",
			parsed.Host,
		)
	}

	query :=
		parsed.Query()

	if query.Get(
		"secret",
	) != secret {

		t.Fatal(
			"enrollment URI secret mismatch",
		)
	}

	if query.Get(
		"issuer",
	) != "Commerce Admin" {

		t.Fatal(
			"enrollment URI issuer mismatch",
		)
	}

	if query.Get(
		"digits",
	) != "6" {

		t.Fatal(
			"enrollment URI digits mismatch",
		)
	}

	if query.Get(
		"period",
	) != "30" {

		t.Fatal(
			"enrollment URI period mismatch",
		)
	}
}
