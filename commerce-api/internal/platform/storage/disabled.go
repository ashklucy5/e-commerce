package storage

import "context"

type DisabledProvider struct{}

func NewDisabledProvider() *DisabledProvider {
	return &DisabledProvider{}
}

func (p *DisabledProvider) Name() string {
	return "disabled"
}

func (p *DisabledProvider) CreateUpload(
	_ context.Context,
	_ UploadRequest,
) (UploadTarget, error) {
	return UploadTarget{},
		ErrDisabled
}

func (p *DisabledProvider) CreateDownload(
	_ context.Context,
	_ DownloadRequest,
) (DownloadTarget, error) {
	return DownloadTarget{},
		ErrDisabled
}

func (p *DisabledProvider) PublicURL(
	_ string,
) (string, error) {
	return "",
		ErrDisabled
}

func (p *DisabledProvider) Stat(
	_ context.Context,
	_ string,
) (ObjectInfo, error) {
	return ObjectInfo{},
		ErrDisabled
}

func (p *DisabledProvider) Delete(
	_ context.Context,
	_ string,
) error {
	return ErrDisabled
}
