"use client";

import Link from "next/link";

import {
  type ChangeEvent,
  type DragEvent,
  useEffect,
  useRef,
  useState,
} from "react";

import AdminPageHeader from "@/components/admin/layout/components/AdminPageHeader";

import {
  adminFetch,
  adminFetchWithStatus,
  AdminRequestError,
} from "@/lib/admin/api";

import {
  prepareCatalogImport,
  uploadCatalogPackageImages,
  validateCatalogImportFile,
} from "@/lib/admin/catalog-import-package";

import type {
  AdminCatalogImportApplyResult,
  AdminCatalogImportBatchListItem,
  AdminCatalogImportBatchListResult,
  AdminCatalogImportBatchStatus,
  AdminCatalogImportPreviewResult,
  AdminCatalogImportRowPreview,
  AdminCatalogImportStageResult,
  AdminCatalogImportValidationError,
} from "@/lib/admin/catalog-import-types";

import styles from "../css/CatalogImportWorkspace.module.css";

const PREVIEW_LIMIT = 100;
const HISTORY_LIMIT = 20;

type CatalogImportWorkspaceProps = {
  portal: string;
};

type ServerErrorPayload = {
  error?:
    | string
    | {
        message?: string;
      };

  message?: string;
};

function requestErrorMessage(
  value: unknown,
  fallback: string,
): string {
  if (
    value instanceof
    AdminRequestError
  ) {
    return value.message;
  }

  if (
    value instanceof Error
  ) {
    return value.message;
  }

  return fallback;
}

function serverPayloadMessage(
  value: unknown,
  fallback: string,
): string {
  if (
    !value ||
    typeof value !== "object"
  ) {
    return fallback;
  }

  const payload =
    value as ServerErrorPayload;

  if (
    typeof payload.error ===
    "string"
  ) {
    return payload.error;
  }

  if (
    payload.error &&
    typeof payload.error ===
      "object" &&
    typeof payload.error.message ===
      "string"
  ) {
    return payload.error.message;
  }

  if (
    typeof payload.message ===
    "string"
  ) {
    return payload.message;
  }

  return fallback;
}

function isStageResult(
  value: unknown,
): value is AdminCatalogImportStageResult {
  if (
    !value ||
    typeof value !== "object"
  ) {
    return false;
  }

  const candidate =
    value as Partial<AdminCatalogImportStageResult>;

  return Boolean(
    candidate.batch &&
      typeof candidate.batch.id ===
        "string" &&
      typeof candidate.batch.status ===
        "string" &&
      Array.isArray(
        candidate.errors,
      ),
  );
}

function validateFile(
  file: File,
): string {
  return validateCatalogImportFile(
    file,
  );
}

function formatBytes(
  bytes: number,
): string {
  if (
    bytes < 1024
  ) {
    return `${bytes} B`;
  }

  if (
    bytes <
    1024 * 1024
  ) {
    return `${(
      bytes / 1024
    ).toFixed(1)} KiB`;
  }

  return `${(
    bytes /
    (1024 * 1024)
  ).toFixed(1)} MiB`;
}

function formatDate(
  value: string,
): string {
  const date =
    new Date(value);

  if (
    Number.isNaN(
      date.getTime(),
    )
  ) {
    return value;
  }

  return new Intl.DateTimeFormat(
    undefined,
    {
      dateStyle: "medium",
      timeStyle: "short",
    },
  ).format(date);
}

function statusClass(
  status:
    AdminCatalogImportBatchStatus,
): string {
  switch (status) {
    case "ready":
      return styles.statusReady;

    case "completed":
      return styles.statusCompleted;

    case "failed":
      return styles.statusFailed;

    case "applying":
    case "parsing":
    case "validating":
      return styles.statusWorking;

    default:
      return styles.statusNeutral;
  }
}

function rowStatusClass(
  status: string,
): string {
  switch (status) {
    case "valid":
      return styles.rowValid;

    case "applied":
      return styles.rowApplied;

    case "skipped":
      return styles.rowSkipped;

    case "invalid":
    case "failed":
      return styles.rowInvalid;

    default:
      return styles.rowPending;
  }
}

function actionClass(
  action: string,
): string {
  switch (action) {
    case "create":
      return styles.actionCreate;

    case "update":
      return styles.actionUpdate;

    case "skip":
      return styles.actionSkip;

    default:
      return styles.actionNone;
  }
}

function rowReference(
  row:
    AdminCatalogImportRowPreview,
): string {
  if (
    row.product_code &&
    row.sku
  ) {
    return `${row.product_code} · ${row.sku}`;
  }

  return (
    row.sku ||
    row.product_code ||
    "—"
  );
}

function validationErrorLabel(
  error:
    AdminCatalogImportValidationError,
): string {
  const field =
    error.field
      ? `${error.field}: `
      : "";

  return `${field}${error.message}`;
}

export default function CatalogImportWorkspace({
  portal,
}: CatalogImportWorkspaceProps) {
  const fileInputRef =
    useRef<HTMLInputElement | null>(
      null,
    );

  const [
    file,
    setFile,
  ] =
    useState<File | null>(
      null,
    );

  const [
    dragActive,
    setDragActive,
  ] =
    useState(false);

  const [
    uploading,
    setUploading,
  ] =
    useState(false);

  const [
    uploadProgress,
    setUploadProgress,
  ] =
    useState("");

  const [
    applying,
    setApplying,
  ] =
    useState(false);

  const [
    confirmApply,
    setConfirmApply,
  ] =
    useState(false);

  const [
    historyLoading,
    setHistoryLoading,
  ] =
    useState(true);

  const [
    previewLoading,
    setPreviewLoading,
  ] =
    useState(false);

  const [
    history,
    setHistory,
  ] =
    useState<
      AdminCatalogImportBatchListItem[]
    >([]);

  const [
    preview,
    setPreview,
  ] =
    useState<
      AdminCatalogImportPreviewResult
        | null
    >(null);

  const [
    previewOffset,
    setPreviewOffset,
  ] =
    useState(0);

  const [
    activeFilename,
    setActiveFilename,
  ] =
    useState("");

  const [
    activeLastError,
    setActiveLastError,
  ] =
    useState("");

  const [
    stageErrors,
    setStageErrors,
  ] =
    useState<
      AdminCatalogImportValidationError[]
    >([]);

  const [
    error,
    setError,
  ] =
    useState("");

  const [
    notice,
    setNotice,
  ] =
    useState("");

  async function loadHistory() {
    setHistoryLoading(
      true,
    );

    try {
      const result =
        await adminFetch<AdminCatalogImportBatchListResult>(
          `/catalog-imports?limit=${HISTORY_LIMIT}&offset=0`,
        );

      setHistory(
        result.items ?? [],
      );
    } catch (value) {
      setError(
        requestErrorMessage(
          value,
          "Catalog import history could not be loaded.",
        ),
      );
    } finally {
      setHistoryLoading(
        false,
      );
    }
  }

  async function loadPreview(
    batchId: string,
    nextOffset = 0,
    filename?: string,
    lastError?: string,
  ) {
    setPreviewLoading(
      true,
    );

    setError("");
    setConfirmApply(false);

    try {
      const result =
        await adminFetch<AdminCatalogImportPreviewResult>(
          `/catalog-imports/${encodeURIComponent(
            batchId,
          )}?limit=${PREVIEW_LIMIT}&offset=${nextOffset}`,
        );

      setPreview(
        result,
      );

      setPreviewOffset(
        nextOffset,
      );

      if (
        typeof filename ===
        "string"
      ) {
        setActiveFilename(
          filename,
        );
      }

      if (
        typeof lastError ===
        "string"
      ) {
        setActiveLastError(
          lastError,
        );
      }
    } catch (value) {
      setError(
        requestErrorMessage(
          value,
          "The catalog import preview could not be loaded.",
        ),
      );
    } finally {
      setPreviewLoading(
        false,
      );
    }
  }

  useEffect(() => {
    let cancelled =
      false;

    void Promise.resolve().then(
      () => {
        if (
          !cancelled
        ) {
          void loadHistory();
        }
      },
    );

    return () => {
      cancelled =
        true;
    };
  }, []);

  function chooseFile(
    nextFile:
      File | null,
  ) {
    setNotice("");
    setError("");
    setUploadProgress("");

    if (
      !nextFile
    ) {
      setFile(
        null,
      );

      return;
    }

    const fileError =
      validateFile(
        nextFile,
      );

    if (
      fileError
    ) {
      setFile(
        null,
      );

      setError(
        fileError,
      );

      if (
        fileInputRef.current
      ) {
        fileInputRef.current.value =
          "";
      }

      return;
    }

    setFile(
      nextFile,
    );
  }

  function handleFileInput(
    event:
      ChangeEvent<HTMLInputElement>,
  ) {
    chooseFile(
      event.target.files?.[0] ??
        null,
    );
  }

  function handleDragOver(
    event:
      DragEvent<HTMLDivElement>,
  ) {
    event.preventDefault();

    event.dataTransfer.dropEffect =
      "copy";

    setDragActive(
      true,
    );
  }

  function handleDragLeave(
    event:
      DragEvent<HTMLDivElement>,
  ) {
    event.preventDefault();

    setDragActive(
      false,
    );
  }

  function handleDrop(
    event:
      DragEvent<HTMLDivElement>,
  ) {
    event.preventDefault();

    setDragActive(
      false,
    );

    chooseFile(
      event.dataTransfer.files?.[0] ??
        null,
    );
  }

  async function stageWorkbook() {
    if (
      !file ||
      uploading
    ) {
      return;
    }

    const fileError =
      validateFile(
        file,
      );

    if (
      fileError
    ) {
      setError(
        fileError,
      );

      return;
    }

    setUploading(
      true,
    );

    setError("");
    setNotice("");
    setStageErrors([]);
    setConfirmApply(false);
    setUploadProgress("");

    try {
      const isZip =
        file.name
          .toLowerCase()
          .endsWith(
            ".zip",
          );

      setUploadProgress(
        isZip
          ? "Opening catalog package…"
          : "Preparing workbook…",
      );

      /*
       * XLSX:
       *   used directly.
       *
       * ZIP:
       *   unpacked locally in the browser.
       *
       * The ZIP itself never needs to pass
       * through the commerce API.
       */
      const prepared =
        await prepareCatalogImport(
          file,
        );

      let assetMap:
        Record<
          string,
          string
        > = {};

      if (
        prepared.isPackage
      ) {
        setUploadProgress(
          `Preparing ${prepared.images.length} product image${prepared.images.length === 1 ? "" : "s"} for storage…`,
        );

        /*
         * Images are uploaded directly from
         * the Admin browser using the existing
         * presigned product-image upload flow.
         */
        assetMap =
          await uploadCatalogPackageImages(
            prepared,
            progress => {
              setUploadProgress(
                `Uploading images ${progress.completed}/${progress.total} · ${progress.filename}`,
              );
            },
          );
      }

      setUploadProgress(
        "Staging workbook & validating catalog…",
      );

      const formData =
        new FormData();

      formData.append(
        "file",
        prepared.workbook,
        prepared.workbook.name,
      );

      /*
       * ZIP packages provide a mapping:
       *
       * filename.webp
       *      ↓
       * public/products/images/uploads/...
       *
       * The backend resolves these keys to
       * permanent public/CDN image URLs.
       */
      if (
        Object.keys(
          assetMap,
        ).length > 0
      ) {
        formData.append(
          "asset_map",
          JSON.stringify(
            assetMap,
          ),
        );
      }

      const result =
        await adminFetchWithStatus<unknown>(
          "/catalog-imports",
          {
            method:
              "POST",

            body:
              formData,
          },
          {
            /*
             * Validation failures are returned
             * as HTTP 422 while still including
             * the created staging batch.
             */
            acceptedStatuses: [
              422,
            ],
          },
        );

      if (
        !isStageResult(
          result.payload,
        )
      ) {
        throw new Error(
          serverPayloadMessage(
            result.payload,
            "The catalog import could not be staged.",
          ),
        );
      }

      const stageResult =
        result.payload;

      setStageErrors(
        stageResult.errors ??
          [],
      );

      /*
       * Keep displaying the package filename
       * for the current import session rather
       * than only the XLSX filename contained
       * inside the package.
       */
      setActiveFilename(
        prepared.sourceFilename,
      );

      setActiveLastError(
        "",
      );

      if (
        stageResult.batch.status ===
        "ready"
      ) {
        if (
          prepared.isPackage
        ) {
          setNotice(
            `${prepared.images.length} product image${prepared.images.length === 1 ? "" : "s"} uploaded to storage. Catalog validation passed. Review the action plan before applying.`,
          );
        } else {
          setNotice(
            "Validation passed. Nothing has changed in the live catalog yet; review the action plan before applying.",
          );
        }
      } else {
        setNotice(
          prepared.isPackage
            ? "The images were uploaded to storage, but catalog validation found problems. The live catalog was not changed. Correct the package and stage a new batch."
            : "Validation found problems. The live catalog was not changed. Correct the workbook and upload a new batch.",
        );
      }

      await loadPreview(
        stageResult.batch.id,
        0,
        prepared.sourceFilename,
        stageResult.batch.status ===
          "failed"
          ? `Validation failed with ${stageResult.errors.length} error${stageResult.errors.length === 1 ? "" : "s"}.`
          : "",
      );

      await loadHistory();

      setFile(
        null,
      );

      if (
        fileInputRef.current
      ) {
        fileInputRef.current.value =
          "";
      }
    } catch (value) {
      setError(
        requestErrorMessage(
          value,
          "The catalog import could not be staged.",
        ),
      );

      await loadHistory();
    } finally {
      setUploading(
        false,
      );

      setUploadProgress(
        "",
      );
    }
  }

  async function applyBatch() {
    if (
      !preview ||
      applying ||
      preview.batch.status !==
        "ready" ||
      preview.batch.failed_rows !==
        0
    ) {
      return;
    }

    setApplying(
      true,
    );

    setError("");
    setNotice("");

    try {
      const result =
        await adminFetch<AdminCatalogImportApplyResult>(
          `/catalog-imports/${encodeURIComponent(
            preview.batch.id,
          )}/apply`,
          {
            method:
              "POST",
          },
        );

      setNotice(
        `Catalog import completed: ${result.applied_rows} row${result.applied_rows === 1 ? "" : "s"} applied and ${result.skipped_rows} skipped.`,
      );

      setConfirmApply(
        false,
      );

      await loadPreview(
        preview.batch.id,
        previewOffset,
        activeFilename,
        "",
      );

      await loadHistory();
    } catch (value) {
      setError(
        requestErrorMessage(
          value,
          "The catalog import could not be applied.",
        ),
      );
    } finally {
      setApplying(
        false,
      );
    }
  }

  const batch =
    preview?.batch ??
    null;

  const canApply =
    Boolean(
      batch &&
        batch.status ===
          "ready" &&
        batch.failed_rows ===
          0 &&
        !applying,
    );

  const previewFirst =
    preview &&
    preview.rows.length >
      0
      ? previewOffset +
        1
      : 0;

  const previewLast =
    preview
      ? previewOffset +
        preview.rows.length
      : 0;

  const canPreviewPrevious =
    Boolean(
      preview &&
        previewOffset >
          0 &&
        !previewLoading,
    );

  const canPreviewNext =
    Boolean(
      preview &&
        previewLast <
          preview.batch
            .total_rows &&
        !previewLoading,
    );

  return (
    <div
      className={
        styles.page
      }
    >
      <div
        className={
          styles.pageTop
        }
      >
        <AdminPageHeader
          eyebrow="Catalog operations"
          title="Excel catalog import"
          description="Stage a simple or advanced XLSX workbook or an XLSX + product-images ZIP package, inspect every planned create/update/skip action, then explicitly apply only a clean validated batch."
        />

        <div
          className={
            styles.topActions
          }
        >
          <Link
            href={`/${portal}/products`}
            className={
              styles.backLink
            }
          >
            <span
              aria-hidden="true"
            >
              ←
            </span>

            Products
          </Link>
        </div>
      </div>

      <div
        className={
          styles.safetyBanner
        }
      >
        <span
          className={
            styles.safetyIcon
          }
          aria-hidden="true"
        >
          ◈
        </span>

        <div>
          <strong>
            Staging does not
            publish catalog
            changes
          </strong>

          <p>
            XLSX files are
            validated first.
            Catalog ZIP images
            are uploaded to
            object storage during
            preparation, but
            products, variants
            and image records
            change only after a
            ready batch is
            explicitly applied.
          </p>
        </div>
      </div>

      {error ? (
        <div
          className={
            styles.errorBanner
          }
          role="alert"
        >
          <div>
            <strong>
              Import request
              failed
            </strong>

            <span>
              {error}
            </span>
          </div>

          <button
            type="button"
            onClick={() =>
              setError("")
            }
          >
            Dismiss
          </button>
        </div>
      ) : null}

      {notice ? (
        <div
          className={
            styles.noticeBanner
          }
          role="status"
          aria-live="polite"
        >
          <span
            aria-hidden="true"
          >
            ✓
          </span>

          {notice}
        </div>
      ) : null}

      <div
        className={
          styles.primaryGrid
        }
      >
        <section
          className={
            styles.uploadCard
          }
          aria-labelledby="catalog-import-upload"
        >
          <div
            className={
              styles.sectionHeading
            }
          >
            <div>
              <span>
                Step 1
              </span>

              <h2
                id="catalog-import-upload"
              >
                Stage catalog
              </h2>
            </div>

            <span
              className={
                styles.fileLimit
              }
            >
              XLSX 20 MiB · ZIP
              512 MiB
            </span>
          </div>

          <div
            className={`${styles.dropZone} ${
              dragActive
                ? styles.dropZoneActive
                : ""
            }`}
            onDragOver={
              handleDragOver
            }
            onDragLeave={
              handleDragLeave
            }
            onDrop={
              handleDrop
            }
          >
            <input
              ref={
                fileInputRef
              }
              type="file"
              accept=".xlsx,.zip,application/vnd.openxmlformats-officedocument.spreadsheetml.sheet,application/zip"
              className={
                styles.fileInput
              }
              onChange={
                handleFileInput
              }
            />

            <button
              type="button"
              className={
                styles.filePickerButton
              }
              onClick={() =>
                fileInputRef.current?.click()
              }
              disabled={
                uploading
              }
            >
              <span
                className={
                  styles.uploadGlyph
                }
                aria-hidden="true"
              >
                ⇧
              </span>

              Choose XLSX or
              catalog ZIP
            </button>

            <p>
              or drag and drop
              one .xlsx or .zip
              file here
            </p>

            {file ? (
              <div
                className={
                  styles.selectedFile
                }
              >
                <div>
                  <strong>
                    {file.name}
                  </strong>

                  <span>
                    {formatBytes(
                      file.size,
                    )}
                  </span>
                </div>

                <button
                  type="button"
                  onClick={() =>
                    chooseFile(
                      null,
                    )
                  }
                  disabled={
                    uploading
                  }
                >
                  Remove
                </button>
              </div>
            ) : null}
          </div>

          <button
            type="button"
            className={
              styles.stageButton
            }
            disabled={
              !file ||
              uploading
            }
            onClick={() =>
              void stageWorkbook()
            }
          >
            {uploading ? (
              <>
                <span
                  className={
                    styles.spinner
                  }
                  aria-hidden="true"
                />

                {uploadProgress ||
                  "Staging & validating…"}
              </>
            ) : (
              <>
                Stage & validate
                catalog

                <span
                  aria-hidden="true"
                >
                  →
                </span>
              </>
            )}
          </button>
        </section>

        <section
          className={
            styles.contractCard
          }
          aria-labelledby="catalog-import-contract"
        >
          <div
            className={
              styles.sectionHeading
            }
          >
            <div>
              <span>
                Workbook contract
              </span>

              <h2
                id="catalog-import-contract"
              >
                Supported workbook
                formats
              </h2>
            </div>
          </div>

          <div
            className={
              styles.sheetList
            }
          >
            <div
              className={
                styles.sheetItem
              }
            >
              <span
                className={
                  styles.requiredTag
                }
              >
                Simple
              </span>

              <strong>
                Products sheet only
              </strong>

              <p>
                Product Name ·
                Category ·
                Subcategory ·
                Colors · Sizes ·
                MOQ · Price ·
                Opening Stock ·
                Image 1–6
              </p>
            </div>

            <div
              className={
                styles.sheetItem
              }
            >
              <span
                className={
                  styles.requiredTag
                }
              >
                Package
              </span>

              <strong>
                XLSX + product
                images ZIP
              </strong>

              <p>
                One XLSX workbook
                plus an images/
                folder. Image 1–6
                cells contain
                matching image
                filenames such as
                product-name-01.webp.
              </p>
            </div>

            <div
              className={
                styles.sheetItem
              }
            >
              <span
                className={
                  styles.requiredTag
                }
              >
                Advanced
              </span>

              <strong>
                Products + Variants
              </strong>

              <p>
                Products:
                product_code ·
                product_name ·
                category_path ·
                Variants:
                product_code ·
                sku · price
              </p>
            </div>

            <div
              className={
                styles.optionalSheets
              }
            >
              <span>
                Advanced optional
                sheets
              </span>

              <strong>
                Images · PriceTiers
                · Categories
              </strong>
            </div>
          </div>

          <div
            className={
              styles.policyNote
            }
          >
            <span
              aria-hidden="true"
            >
              !
            </span>

            <p>
              <strong>
                Quantity rule:
              </strong>{" "}

              <code>
                order_increment
              </code>{" "}

              must be{" "}

              <code>
                1
              </code>
              . MOQ controls the
              minimum; quantities
              above MOQ are
              unrestricted whole
              numbers.
            </p>
          </div>
        </section>
      </div>

      <section
        className={
          styles.previewCard
        }
        aria-labelledby="catalog-import-preview"
      >
        <div
          className={
            styles.previewHeader
          }
        >
          <div>
            <span>
              Step 2
            </span>

            <h2
              id="catalog-import-preview"
            >
              Validation & action
              preview
            </h2>

            <p>
              {batch
                ? activeFilename ||
                  `Batch ${batch.id}`
                : "Stage a workbook or open a previous batch to inspect the backend action plan."}
            </p>
          </div>

          {batch ? (
            <span
              className={`${styles.batchStatus} ${statusClass(
                batch.status,
              )}`}
            >
              {batch.status}
            </span>
          ) : null}
        </div>

        {previewLoading &&
        !preview ? (
          <div
            className={
              styles.previewEmpty
            }
          >
            <span
              className={
                styles.spinner
              }
              aria-hidden="true"
            />

            <strong>
              Loading import
              preview
            </strong>
          </div>
        ) : !batch ||
          !preview ? (
          <div
            className={
              styles.previewEmpty
            }
          >
            <span
              className={
                styles.previewEmptyGlyph
              }
              aria-hidden="true"
            >
              ▦
            </span>

            <strong>
              No batch selected
            </strong>

            <p>
              A staged workbook
              will appear here with
              row-level validation
              and create/update/skip
              actions.
            </p>
          </div>
        ) : (
          <>
            <div
              className={
                styles.metricGrid
              }
            >
              <div
                className={
                  styles.metric
                }
              >
                <span>
                  Total rows
                </span>

                <strong>
                  {
                    batch.total_rows
                  }
                </strong>
              </div>

              <div
                className={
                  styles.metric
                }
              >
                <span>
                  Valid
                </span>

                <strong>
                  {
                    batch.valid_rows
                  }
                </strong>
              </div>

              <div
                className={`${styles.metric} ${
                  batch.failed_rows >
                  0
                    ? styles.metricDanger
                    : ""
                }`}
              >
                <span>
                  Failed
                </span>

                <strong>
                  {
                    batch.failed_rows
                  }
                </strong>
              </div>

              <div
                className={
                  styles.metric
                }
              >
                <span>
                  Products
                </span>

                <strong>
                  +
                  {
                    batch.created_products
                  }

                  <small>
                    {" "}/{" "}
                    {
                      batch.updated_products
                    }{" "}
                    update
                  </small>
                </strong>
              </div>

              <div
                className={
                  styles.metric
                }
              >
                <span>
                  Variants
                </span>

                <strong>
                  +
                  {
                    batch.created_variants
                  }

                  <small>
                    {" "}/{" "}
                    {
                      batch.updated_variants
                    }{" "}
                    update
                  </small>
                </strong>
              </div>

              <div
                className={
                  styles.metric
                }
              >
                <span>
                  Categories
                </span>

                <strong>
                  +
                  {
                    batch.created_categories
                  }
                </strong>
              </div>
            </div>

            {activeLastError ? (
              <div
                className={
                  styles.batchError
                }
              >
                <strong>
                  Batch note
                </strong>

                <span>
                  {
                    activeLastError
                  }
                </span>
              </div>
            ) : null}

            {stageErrors.length >
            0 ? (
              <div
                className={
                  styles.validationSummary
                }
              >
                <strong>
                  {
                    stageErrors.length
                  }{" "}
                  validation error
                  {stageErrors.length ===
                  1
                    ? ""
                    : "s"}{" "}
                  returned by the
                  backend
                </strong>

                <div>
                  {stageErrors
                    .slice(
                      0,
                      5,
                    )
                    .map(
                      (
                        item,
                        index,
                      ) => (
                        <span
                          key={`${item.sheet}-${item.row}-${item.code}-${index}`}
                        >
                          {
                            item.sheet
                          }{" "}
                          row{" "}
                          {
                            item.row
                          }
                          :{" "}
                          {validationErrorLabel(
                            item,
                          )}
                        </span>
                      ),
                    )}

                  {stageErrors.length >
                  5 ? (
                    <span>
                      +
                      {stageErrors.length -
                        5}{" "}
                      more — inspect
                      the row table
                      below.
                    </span>
                  ) : null}
                </div>
              </div>
            ) : null}

            <div
              className={
                styles.tableWrap
              }
            >
              <div
                className={
                  styles.previewTableHeader
                }
              >
                <span>
                  Source
                </span>

                <span>
                  Reference
                </span>

                <span>
                  Action
                </span>

                <span>
                  Status
                </span>

                <span>
                  Validation
                </span>
              </div>

              <div
                className={
                  styles.previewRows
                }
                aria-busy={
                  previewLoading
                }
              >
                {preview.rows.length ===
                0 ? (
                  <div
                    className={
                      styles.noRows
                    }
                  >
                    No staged rows
                    are available for
                    this batch.
                  </div>
                ) : (
                  preview.rows.map(
                    row => (
                      <div
                        key={
                          row.id
                        }
                        className={
                          styles.previewRow
                        }
                      >
                        <span
                          className={
                            styles.sourceCell
                          }
                          data-label="Source"
                        >
                          <strong>
                            {
                              row.sheet_name
                            }
                          </strong>

                          <small>
                            Row{" "}
                            {
                              row.row_number
                            }
                          </small>
                        </span>

                        <span
                          className={
                            styles.referenceCell
                          }
                          data-label="Reference"
                        >
                          {rowReference(
                            row,
                          )}
                        </span>

                        <span
                          data-label="Action"
                        >
                          <span
                            className={`${styles.actionBadge} ${actionClass(
                              row.action,
                            )}`}
                          >
                            {row.action ||
                              "—"}
                          </span>
                        </span>

                        <span
                          data-label="Status"
                        >
                          <span
                            className={`${styles.rowStatus} ${rowStatusClass(
                              row.status,
                            )}`}
                          >
                            {
                              row.status
                            }
                          </span>
                        </span>

                        <span
                          className={
                            styles.validationCell
                          }
                          data-label="Validation"
                        >
                          {row.errors &&
                          row.errors
                            .length >
                            0 ? (
                            row.errors.map(
                              (
                                item,
                                index,
                              ) => (
                                <span
                                  key={`${item.code}-${index}`}
                                >
                                  {validationErrorLabel(
                                    item,
                                  )}
                                </span>
                              ),
                            )
                          ) : (
                            <span
                              className={
                                styles.noValidationError
                              }
                            >
                              Clear
                            </span>
                          )}
                        </span>
                      </div>
                    ),
                  )
                )}
              </div>
            </div>

            <div
              className={
                styles.previewFooter
              }
            >
              <div
                className={
                  styles.previewPagination
                }
              >
                <button
                  type="button"
                  disabled={
                    !canPreviewPrevious
                  }
                  onClick={() =>
                    void loadPreview(
                      batch.id,
                      Math.max(
                        0,
                        previewOffset -
                          PREVIEW_LIMIT,
                      ),
                    )
                  }
                >
                  ← Previous
                </button>

                <span>
                  {previewFirst}–
                  {previewLast} of{" "}
                  {
                    batch.total_rows
                  }
                </span>

                <button
                  type="button"
                  disabled={
                    !canPreviewNext
                  }
                  onClick={() =>
                    void loadPreview(
                      batch.id,
                      previewOffset +
                        PREVIEW_LIMIT,
                    )
                  }
                >
                  Next →
                </button>
              </div>

              <div
                className={
                  styles.applyArea
                }
              >
                {batch.status ===
                "completed" ? (
                  <Link
                    href={`/${portal}/products`}
                    className={
                      styles.viewProductsLink
                    }
                  >
                    View refreshed
                    products

                    <span
                      aria-hidden="true"
                    >
                      →
                    </span>
                  </Link>
                ) : (
                  <button
                    type="button"
                    className={
                      styles.applyButton
                    }
                    disabled={
                      !canApply
                    }
                    onClick={() =>
                      setConfirmApply(
                        true,
                      )
                    }
                  >
                    Apply to live
                    catalog
                  </button>
                )}
              </div>
            </div>

            {batch.status ===
            "failed" ? (
              <p
                className={
                  styles.applyHintDanger
                }
              >
                Apply is blocked
                because this batch
                contains invalid
                rows. Fix the
                workbook and stage
                a new batch.
              </p>
            ) : batch.status ===
              "ready" ? (
              <p
                className={
                  styles.applyHint
                }
              >
                Ready means
                validation passed;
                no live catalog
                changes occur until
                you confirm Apply.
                ZIP image files have
                already reached
                object storage, but
                no product-image
                records are live
                yet.
              </p>
            ) : null}

            {confirmApply &&
            canApply ? (
              <div
                className={
                  styles.confirmPanel
                }
                role="alertdialog"
                aria-labelledby="confirm-import-title"
              >
                <div>
                  <strong
                    id="confirm-import-title"
                  >
                    Apply this
                    validated batch?
                  </strong>

                  <p>
                    This will write
                    the planned
                    create/update
                    actions to the
                    live catalog.
                    Product images
                    from a ZIP
                    package will now
                    be attached using
                    their permanent
                    storage URLs.
                  </p>
                </div>

                <div>
                  <button
                    type="button"
                    className={
                      styles.cancelApplyButton
                    }
                    onClick={() =>
                      setConfirmApply(
                        false,
                      )
                    }
                    disabled={
                      applying
                    }
                  >
                    Cancel
                  </button>

                  <button
                    type="button"
                    className={
                      styles.confirmApplyButton
                    }
                    onClick={() =>
                      void applyBatch()
                    }
                    disabled={
                      applying
                    }
                  >
                    {applying ? (
                      <>
                        <span
                          className={
                            styles.spinner
                          }
                          aria-hidden="true"
                        />

                        Applying…
                      </>
                    ) : (
                      "Confirm apply"
                    )}
                  </button>
                </div>
              </div>
            ) : null}
          </>
        )}
      </section>

      <section
        className={
          styles.historyCard
        }
        aria-labelledby="catalog-import-history"
      >
        <div
          className={
            styles.historyHeader
          }
        >
          <div>
            <span>
              Recent activity
            </span>

            <h2
              id="catalog-import-history"
            >
              Import history
            </h2>
          </div>

          <button
            type="button"
            className={
              styles.refreshHistoryButton
            }
            onClick={() =>
              void loadHistory()
            }
            disabled={
              historyLoading
            }
          >
            {historyLoading
              ? "Refreshing…"
              : "Refresh"}
          </button>
        </div>

        {historyLoading &&
        history.length ===
          0 ? (
          <div
            className={
              styles.historyEmpty
            }
          >
            <span
              className={
                styles.spinner
              }
              aria-hidden="true"
            />

            Loading recent
            imports…
          </div>
        ) : history.length ===
          0 ? (
          <div
            className={
              styles.historyEmpty
            }
          >
            No catalog import
            batches yet.
          </div>
        ) : (
          <div
            className={
              styles.historyList
            }
          >
            {history.map(
              item => (
                <button
                  type="button"
                  key={
                    item.id
                  }
                  className={`${styles.historyRow} ${
                    preview?.batch.id ===
                    item.id
                      ? styles.historyRowActive
                      : ""
                  }`}
                  onClick={() => {
                    setStageErrors(
                      [],
                    );

                    setNotice("");

                    void loadPreview(
                      item.id,
                      0,
                      item.source_filename,
                      item.last_error ??
                        "",
                    );
                  }}
                >
                  <span
                    className={
                      styles.historyIdentity
                    }
                  >
                    <strong>
                      {
                        item.source_filename
                      }
                    </strong>

                    <small>
                      {formatDate(
                        item.created_at,
                      )}
                    </small>
                  </span>

                  <span
                    className={
                      styles.historyCounts
                    }
                  >
                    <strong>
                      {
                        item.total_rows
                      }
                    </strong>

                    <small>
                      rows
                    </small>
                  </span>

                  <span
                    className={
                      styles.historyCounts
                    }
                  >
                    <strong>
                      {
                        item.failed_rows
                      }
                    </strong>

                    <small>
                      failed
                    </small>
                  </span>

                  <span
                    className={`${styles.batchStatus} ${statusClass(
                      item.status,
                    )}`}
                  >
                    {
                      item.status
                    }
                  </span>

                  <span
                    className={
                      styles.historyOpen
                    }
                  >
                    Open →
                  </span>
                </button>
              ),
            )}
          </div>
        )}
      </section>
    </div>
  );
}