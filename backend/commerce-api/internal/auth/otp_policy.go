package auth

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

type OTPPolicy struct {
	Enabled bool

	AllowUnverifiedSignup bool
}

func DefaultOTPPolicy() OTPPolicy {
	return OTPPolicy{
		/*
			Fail toward verification if the environment
			switch was not wired for some reason.
		*/
		Enabled: true,

		AllowUnverifiedSignup: false,
	}
}

func LoadOTPPolicyFromEnv() (
	OTPPolicy,
	error,
) {
	enabled, err :=
		readBoolEnv(
			"AUTH_OTP_ENABLED",
			true,
		)
	if err != nil {
		return OTPPolicy{},
			err
	}

	allowUnverifiedSignup, err :=
		readBoolEnv(
			"AUTH_ALLOW_UNVERIFIED_SIGNUP",
			false,
		)
	if err != nil {
		return OTPPolicy{},
			err
	}

	return OTPPolicy{
		Enabled: enabled,

		AllowUnverifiedSignup: allowUnverifiedSignup,
	}, nil
}

func readBoolEnv(
	key string,
	fallback bool,
) (
	bool,
	error,
) {
	raw :=
		strings.TrimSpace(
			os.Getenv(
				key,
			),
		)

	if raw == "" {
		return fallback,
			nil
	}

	value, err :=
		strconv.ParseBool(
			raw,
		)
	if err != nil {
		return false,
			fmt.Errorf(
				"invalid %s: %w",
				key,
				err,
			)
	}

	return value,
		nil
}
