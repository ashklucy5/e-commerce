package returns

import "strings"

const (
	ReasonDamaged        = "damaged"
	ReasonDefective      = "defective"
	ReasonWrongItem      = "wrong_item"
	ReasonNotAsDescribed = "not_as_described"
	ReasonSizeIssue      = "size_issue"
	ReasonQuantityIssue  = "quantity_issue"
	ReasonChangedMind    = "changed_mind"
	ReasonOther          = "other"
)

func validReasonCode(
	value string,
) bool {
	switch strings.ToLower(
		strings.TrimSpace(
			value,
		),
	) {
	case ReasonDamaged,
		ReasonDefective,
		ReasonWrongItem,
		ReasonNotAsDescribed,
		ReasonSizeIssue,
		ReasonQuantityIssue,
		ReasonChangedMind,
		ReasonOther:
		return true

	default:
		return false
	}
}
