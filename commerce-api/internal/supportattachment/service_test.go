package supportattachment

import (
	"context"
	"errors"
	"testing"
	"time"

	"project.local/commerce-api/internal/platform/storage"
)

const (
	testCustomerID = "11111111-1111-4111-8111-111111111111"

	testSupportActorID = "22222222-2222-4222-8222-222222222222"

	testCaseID = "33333333-3333-4333-8333-333333333333"

	testAttachmentID = "44444444-4444-4444-8444-444444444444"
)

type fakeRepository struct {
	createInput CreatePendingInput

	createResult Attachment
	createErr    error

	getResult Attachment
	getErr    error

	markReadyResult Attachment
	markReadyErr    error

	markReadyCalled bool
}

func (r *fakeRepository) CreatePending(
	_ context.Context,
	input CreatePendingInput,
) (Attachment, error) {
	r.createInput = input

	if r.createErr != nil {
		return Attachment{},
			r.createErr
	}

	if r.createResult.ID != "" {
		return r.createResult, nil
	}

	return Attachment{
		ID: input.ID,

		CaseID: input.CaseID,

		UploaderType: input.UploaderType,

		CustomerID: input.CustomerID,

		SupportActorID: input.SupportActorID,

		StorageKey: input.StorageKey,

		OriginalFilename: input.OriginalFilename,

		MimeType: input.MimeType,

		ByteSize: input.ByteSize,

		Status: StatusPending,
	}, nil
}

func (r *fakeRepository) GetByID(
	_ context.Context,
	_ string,
) (Attachment, error) {
	if r.getErr != nil {
		return Attachment{},
			r.getErr
	}

	return r.getResult, nil
}

func (r *fakeRepository) MarkReady(
	_ context.Context,
	_ string,
	_ UploaderType,
	_ string,
) (Attachment, error) {
	r.markReadyCalled = true

	if r.markReadyErr != nil {
		return Attachment{},
			r.markReadyErr
	}

	return r.markReadyResult, nil
}

type fakeStorageProvider struct {
	uploadRequest storage.UploadRequest

	uploadTarget storage.UploadTarget
	uploadErr    error

	objectInfo storage.ObjectInfo
	statErr    error

	downloadRequest storage.DownloadRequest

	downloadTarget storage.DownloadTarget
	downloadErr    error
}

func (p *fakeStorageProvider) Name() string {
	return "fake"
}

func (p *fakeStorageProvider) CreateUpload(
	_ context.Context,
	request storage.UploadRequest,
) (storage.UploadTarget, error) {
	p.uploadRequest = request

	if p.uploadErr != nil {
		return storage.UploadTarget{},
			p.uploadErr
	}

	if p.uploadTarget.URL != "" {
		return p.uploadTarget, nil
	}

	return storage.UploadTarget{
		Provider: "fake",

		Key: request.Key,

		Method: "PUT",

		URL: "https://example.invalid/upload",

		ExpiresAt: time.Now().
			Add(
				15 * time.Minute,
			),
	}, nil
}

func (p *fakeStorageProvider) CreateDownload(
	_ context.Context,
	request storage.DownloadRequest,
) (storage.DownloadTarget, error) {
	p.downloadRequest = request

	if p.downloadErr != nil {
		return storage.DownloadTarget{},
			p.downloadErr
	}

	if p.downloadTarget.URL != "" {
		return p.downloadTarget, nil
	}

	return storage.DownloadTarget{
		Provider: "fake",

		Key: request.Key,

		URL: "https://example.invalid/download",

		ExpiresAt: time.Now().
			Add(
				request.ExpiresIn,
			),
	}, nil
}

func (p *fakeStorageProvider) PublicURL(
	_ string,
) (string, error) {
	return "",
		errors.New(
			"unexpected PublicURL call",
		)
}

func (p *fakeStorageProvider) Stat(
	_ context.Context,
	_ string,
) (storage.ObjectInfo, error) {
	if p.statErr != nil {
		return storage.ObjectInfo{},
			p.statErr
	}

	return p.objectInfo, nil
}

func (p *fakeStorageProvider) Delete(
	_ context.Context,
	_ string,
) error {
	return nil
}

func TestCreateUploadTargetCustomerStaging(
	t *testing.T,
) {
	repository :=
		&fakeRepository{}

	provider :=
		&fakeStorageProvider{}

	service :=
		NewService(
			repository,
			storage.NewGateway(
				provider,
			),
		)

	result, err :=
		service.CreateUploadTarget(
			context.Background(),
			CreateUploadRequest{
				UploaderType: UploaderCustomer,

				OwnerID: testCustomerID,

				OriginalFilename: `C:\fakepath\damage.PNG`,

				MimeType: "image/png",

				ByteSize: 1024,
			},
		)
	if err != nil {
		t.Fatalf(
			"CreateUploadTarget returned error: %v",
			err,
		)
	}

	if result.Attachment.ID == "" {
		t.Fatal(
			"expected generated attachment id",
		)
	}

	if repository.createInput.CustomerID !=
		testCustomerID {
		t.Fatalf(
			"customer id = %q, want %q",
			repository.createInput.CustomerID,
			testCustomerID,
		)
	}

	if repository.createInput.SupportActorID !=
		"" {
		t.Fatalf(
			"support actor id = %q, want empty",
			repository.createInput.SupportActorID,
		)
	}

	if repository.createInput.CaseID != "" {
		t.Fatalf(
			"case id = %q, want empty staging case",
			repository.createInput.CaseID,
		)
	}

	if repository.createInput.OriginalFilename !=
		"damage.PNG" {
		t.Fatalf(
			"filename = %q, want %q",
			repository.createInput.OriginalFilename,
			"damage.PNG",
		)
	}

	if provider.uploadRequest.Access !=
		storage.AccessPrivate {
		t.Fatalf(
			"access = %q, want %q",
			provider.uploadRequest.Access,
			storage.AccessPrivate,
		)
	}

	if provider.uploadRequest.Purpose !=
		storage.PurposeSupportImage {
		t.Fatalf(
			"purpose = %q, want %q",
			provider.uploadRequest.Purpose,
			storage.PurposeSupportImage,
		)
	}

	expectedPrefix :=
		"private/support/staging/"

	if len(provider.uploadRequest.Key) <=
		len(expectedPrefix) ||
		provider.uploadRequest.Key[:len(expectedPrefix)] !=
			expectedPrefix {
		t.Fatalf(
			"storage key = %q, expected staging prefix",
			provider.uploadRequest.Key,
		)
	}
}

func TestCreateUploadTargetExistingCase(
	t *testing.T,
) {
	repository :=
		&fakeRepository{}

	provider :=
		&fakeStorageProvider{}

	service :=
		NewService(
			repository,
			storage.NewGateway(
				provider,
			),
		)

	_, err :=
		service.CreateUploadTarget(
			context.Background(),
			CreateUploadRequest{
				CaseID: testCaseID,

				UploaderType: UploaderSupport,

				OwnerID: testSupportActorID,

				OriginalFilename: "proof.webp",

				MimeType: "image/webp",

				ByteSize: 2048,
			},
		)
	if err != nil {
		t.Fatalf(
			"CreateUploadTarget returned error: %v",
			err,
		)
	}

	if repository.createInput.CaseID !=
		testCaseID {
		t.Fatalf(
			"case id = %q, want %q",
			repository.createInput.CaseID,
			testCaseID,
		)
	}

	if repository.createInput.SupportActorID !=
		testSupportActorID {
		t.Fatalf(
			"support actor id = %q, want %q",
			repository.createInput.SupportActorID,
			testSupportActorID,
		)
	}

	expectedPrefix :=
		"private/support/" +
			testCaseID +
			"/"

	if len(provider.uploadRequest.Key) <=
		len(expectedPrefix) ||
		provider.uploadRequest.Key[:len(expectedPrefix)] !=
			expectedPrefix {
		t.Fatalf(
			"storage key = %q, expected case prefix",
			provider.uploadRequest.Key,
		)
	}
}

func TestCompleteUploadSuccess(
	t *testing.T,
) {
	pending :=
		Attachment{
			ID: testAttachmentID,

			UploaderType: UploaderCustomer,

			CustomerID: testCustomerID,

			StorageKey: "private/support/staging/" +
				testAttachmentID +
				".jpg",

			OriginalFilename: "damage.jpg",

			MimeType: "image/jpeg",

			ByteSize: 4096,

			Status: StatusPending,
		}

	ready :=
		pending

	ready.Status =
		StatusReady

	now :=
		time.Now().UTC()

	ready.UploadedAt =
		&now

	repository :=
		&fakeRepository{
			getResult: pending,

			markReadyResult: ready,
		}

	provider :=
		&fakeStorageProvider{
			objectInfo: storage.ObjectInfo{
				Key: pending.StorageKey,

				ContentType: "image/jpeg",

				ContentLength: 4096,
			},
		}

	service :=
		NewService(
			repository,
			storage.NewGateway(
				provider,
			),
		)

	result, err :=
		service.CompleteUpload(
			context.Background(),
			testAttachmentID,
			UploaderCustomer,
			testCustomerID,
		)
	if err != nil {
		t.Fatalf(
			"CompleteUpload returned error: %v",
			err,
		)
	}

	if !repository.markReadyCalled {
		t.Fatal(
			"expected MarkReady to be called",
		)
	}

	if result.Status !=
		StatusReady {
		t.Fatalf(
			"status = %q, want %q",
			result.Status,
			StatusReady,
		)
	}
}

func TestCompleteUploadRejectsSizeMismatch(
	t *testing.T,
) {
	pending :=
		Attachment{
			ID: testAttachmentID,

			UploaderType: UploaderCustomer,

			CustomerID: testCustomerID,

			StorageKey: "private/support/staging/" +
				testAttachmentID +
				".png",

			OriginalFilename: "damage.png",

			MimeType: "image/png",

			ByteSize: 4096,

			Status: StatusPending,
		}

	repository :=
		&fakeRepository{
			getResult: pending,
		}

	provider :=
		&fakeStorageProvider{
			objectInfo: storage.ObjectInfo{
				Key: pending.StorageKey,

				ContentType: "image/png",

				ContentLength: 4095,
			},
		}

	service :=
		NewService(
			repository,
			storage.NewGateway(
				provider,
			),
		)

	_, err :=
		service.CompleteUpload(
			context.Background(),
			testAttachmentID,
			UploaderCustomer,
			testCustomerID,
		)

	if !errors.Is(
		err,
		ErrUploadedObjectMismatch,
	) {
		t.Fatalf(
			"error = %v, want ErrUploadedObjectMismatch",
			err,
		)
	}

	if repository.markReadyCalled {
		t.Fatal(
			"MarkReady must not be called for mismatched upload",
		)
	}
}

func TestCompleteUploadIsIdempotentWhenReady(
	t *testing.T,
) {
	ready :=
		Attachment{
			ID: testAttachmentID,

			UploaderType: UploaderCustomer,

			CustomerID: testCustomerID,

			StorageKey: "private/support/staging/" +
				testAttachmentID +
				".webp",

			OriginalFilename: "damage.webp",

			MimeType: "image/webp",

			ByteSize: 1024,

			Status: StatusReady,
		}

	repository :=
		&fakeRepository{
			getResult: ready,
		}

	provider :=
		&fakeStorageProvider{}

	service :=
		NewService(
			repository,
			storage.NewGateway(
				provider,
			),
		)

	result, err :=
		service.CompleteUpload(
			context.Background(),
			testAttachmentID,
			UploaderCustomer,
			testCustomerID,
		)
	if err != nil {
		t.Fatalf(
			"CompleteUpload returned error: %v",
			err,
		)
	}

	if result.Status !=
		StatusReady {
		t.Fatalf(
			"status = %q, want ready",
			result.Status,
		)
	}

	if repository.markReadyCalled {
		t.Fatal(
			"ready upload should not be marked ready again",
		)
	}
}

func TestCreateDownload(
	t *testing.T,
) {
	provider :=
		&fakeStorageProvider{}

	service :=
		NewService(
			&fakeRepository{},
			storage.NewGateway(
				provider,
			),
		)

	attachment :=
		Attachment{
			ID: testAttachmentID,

			StorageKey: "private/support/" +
				testCaseID +
				"/" +
				testAttachmentID +
				".jpg",

			OriginalFilename: "evidence.jpg",

			MimeType: "image/jpeg",

			ByteSize: 1234,

			Status: StatusAttached,
		}

	result, err :=
		service.CreateDownload(
			context.Background(),
			attachment,
		)
	if err != nil {
		t.Fatalf(
			"CreateDownload returned error: %v",
			err,
		)
	}

	if result.URL == "" {
		t.Fatal(
			"expected signed download URL",
		)
	}

	if provider.downloadRequest.Key !=
		attachment.StorageKey {
		t.Fatalf(
			"download key = %q, want %q",
			provider.downloadRequest.Key,
			attachment.StorageKey,
		)
	}

	if provider.downloadRequest.ExpiresIn !=
		DownloadURLTTL {
		t.Fatalf(
			"download ttl = %s, want %s",
			provider.downloadRequest.ExpiresIn,
			DownloadURLTTL,
		)
	}
}

func TestCreateDownloadRejectsDeletedAttachment(
	t *testing.T,
) {
	provider :=
		&fakeStorageProvider{}

	service :=
		NewService(
			&fakeRepository{},
			storage.NewGateway(
				provider,
			),
		)

	now :=
		time.Now().UTC()

	_, err :=
		service.CreateDownload(
			context.Background(),
			Attachment{
				ID: testAttachmentID,

				StorageKey: "private/support/" +
					testCaseID +
					"/" +
					testAttachmentID +
					".jpg",

				Status: StatusDeleted,

				DeletedAt: &now,
			},
		)

	if !errors.Is(
		err,
		ErrInvalidState,
	) {
		t.Fatalf(
			"error = %v, want ErrInvalidState",
			err,
		)
	}
}
