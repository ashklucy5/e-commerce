package refund

import "strings"

type SuccessRequest struct {
	ProviderRefundID string `json:"provider_refund_id"`
}

type FailureRequest struct {
	FailureCode    string `json:"failure_code"`
	FailureMessage string `json:"failure_message"`
}

func normalizeSuccessRequest(
	request *SuccessRequest,
) error {
	request.ProviderRefundID =
		strings.TrimSpace(
			request.ProviderRefundID,
		)

	if request.ProviderRefundID == "" ||
		len(request.ProviderRefundID) > 160 {
		return ErrProviderRefundIDRequired
	}

	return nil
}

func normalizeFailureRequest(
	request *FailureRequest,
) error {
	request.FailureCode =
		strings.TrimSpace(
			request.FailureCode,
		)

	request.FailureMessage =
		strings.TrimSpace(
			request.FailureMessage,
		)

	if request.FailureCode == "" ||
		len(request.FailureCode) > 100 ||
		request.FailureMessage == "" ||
		len(request.FailureMessage) > 500 {
		return ErrFailureDetailsRequired
	}

	return nil
}
