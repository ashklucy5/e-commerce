package analytics

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

const (
	DefaultRangeDays = 30
	MaxRangeDays     = 366
	DefaultCurrency  = "BDT"

	GranularityDay  = "day"
	GranularityWeek = "week"

	BusinessTimezone = "Asia/Dhaka"
)

var (
	ErrInvalidDateRange   = errors.New("invalid analytics date range")
	ErrInvalidGranularity = errors.New("invalid analytics granularity")
	ErrInvalidCurrency    = errors.New("invalid analytics currency")
	ErrInvalidLimit       = errors.New("invalid analytics limit")
)

// Bangladesh currently has no DST, so a fixed +06:00 zone gives stable
// business-day boundaries without depending on host timezone data.
var businessLocation = time.FixedZone(BusinessTimezone, 6*60*60)

type Query struct {
	Start time.Time
	End   time.Time

	FromDate string
	ToDate   string

	Granularity string
	Currency    string
}

type QueryMeta struct {
	FromDate    string `json:"from"`
	ToDate      string `json:"to"`
	Timezone    string `json:"timezone"`
	Granularity string `json:"granularity,omitempty"`
	Currency    string `json:"currency,omitempty"`
}

func ParseQuery(
	fromValue,
	toValue,
	granularityValue,
	currencyValue string,
	now time.Time,
) (
	Query,
	error,
) {
	granularity :=
		strings.ToLower(
			strings.TrimSpace(
				granularityValue,
			),
		)

	if granularity == "" {
		granularity =
			GranularityDay
	}

	if granularity != GranularityDay &&
		granularity != GranularityWeek {

		return Query{},
			fmt.Errorf(
				"%w: granularity must be day or week",
				ErrInvalidGranularity,
			)
	}

	currency :=
		strings.ToUpper(
			strings.TrimSpace(
				currencyValue,
			),
		)

	if currency == "" {
		currency =
			DefaultCurrency
	}

	if !validCurrency(
		currency,
	) {
		return Query{},
			fmt.Errorf(
				"%w: currency must be a 3-letter uppercase code",
				ErrInvalidCurrency,
			)
	}

	localNow :=
		now.In(
			businessLocation,
		)

	today :=
		time.Date(
			localNow.Year(),
			localNow.Month(),
			localNow.Day(),
			0,
			0,
			0,
			0,
			businessLocation,
		)

	toDate, err :=
		parseDateOrDefault(
			toValue,
			today,
		)
	if err != nil {
		return Query{}, err
	}

	defaultFrom :=
		toDate.AddDate(
			0,
			0,
			-(DefaultRangeDays - 1),
		)

	fromDate, err :=
		parseDateOrDefault(
			fromValue,
			defaultFrom,
		)
	if err != nil {
		return Query{}, err
	}

	if fromDate.After(
		toDate,
	) {
		return Query{},
			fmt.Errorf(
				"%w: from must not be after to",
				ErrInvalidDateRange,
			)
	}

	days :=
		int(
			toDate.Sub(
				fromDate,
			).Hours()/24,
		) +
			1

	if days >
		MaxRangeDays {

		return Query{},
			fmt.Errorf(
				"%w: range cannot exceed %d days",
				ErrInvalidDateRange,
				MaxRangeDays,
			)
	}

	return Query{
			Start: fromDate.UTC(),

			End: toDate.
				AddDate(
					0,
					0,
					1,
				).
				UTC(),

			FromDate: fromDate.Format(
				time.DateOnly,
			),

			ToDate: toDate.Format(
				time.DateOnly,
			),

			Granularity: granularity,

			Currency: currency,
		},
		nil
}

func (q Query) Meta() QueryMeta {
	return QueryMeta{
		FromDate: q.FromDate,

		ToDate: q.ToDate,

		Timezone: BusinessTimezone,

		Granularity: q.Granularity,

		Currency: q.Currency,
	}
}

func parseDateOrDefault(
	value string,
	fallback time.Time,
) (
	time.Time,
	error,
) {
	value =
		strings.TrimSpace(
			value,
		)

	if value == "" {
		return fallback,
			nil
	}

	parsed, err :=
		time.ParseInLocation(
			time.DateOnly,
			value,
			businessLocation,
		)
	if err != nil {
		return time.Time{},
			fmt.Errorf(
				"%w: dates must use YYYY-MM-DD",
				ErrInvalidDateRange,
			)
	}

	return parsed, nil
}

func validCurrency(
	value string,
) bool {
	if len(value) != 3 {
		return false
	}

	for index :=
		0; index < len(value); index++ {

		if value[index] < 'A' ||
			value[index] > 'Z' {

			return false
		}
	}

	return true
}

func analyticsBucketStart(
	date time.Time,
	granularity string,
) time.Time {
	date =
		date.In(
			businessLocation,
		)

	date =
		time.Date(
			date.Year(),
			date.Month(),
			date.Day(),
			0,
			0,
			0,
			0,
			businessLocation,
		)

	if granularity !=
		GranularityWeek {

		return date
	}

	weekday :=
		int(
			date.Weekday(),
		)

	if weekday == 0 {
		weekday = 7
	}

	return date.AddDate(
		0,
		0,
		-(weekday - 1),
	)
}
