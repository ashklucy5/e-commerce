package supportattachment_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	platformconfig "project.local/commerce-api/internal/platform/config"
	platformstorage "project.local/commerce-api/internal/platform/storage"
)

const runSupportAttachmentB2TestEnv = "RUN_SUPPORT_ATTACHMENT_B2_TEST"

func TestSupportAttachmentRealB2Lifecycle(
	t *testing.T,
) {
	if os.Getenv(
		runSupportAttachmentB2TestEnv,
	) != "1" {

		t.Skipf(
			"set %s=1 to run the real support attachment storage lifecycle test",
			runSupportAttachmentB2TestEnv,
		)
	}

	ctx, cancel :=
		context.WithTimeout(
			context.Background(),
			60*time.Second,
		)

	defer cancel()

	// ---------------------------------------------------------
	// Load the exact storage configuration used by production.
	// ---------------------------------------------------------

	cfg, err :=
		platformconfig.LoadStorage()
	if err != nil {
		t.Fatalf(
			"load storage configuration: %v",
			err,
		)
	}

	if strings.TrimSpace(
		cfg.Provider,
	) != "s3" {

		t.Fatalf(
			"real B2 lifecycle test requires STORAGE_PROVIDER=s3, got %q",
			cfg.Provider,
		)
	}

	gateway, err :=
		platformstorage.NewFromConfig(
			cfg,
		)
	if err != nil {
		t.Fatalf(
			"create storage gateway: %v",
			err,
		)
	}

	if gateway.ProviderName() !=
		"s3" {

		t.Fatalf(
			"storage provider = %q, want s3",
			gateway.ProviderName(),
		)
	}

	// ---------------------------------------------------------
	// Tiny valid JPEG-shaped payload.
	//
	// The test verifies storage behavior, not image decoding.
	// ---------------------------------------------------------

	payload :=
		[]byte{
			0xff,
			0xd8,
			0xff,
			0xd9,
		}

	contentType :=
		"image/jpeg"

	key :=
		fmt.Sprintf(
			"private/support/storage-check/%d.jpg",
			time.Now().
				UTC().
				UnixNano(),
		)

	/*
		Always attempt cleanup.

		This makes the test safe to rerun if an assertion fails
		after the PUT has already succeeded.
	*/
	defer func() {
		cleanupCtx,
			cleanupCancel :=
			context.WithTimeout(
				context.Background(),
				15*time.Second,
			)

		defer cleanupCancel()

		err :=
			gateway.Delete(
				cleanupCtx,
				key,
			)

		if err != nil &&
			!errors.Is(
				err,
				platformstorage.ErrNotFound,
			) {

			t.Logf(
				"cleanup warning for temporary support object: %v",
				err,
			)
		}
	}()

	// ---------------------------------------------------------
	// 1. Create private presigned PUT
	// ---------------------------------------------------------

	uploadTarget, err :=
		gateway.CreateUpload(
			ctx,
			platformstorage.UploadRequest{
				Key: key,

				ContentType: contentType,

				ContentLength: int64(
					len(
						payload,
					),
				),

				Access: platformstorage.AccessPrivate,

				Purpose: platformstorage.
					PurposeSupportImage,
			},
		)
	if err != nil {
		t.Fatalf(
			"create support image upload target: %v",
			err,
		)
	}

	if uploadTarget.Key !=
		key {

		t.Fatalf(
			"upload target key = %q, want %q",
			uploadTarget.Key,
			key,
		)
	}

	if strings.TrimSpace(
		uploadTarget.URL,
	) == "" {

		t.Fatal(
			"upload target URL is empty",
		)
	}

	method :=
		strings.TrimSpace(
			uploadTarget.Method,
		)

	if method == "" {
		method =
			http.MethodPut
	}

	if method !=
		http.MethodPut {

		t.Fatalf(
			"upload target method = %q, want PUT",
			method,
		)
	}

	// ---------------------------------------------------------
	// 2. Execute the real presigned PUT against B2
	// ---------------------------------------------------------

	putRequest, err :=
		http.NewRequestWithContext(
			ctx,
			method,
			uploadTarget.URL,
			bytes.NewReader(
				payload,
			),
		)
	if err != nil {
		t.Fatalf(
			"create presigned PUT request: %v",
			err,
		)
	}

	for name, value := range uploadTarget.Headers {

		putRequest.Header.Set(
			name,
			value,
		)
	}

	if putRequest.Header.Get(
		"Content-Type",
	) == "" {

		putRequest.Header.Set(
			"Content-Type",
			contentType,
		)
	}

	putRequest.ContentLength =
		int64(
			len(
				payload,
			),
		)

	client :=
		&http.Client{
			Timeout: 30 * time.Second,
		}

	putResponse, err :=
		client.Do(
			putRequest,
		)
	if err != nil {
		t.Fatalf(
			"execute presigned support image PUT: %v",
			err,
		)
	}

	_, _ =
		io.Copy(
			io.Discard,
			io.LimitReader(
				putResponse.Body,
				1<<20,
			),
		)

	putResponse.Body.Close()

	if putResponse.StatusCode <
		http.StatusOK ||
		putResponse.StatusCode >=
			http.StatusMultipleChoices {

		t.Fatalf(
			"presigned support image PUT returned HTTP %d",
			putResponse.StatusCode,
		)
	}

	// ---------------------------------------------------------
	// 3. Real B2 HEAD / Stat
	// ---------------------------------------------------------

	info, err :=
		gateway.Stat(
			ctx,
			key,
		)
	if err != nil {
		t.Fatalf(
			"stat uploaded support image: %v",
			err,
		)
	}

	if info.Key !=
		key {

		t.Fatalf(
			"stat key = %q, want %q",
			info.Key,
			key,
		)
	}

	if info.ContentLength !=
		int64(
			len(
				payload,
			),
		) {

		t.Fatalf(
			"stat content length = %d, want %d",
			info.ContentLength,
			len(
				payload,
			),
		)
	}

	if !sameMediaType(
		info.ContentType,
		contentType,
	) {
		t.Fatalf(
			"stat content type = %q, want %q",
			info.ContentType,
			contentType,
		)
	}

	// ---------------------------------------------------------
	// 4. Create private signed GET
	//
	// Production support downloads use five minutes.
	// ---------------------------------------------------------

	downloadTarget, err :=
		gateway.CreateDownload(
			ctx,
			platformstorage.DownloadRequest{
				Key: key,

				ExpiresIn: 5 * time.Minute,
			},
		)
	if err != nil {
		t.Fatalf(
			"create support image download target: %v",
			err,
		)
	}

	if downloadTarget.Key !=
		key {

		t.Fatalf(
			"download target key = %q, want %q",
			downloadTarget.Key,
			key,
		)
	}

	if strings.TrimSpace(
		downloadTarget.URL,
	) == "" {

		t.Fatal(
			"download target URL is empty",
		)
	}

	// ---------------------------------------------------------
	// 5. Execute real signed GET and verify exact bytes
	// ---------------------------------------------------------

	getRequest, err :=
		http.NewRequestWithContext(
			ctx,
			http.MethodGet,
			downloadTarget.URL,
			nil,
		)
	if err != nil {
		t.Fatalf(
			"create signed support image GET request: %v",
			err,
		)
	}

	getResponse, err :=
		client.Do(
			getRequest,
		)
	if err != nil {
		t.Fatalf(
			"execute signed support image GET: %v",
			err,
		)
	}

	downloaded,
		readErr :=
		io.ReadAll(
			io.LimitReader(
				getResponse.Body,
				1<<20,
			),
		)

	getResponse.Body.Close()

	if readErr != nil {
		t.Fatalf(
			"read signed support image GET body: %v",
			readErr,
		)
	}

	if getResponse.StatusCode !=
		http.StatusOK {

		t.Fatalf(
			"signed support image GET returned HTTP %d",
			getResponse.StatusCode,
		)
	}

	if !bytes.Equal(
		downloaded,
		payload,
	) {
		t.Fatalf(
			"downloaded support image bytes differ from uploaded bytes: got=%d want=%d",
			len(
				downloaded,
			),
			len(
				payload,
			),
		)
	}

	// ---------------------------------------------------------
	// 6. Real B2 DELETE
	// ---------------------------------------------------------

	if err :=
		gateway.Delete(
			ctx,
			key,
		); err != nil {

		t.Fatalf(
			"delete support image from storage: %v",
			err,
		)
	}

	// ---------------------------------------------------------
	// 7. Verify object is no longer readable through Stat
	// ---------------------------------------------------------

	_, err =
		gateway.Stat(
			ctx,
			key,
		)

	if !errors.Is(
		err,
		platformstorage.ErrNotFound,
	) {
		t.Fatalf(
			"stat after deletion error = %v, want storage.ErrNotFound",
			err,
		)
	}

	t.Log(
		"real B2 support image lifecycle passed: presigned PUT -> Stat -> signed GET -> Delete -> NotFound",
	)
}

func sameMediaType(
	actual string,
	expected string,
) bool {
	actual =
		strings.ToLower(
			strings.TrimSpace(
				strings.SplitN(
					actual,
					";",
					2,
				)[0],
			),
		)

	expected =
		strings.ToLower(
			strings.TrimSpace(
				strings.SplitN(
					expected,
					";",
					2,
				)[0],
			),
		)

	return actual ==
		expected
}
