package inventory

type Service struct {
	repository *Repository
}

func NewService(
	repository *Repository,
) *Service {
	return &Service{
		repository: repository,
	}
}

func validActorPair(
	actorType string,
	actorID string,
) bool {
	if actorType == "" &&
		actorID == "" {
		return true
	}

	return actorType != "" &&
		actorID != ""
}
