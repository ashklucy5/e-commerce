package notification

import (
	"context"
	"fmt"
	"strings"
)

func (s *Service) ListCustomerInbox(
	ctx context.Context,
	customerID string,
	options InboxListOptions,
) (InboxListResult, error) {
	if s == nil || s.repository == nil {
		return InboxListResult{}, fmt.Errorf("notification service is not configured")
	}

	customerID = strings.TrimSpace(customerID)

	if customerID == "" ||
		!validOptionalUUID(customerID) {

		return InboxListResult{}, ErrInvalidInput
	}

	return s.repository.ListCustomerInbox(
		ctx,
		customerID,
		options,
	)
}

func (s *Service) CustomerInboxSummary(
	ctx context.Context,
	customerID string,
) (InboxSummary, error) {
	if s == nil || s.repository == nil {
		return InboxSummary{}, fmt.Errorf("notification service is not configured")
	}

	customerID = strings.TrimSpace(customerID)

	if customerID == "" ||
		!validOptionalUUID(customerID) {

		return InboxSummary{}, ErrInvalidInput
	}

	return s.repository.CustomerInboxSummary(
		ctx,
		customerID,
	)
}

func (s *Service) MarkCustomerInboxRead(
	ctx context.Context,
	customerID string,
	notificationID string,
) (bool, error) {
	if s == nil || s.repository == nil {
		return false, fmt.Errorf("notification service is not configured")
	}

	customerID = strings.TrimSpace(customerID)
	notificationID = strings.TrimSpace(notificationID)

	if customerID == "" ||
		notificationID == "" ||
		!validOptionalUUID(customerID) ||
		!validOptionalUUID(notificationID) {

		return false, ErrInvalidInput
	}

	return s.repository.MarkCustomerInboxRead(
		ctx,
		customerID,
		notificationID,
	)
}

func (s *Service) MarkAllCustomerInboxRead(
	ctx context.Context,
	customerID string,
) (int64, error) {
	if s == nil || s.repository == nil {
		return 0, fmt.Errorf("notification service is not configured")
	}

	customerID = strings.TrimSpace(customerID)

	if customerID == "" ||
		!validOptionalUUID(customerID) {

		return 0, ErrInvalidInput
	}

	return s.repository.MarkAllCustomerInboxRead(
		ctx,
		customerID,
	)
}

func (s *Service) ListStaffInbox(
	ctx context.Context,
	staffID string,
	options InboxListOptions,
) (InboxListResult, error) {
	if s == nil || s.repository == nil {
		return InboxListResult{}, fmt.Errorf("notification service is not configured")
	}

	staffID = strings.TrimSpace(staffID)

	if staffID == "" ||
		!validOptionalUUID(staffID) {

		return InboxListResult{}, ErrInvalidInput
	}

	return s.repository.ListStaffInbox(
		ctx,
		staffID,
		options,
	)
}

func (s *Service) StaffInboxSummary(
	ctx context.Context,
	staffID string,
) (InboxSummary, error) {
	if s == nil || s.repository == nil {
		return InboxSummary{}, fmt.Errorf("notification service is not configured")
	}

	staffID = strings.TrimSpace(staffID)

	if staffID == "" ||
		!validOptionalUUID(staffID) {

		return InboxSummary{}, ErrInvalidInput
	}

	return s.repository.StaffInboxSummary(
		ctx,
		staffID,
	)
}

func (s *Service) MarkStaffInboxRead(
	ctx context.Context,
	staffID string,
	notificationID string,
) (bool, error) {
	if s == nil || s.repository == nil {
		return false, fmt.Errorf("notification service is not configured")
	}

	staffID = strings.TrimSpace(staffID)
	notificationID = strings.TrimSpace(notificationID)

	if staffID == "" ||
		notificationID == "" ||
		!validOptionalUUID(staffID) ||
		!validOptionalUUID(notificationID) {

		return false, ErrInvalidInput
	}

	return s.repository.MarkStaffInboxRead(
		ctx,
		staffID,
		notificationID,
	)
}

func (s *Service) MarkAllStaffInboxRead(
	ctx context.Context,
	staffID string,
) (int64, error) {
	if s == nil || s.repository == nil {
		return 0, fmt.Errorf("notification service is not configured")
	}

	staffID = strings.TrimSpace(staffID)

	if staffID == "" ||
		!validOptionalUUID(staffID) {

		return 0, ErrInvalidInput
	}

	return s.repository.MarkAllStaffInboxRead(
		ctx,
		staffID,
	)
}
