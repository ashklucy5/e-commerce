package returns

func orderEligibleForReturn(
	status string,
) bool {
	switch status {
	case "delivered",
		"completed":
		return true

	default:
		return false
	}
}

func expectedSourceStatus(
	target string,
) (string, bool) {
	switch target {
	case StatusApproved:
		return StatusRequested, true

	case StatusRejected:
		return StatusRequested, true

	case StatusReceived:
		return StatusApproved, true

	case StatusInspected:
		return StatusReceived, true

	case StatusCompleted:
		return StatusInspected, true

	default:
		return "", false
	}
}
