package support

func DefaultPriority(caseType string) string {
	switch caseType {
	case "payment_issue", "complaint":
		return "high"
	default:
		return "normal"
	}
}
