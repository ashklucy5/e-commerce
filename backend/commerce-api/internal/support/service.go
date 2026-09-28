package support

import (
	"context"
	"encoding/hex"
	"errors"
	"strings"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"

	"project.local/commerce-api/internal/staff"
)

type Service struct {
	repository Repository
}

func NewService(
	repository Repository,
) *Service {
	return &Service{
		repository: repository,
	}
}

func (s *Service) Me(
	ctx context.Context,
	account staff.Account,
) (MeResponse, error) {
	actor, err := s.getActiveActor(
		ctx,
		account.ID,
	)
	if err != nil {
		return MeResponse{}, err
	}

	queues, err :=
		s.repository.ListQueuesForActor(
			ctx,
			actor.ID,
		)
	if err != nil {
		return MeResponse{}, err
	}

	return MeResponse{
		Staff: account,

		Actor: actor,

		Queues: queues,
	}, nil
}

func (s *Service) Queues(
	ctx context.Context,
	staffID string,
) ([]Queue, error) {
	actor, err :=
		s.getActiveActor(
			ctx,
			staffID,
		)
	if err != nil {
		return nil, err
	}

	return s.repository.ListQueuesForActor(
		ctx,
		actor.ID,
	)
}

func (s *Service) UpdatePresence(
	ctx context.Context,
	staffID string,
	request UpdatePresenceRequest,
) (Actor, error) {
	presence :=
		strings.ToLower(
			strings.TrimSpace(
				request.Presence,
			),
		)

	if !validPresence(
		presence,
	) {
		return Actor{},
			ErrInvalidPresence
	}

	actor, err :=
		s.getActiveActor(
			ctx,
			staffID,
		)
	if err != nil {
		return Actor{}, err
	}

	updated, err :=
		s.repository.UpdatePresence(
			ctx,
			actor.ID,
			presence,
		)
	if err != nil {
		if errors.Is(
			err,
			pgx.ErrNoRows,
		) {
			return Actor{},
				ErrSupportActorDisabled
		}

		return Actor{}, err
	}

	return updated, nil
}

func (s *Service) ListCases(
	ctx context.Context,
	staffID string,
	limit int,
	offset int,
) ([]Ticket, int, int, error) {
	actor, err :=
		s.getActiveActor(
			ctx,
			staffID,
		)
	if err != nil {
		return nil,
			0,
			0,
			err
	}

	if limit <= 0 {
		limit = 50
	}

	if limit > 100 {
		limit = 100
	}

	if offset < 0 {
		return nil,
			0,
			0,
			ErrCaseNotFound
	}

	items, err :=
		s.repository.ListCasesForActor(
			ctx,
			actor.ID,
			limit,
			offset,
		)
	if err != nil {
		return nil,
			0,
			0,
			err
	}

	return items,
		limit,
		offset,
		nil
}

func (s *Service) GetCase(
	ctx context.Context,
	staffID string,
	caseID string,
) (Ticket, error) {
	if !validSupportUUID(
		caseID,
	) {
		return Ticket{},
			ErrCaseNotFound
	}

	actor, err :=
		s.getActiveActor(
			ctx,
			staffID,
		)
	if err != nil {
		return Ticket{}, err
	}

	return s.repository.GetCaseForActor(
		ctx,
		actor.ID,
		caseID,
	)
}

func (s *Service) ListMessages(
	ctx context.Context,
	staffID string,
	caseID string,
	limit int,
	offset int,
) ([]SupportMessage, int, int, error) {
	if !validSupportUUID(
		caseID,
	) {
		return nil,
			0,
			0,
			ErrCaseNotFound
	}

	if limit <= 0 {
		limit = 100
	}

	if limit > 200 {
		limit = 200
	}

	if offset < 0 {
		return nil,
			0,
			0,
			ErrCaseNotFound
	}

	actor, err :=
		s.getActiveActor(
			ctx,
			staffID,
		)
	if err != nil {
		return nil,
			0,
			0,
			err
	}

	items, err :=
		s.repository.ListCaseMessagesForActor(
			ctx,
			actor.ID,
			caseID,
			limit,
			offset,
		)
	if err != nil {
		return nil,
			0,
			0,
			err
	}

	return items,
		limit,
		offset,
		nil
}

func (s *Service) ClaimCase(
	ctx context.Context,
	staffID string,
	caseID string,
) (Ticket, error) {
	if !validSupportUUID(
		caseID,
	) {
		return Ticket{},
			ErrCaseNotFound
	}

	actor, err :=
		s.getActiveActor(
			ctx,
			staffID,
		)
	if err != nil {
		return Ticket{}, err
	}

	return s.repository.ClaimCase(
		ctx,
		actor.ID,
		caseID,
	)
}

func (s *Service) Reply(
	ctx context.Context,
	staffID string,
	caseID string,
	request ReplyRequest,
) (SupportMessage, error) {
	if !validSupportUUID(
		caseID,
	) {
		return SupportMessage{},
			ErrCaseNotFound
	}

	body :=
		strings.TrimSpace(
			request.Message,
		)

	if body == "" ||
		utf8.RuneCountInString(
			body,
		) > 5000 {
		return SupportMessage{},
			ErrInvalidMessage
	}

	visibility :=
		strings.ToLower(
			strings.TrimSpace(
				request.Visibility,
			),
		)

	if visibility == "" {
		visibility =
			"customer"
	}

	if visibility != "customer" &&
		visibility != "internal" {
		return SupportMessage{},
			ErrInvalidVisibility
	}

	actor, err :=
		s.getActiveActor(
			ctx,
			staffID,
		)
	if err != nil {
		return SupportMessage{}, err
	}

	return s.repository.AddSupportMessage(
		ctx,
		actor.ID,
		caseID,
		body,
		visibility,
	)
}

func (s *Service) Resolve(
	ctx context.Context,
	staffID string,
	caseID string,
) (Ticket, error) {
	if !validSupportUUID(
		caseID,
	) {
		return Ticket{},
			ErrCaseNotFound
	}

	actor, err :=
		s.getActiveActor(
			ctx,
			staffID,
		)
	if err != nil {
		return Ticket{}, err
	}

	return s.repository.ResolveCase(
		ctx,
		actor.ID,
		caseID,
	)
}

func (s *Service) Escalate(
	ctx context.Context,
	staffID string,
	caseID string,
	request EscalateRequest,
) (EscalationResult, error) {
	if !validSupportUUID(
		caseID,
	) {
		return EscalationResult{},
			ErrCaseNotFound
	}

	queueCode :=
		strings.ToLower(
			strings.TrimSpace(
				request.QueueCode,
			),
		)

	if queueCode == "" ||
		len(queueCode) > 80 {
		return EscalationResult{},
			ErrInvalidQueue
	}

	actor, err :=
		s.getActiveActor(
			ctx,
			staffID,
		)
	if err != nil {
		return EscalationResult{}, err
	}

	return s.repository.EscalateCase(
		ctx,
		actor.ID,
		caseID,
		queueCode,
	)
}

func (s *Service) getActiveActor(
	ctx context.Context,
	staffID string,
) (Actor, error) {
	actor, err :=
		s.repository.GetActorByStaffID(
			ctx,
			staffID,
		)
	if err != nil {
		if errors.Is(
			err,
			pgx.ErrNoRows,
		) {
			return Actor{},
				ErrNotSupportActor
		}

		return Actor{}, err
	}

	if actor.Status != "active" {
		return Actor{},
			ErrSupportActorDisabled
	}

	return actor, nil
}

func validPresence(
	presence string,
) bool {
	switch presence {
	case "offline",
		"available",
		"busy",
		"away":
		return true

	default:
		return false
	}
}

func validSupportUUID(
	value string,
) bool {
	if len(value) != 36 ||
		value[8] != '-' ||
		value[13] != '-' ||
		value[18] != '-' ||
		value[23] != '-' {
		return false
	}

	compact :=
		strings.ReplaceAll(
			value,
			"-",
			"",
		)

	if len(compact) != 32 {
		return false
	}

	_, err :=
		hex.DecodeString(
			compact,
		)

	return err == nil
}
