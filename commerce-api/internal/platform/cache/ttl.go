package cache

import (
	"hash/fnv"
	"time"
)

const (
	PublicCategoriesTTL = 2 * time.Minute

	DefaultTTLJitterPercent = 10
)

func TTLWithJitter(
	key string,
	base time.Duration,
) time.Duration {
	if base <= 0 {
		return 0
	}

	const percent = DefaultTTLJitterPercent

	window :=
		base *
			percent /
			100

	if window <= 0 {
		return base
	}

	hasher :=
		fnv.New64a()

	_, _ =
		hasher.Write(
			[]byte(
				key,
			),
		)

	rangeSize :=
		uint64(
			(window * 2) +
				1,
		)

	offset :=
		time.Duration(
			hasher.Sum64() %
				rangeSize,
		)

	result :=
		base -
			window +
			offset

	if result <= 0 {
		return base
	}

	return result
}
