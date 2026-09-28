package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"project.local/commerce-api/internal/platform/config"
	"project.local/commerce-api/internal/platform/storage"
)

const (
	liveCheckTimeout = 45 * time.Second
	cleanupTimeout   = 15 * time.Second
)

func main() {
	cfg, err := config.LoadStorage()
	if err != nil {
		log.Fatalf(
			"storage configuration error: %v",
			err,
		)
	}

	gateway, err :=
		storage.NewFromConfig(cfg)
	if err != nil {
		log.Fatalf(
			"storage startup error: %v",
			err,
		)
	}

	fmt.Println("Storage Gateway")
	fmt.Println("---------------")

	fmt.Printf(
		"Provider: %s\n",
		gateway.ProviderName(),
	)

	fmt.Println(
		"Credentials: not displayed",
	)

	if cfg.Provider == "disabled" {
		fmt.Println(
			"Status: disabled",
		)

		fmt.Println(
			"Network calls: none",
		)

		fmt.Println(
			"Storage cost: none",
		)

		return
	}

	if !envEnabled(
		"STORAGE_CHECK_LIVE",
	) {
		fmt.Println(
			"Live check: skipped",
		)

		fmt.Println(
			"Set STORAGE_CHECK_LIVE=true to run a temporary upload/download/delete test.",
		)

		return
	}

	fmt.Println(
		"Live check: starting",
	)

	ctx, cancel :=
		context.WithTimeout(
			context.Background(),
			liveCheckTimeout,
		)
	defer cancel()

	if err :=
		runLiveCheck(
			ctx,
			gateway,
		); err != nil {

		log.Fatalf(
			"live storage check failed: %v",
			err,
		)
	}

	fmt.Println(
		"Live check: PASS",
	)
}

func runLiveCheck(
	ctx context.Context,
	gateway *storage.Gateway,
) error {
	objectID, err :=
		randomHex(
			12,
		)
	if err != nil {
		return fmt.Errorf(
			"generate temporary object id: %w",
			err,
		)
	}

	key :=
		"private/storage-check/" +
			objectID +
			".txt"

	payload :=
		[]byte(
			"ene dei storage live check\n",
		)

	uploaded := false

	defer func() {
		if !uploaded {
			return
		}

		cleanupCtx,
			cancel :=
			context.WithTimeout(
				context.Background(),
				cleanupTimeout,
			)

		defer cancel()

		_ = gateway.Delete(
			cleanupCtx,
			key,
		)
	}()

	uploadTarget, err :=
		gateway.CreateUpload(
			ctx,
			storage.UploadRequest{
				Key: key,

				ContentType: "text/plain",

				ContentLength: int64(
					len(payload),
				),

				Access: storage.AccessPrivate,

				CacheControl: "private, no-store",
			},
		)

	if err != nil {
		return fmt.Errorf(
			"create upload target: %w",
			err,
		)
	}

	fmt.Println(
		"Upload target: created",
	)

	if err :=
		putObject(
			ctx,
			uploadTarget,
			payload,
		); err != nil {

		return err
	}

	uploaded = true

	fmt.Println(
		"Upload: PASS",
	)

	info, err :=
		gateway.Stat(
			ctx,
			key,
		)
	if err != nil {
		return fmt.Errorf(
			"stat uploaded object: %w",
			err,
		)
	}

	if info.ContentLength !=
		int64(
			len(payload),
		) {

		return fmt.Errorf(
			"stat uploaded object: expected %d bytes, got %d",
			len(payload),
			info.ContentLength,
		)
	}

	fmt.Println(
		"Stat: PASS",
	)

	downloadTarget, err :=
		gateway.CreateDownload(
			ctx,
			storage.DownloadRequest{
				Key: key,
			},
		)
	if err != nil {
		return fmt.Errorf(
			"create download target: %w",
			err,
		)
	}

	fmt.Println(
		"Download target: created",
	)

	downloaded,
		err :=
		getObject(
			ctx,
			downloadTarget,
		)

	if err != nil {
		return err
	}

	if !bytes.Equal(
		downloaded,
		payload,
	) {
		return fmt.Errorf(
			"downloaded object content does not match uploaded content",
		)
	}

	fmt.Println(
		"Download: PASS",
	)

	if err :=
		gateway.Delete(
			ctx,
			key,
		); err != nil {

		return fmt.Errorf(
			"delete temporary object: %w",
			err,
		)
	}

	uploaded = false

	fmt.Println(
		"Delete: PASS",
	)

	_, err =
		gateway.Stat(
			ctx,
			key,
		)

	if !errors.Is(
		err,
		storage.ErrNotFound,
	) {
		if err == nil {
			return fmt.Errorf(
				"deleted object still exists",
			)
		}

		return fmt.Errorf(
			"verify object deletion: %w",
			err,
		)
	}

	fmt.Println(
		"Deletion verification: PASS",
	)

	return nil
}

func putObject(
	ctx context.Context,
	target storage.UploadTarget,
	payload []byte,
) error {
	request, err :=
		http.NewRequestWithContext(
			ctx,
			target.Method,
			target.URL,
			bytes.NewReader(
				payload,
			),
		)

	if err != nil {
		return fmt.Errorf(
			"create upload request: %w",
			err,
		)
	}

	for name, value := range target.Headers {

		request.Header.Set(
			name,
			value,
		)
	}

	client :=
		&http.Client{
			Timeout: 30 * time.Second,
		}

	response, err :=
		client.Do(
			request,
		)

	if err != nil {
		return fmt.Errorf(
			"upload temporary object: %w",
			err,
		)
	}

	defer response.Body.Close()

	_,
		_ =
		io.Copy(
			io.Discard,
			response.Body,
		)

	if response.StatusCode <
		http.StatusOK ||
		response.StatusCode >=
			http.StatusMultipleChoices {

		return fmt.Errorf(
			"upload temporary object: storage returned %s",
			response.Status,
		)
	}

	return nil
}

func getObject(
	ctx context.Context,
	target storage.DownloadTarget,
) ([]byte, error) {
	request, err :=
		http.NewRequestWithContext(
			ctx,
			http.MethodGet,
			target.URL,
			nil,
		)

	if err != nil {
		return nil,
			fmt.Errorf(
				"create download request: %w",
				err,
			)
	}

	client :=
		&http.Client{
			Timeout: 30 * time.Second,
		}

	response, err :=
		client.Do(
			request,
		)

	if err != nil {
		return nil,
			fmt.Errorf(
				"download temporary object: %w",
				err,
			)
	}

	defer response.Body.Close()

	if response.StatusCode <
		http.StatusOK ||
		response.StatusCode >=
			http.StatusMultipleChoices {

		_,
			_ =
			io.Copy(
				io.Discard,
				response.Body,
			)

		return nil,
			fmt.Errorf(
				"download temporary object: storage returned %s",
				response.Status,
			)
	}

	body, err :=
		io.ReadAll(
			io.LimitReader(
				response.Body,
				1024*1024,
			),
		)

	if err != nil {
		return nil,
			fmt.Errorf(
				"read downloaded object: %w",
				err,
			)
	}

	return body, nil
}

func envEnabled(
	name string,
) bool {
	switch strings.ToLower(
		strings.TrimSpace(
			os.Getenv(
				name,
			),
		),
	) {
	case "1",
		"true",
		"yes",
		"on":

		return true

	default:
		return false
	}
}

func randomHex(
	byteLength int,
) (string, error) {
	buffer :=
		make(
			[]byte,
			byteLength,
		)

	if _, err :=
		rand.Read(
			buffer,
		); err != nil {

		return "",
			err
	}

	return hex.EncodeToString(
		buffer,
	), nil
}
