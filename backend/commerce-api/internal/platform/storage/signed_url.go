package storage

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/url"
	"strings"
	"time"
)

const (
	awsAlgorithm = "AWS4-HMAC-SHA256"

	awsService = "s3"

	unsignedPayload = "UNSIGNED-PAYLOAD"
)

func sha256Hex(
	value string,
) string {
	digest := sha256.Sum256(
		[]byte(value),
	)

	return hex.EncodeToString(
		digest[:],
	)
}

func hmacSHA256(
	key []byte,
	value string,
) []byte {
	mac := hmac.New(
		sha256.New,
		key,
	)

	_, _ = mac.Write(
		[]byte(value),
	)

	return mac.Sum(nil)
}

func signingKey(
	secret string,
	date string,
	region string,
) []byte {
	dateKey := hmacSHA256(
		[]byte(
			"AWS4"+secret,
		),
		date,
	)

	regionKey := hmacSHA256(
		dateKey,
		region,
	)

	serviceKey := hmacSHA256(
		regionKey,
		awsService,
	)

	return hmacSHA256(
		serviceKey,
		"aws4_request",
	)
}

func credentialScope(
	now time.Time,
	region string,
) string {
	return strings.Join(
		[]string{
			now.UTC().Format(
				"20060102",
			),
			region,
			awsService,
			"aws4_request",
		},
		"/",
	)
}

func canonicalQuery(
	values url.Values,
) string {
	return values.Encode()
}
