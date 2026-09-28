package validation

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

const (
	CodeRequired   = "required"
	CodeInvalid    = "invalid"
	CodeTooShort   = "too_short"
	CodeTooLong    = "too_long"
	CodeNotAllowed = "not_allowed"
	CodeOutOfRange = "out_of_range"
)

type FieldError struct {
	Field   string `json:"field"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

type Errors struct {
	Fields []FieldError `json:"fields"`
}

func (e *Errors) Error() string {
	if e == nil ||
		len(e.Fields) == 0 {
		return ""
	}

	return fmt.Sprintf(
		"validation failed for %d field(s)",
		len(e.Fields),
	)
}

func (e *Errors) Len() int {
	if e == nil {
		return 0
	}

	return len(
		e.Fields,
	)
}

type Validator struct {
	fields []FieldError
}

func New() *Validator {
	return &Validator{
		fields: make(
			[]FieldError,
			0,
		),
	}
}

func (v *Validator) Add(
	field string,
	code string,
	message string,
) {
	if v == nil {
		return
	}

	field = strings.TrimSpace(
		field,
	)
	if field == "" {
		field = "value"
	}

	code = strings.TrimSpace(
		code,
	)
	if code == "" {
		code = CodeInvalid
	}

	message = strings.TrimSpace(
		message,
	)
	if message == "" {
		message = "Invalid value"
	}

	// Avoid adding the same validation failure more than once.
	for _, existing := range v.fields {
		if existing.Field == field &&
			existing.Code == code {
			return
		}
	}

	v.fields = append(
		v.fields,
		FieldError{
			Field:   field,
			Code:    code,
			Message: message,
		},
	)
}

func (v *Validator) Required(
	field string,
	value string,
) bool {
	if strings.TrimSpace(
		value,
	) != "" {
		return true
	}

	v.Add(
		field,
		CodeRequired,
		fmt.Sprintf(
			"%s is required",
			field,
		),
	)

	return false
}

// MinLength validates non-empty strings.
//
// Use Required separately when the value itself is mandatory.
func (v *Validator) MinLength(
	field string,
	value string,
	minimum int,
) {
	value = strings.TrimSpace(
		value,
	)

	if value == "" {
		return
	}

	if minimum < 0 {
		minimum = 0
	}

	if utf8.RuneCountInString(
		value,
	) >= minimum {
		return
	}

	v.Add(
		field,
		CodeTooShort,
		fmt.Sprintf(
			"%s must be at least %d characters",
			field,
			minimum,
		),
	)
}

// MaxLength validates non-empty strings.
//
// Length is measured as Unicode characters rather than raw bytes.
func (v *Validator) MaxLength(
	field string,
	value string,
	maximum int,
) {
	value = strings.TrimSpace(
		value,
	)

	if value == "" {
		return
	}

	if maximum < 0 {
		maximum = 0
	}

	if utf8.RuneCountInString(
		value,
	) <= maximum {
		return
	}

	v.Add(
		field,
		CodeTooLong,
		fmt.Sprintf(
			"%s must be at most %d characters",
			field,
			maximum,
		),
	)
}

// OneOf validates non-empty values.
//
// Use Required separately when an empty value is not allowed.
func (v *Validator) OneOf(
	field string,
	value string,
	allowed ...string,
) {
	value = strings.TrimSpace(
		value,
	)

	if value == "" {
		return
	}

	for _, candidate := range allowed {
		if value == candidate {
			return
		}
	}

	v.Add(
		field,
		CodeNotAllowed,
		fmt.Sprintf(
			"%s contains an unsupported value",
			field,
		),
	)
}

// UUID validates a conventional hyphenated UUID string.
//
// It deliberately accepts all UUID versions because database IDs,
// externally supplied provider IDs, and future UUID versions should
// not be unnecessarily restricted at the HTTP boundary.
//
// Use Required separately if an empty value is not allowed.
func (v *Validator) UUID(
	field string,
	value string,
) {
	value = strings.TrimSpace(
		value,
	)

	if value == "" {
		return
	}

	if IsUUID(
		value,
	) {
		return
	}

	v.Add(
		field,
		CodeInvalid,
		fmt.Sprintf(
			"%s must be a valid UUID",
			field,
		),
	)
}

func (v *Validator) IntRange(
	field string,
	value int,
	minimum int,
	maximum int,
) {
	if value >= minimum &&
		value <= maximum {
		return
	}

	v.Add(
		field,
		CodeOutOfRange,
		fmt.Sprintf(
			"%s must be between %d and %d",
			field,
			minimum,
			maximum,
		),
	)
}

func (v *Validator) PositiveInt(
	field string,
	value int,
) {
	if value > 0 {
		return
	}

	v.Add(
		field,
		CodeOutOfRange,
		fmt.Sprintf(
			"%s must be greater than zero",
			field,
		),
	)
}

func (v *Validator) NonNegativeInt(
	field string,
	value int,
) {
	if value >= 0 {
		return
	}

	v.Add(
		field,
		CodeOutOfRange,
		fmt.Sprintf(
			"%s cannot be negative",
			field,
		),
	)
}

func (v *Validator) Valid() bool {
	return v == nil ||
		len(v.fields) == 0
}

func (v *Validator) Err() error {
	if v == nil ||
		len(v.fields) == 0 {
		return nil
	}

	fields := make(
		[]FieldError,
		len(v.fields),
	)

	copy(
		fields,
		v.fields,
	)

	return &Errors{
		Fields: fields,
	}
}

func (v *Validator) FieldErrors() []FieldError {
	if v == nil {
		return nil
	}

	result := make(
		[]FieldError,
		len(v.fields),
	)

	copy(
		result,
		v.fields,
	)

	return result
}

func IsUUID(
	value string,
) bool {
	value = strings.TrimSpace(
		value,
	)

	if len(value) != 36 {
		return false
	}

	for index := 0; index < len(value); index++ {
		switch index {
		case 8, 13, 18, 23:
			if value[index] != '-' {
				return false
			}

		default:
			if !isHex(
				value[index],
			) {
				return false
			}
		}
	}

	return true
}

func isHex(
	value byte,
) bool {
	return (value >= '0' && value <= '9') ||
		(value >= 'a' && value <= 'f') ||
		(value >= 'A' && value <= 'F')
}
