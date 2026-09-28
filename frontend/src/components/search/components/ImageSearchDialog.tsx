"use client";

import Image from "next/image";
import Link from "next/link";

import {
  type ChangeEvent,
  type FormEvent,
  useEffect,
  useRef,
  useState,
} from "react";

import {
  Icon,
} from "@/components/ui/Icon";

import type {
  ImageSearchProduct,
  ImageSearchResponse,
} from "@/lib/api/contracts/image-search";

import {
  formatMoney,
} from "@/lib/money/format";

import styles from "../css/ImageSearchDialog.module.css";

const MAX_IMAGE_BYTES =
  10 * 1024 * 1024;

const ACCEPTED_IMAGE_TYPES =
  new Set([
    "image/jpeg",
    "image/png",
    "image/webp",
    "image/gif",
  ]);

type Props = {
  open: boolean;

  onClose:
    () => void;
};

type ErrorPayload = {
  error?: {
    code?: string;
    message?: string;
  };
};

async function responseMessage(
  response:
    Response,
) {
  try {
    const payload =
      (await response.json()) as
        ErrorPayload;

    return (
      payload.error
        ?.message ??
      "Image search could not be completed."
    );
  } catch {
    return "Image search could not be completed.";
  }
}

function ResultCard({
  product,
  onNavigate,
}: {
  product:
    ImageSearchProduct;

  onNavigate:
    () => void;
}) {
  return (
    <Link
      href={`/product/${product.slug}`}
      className={
        styles.resultCard
      }
      onClick={
        onNavigate
      }
    >
      <span
        className={
          styles.resultMedia
        }
      >
        {product.primary_image_url ? (
          <Image
            src={
              product.primary_image_url
            }
            alt=""
            fill
            sizes="(max-width: 640px) 44vw, (max-width: 820px) 28vw, 180px"
            className={
              styles.resultImage
            }
          />
        ) : (
          <span
            className={
              styles.resultFallback
            }
            aria-hidden="true"
          >
            E
          </span>
        )}

        {product.match_type ? (
          <small
            className={
              styles.matchBadge
            }
          >
            {product.match_type ===
            "primary"
              ? "Visual match"
              : "Similar"}
          </small>
        ) : null}
      </span>

      <span
        className={
          styles.resultBody
        }
      >
        <strong>
          {product.name}
        </strong>

        <small>
          {[
            product.brand,
            product.category
              ?.name,
          ]
            .filter(Boolean)
            .join(" · ")}
        </small>

        <b>
          {formatMoney(
            product.price_amount,
            product.currency,
          )}
        </b>
      </span>
    </Link>
  );
}

export function ImageSearchDialog({
  open,
  onClose,
}: Props) {
  const dialogRef =
    useRef<HTMLDialogElement | null>(
      null,
    );

  const inputRef =
    useRef<HTMLInputElement | null>(
      null,
    );

  const previewURLRef =
    useRef("");

  const [
    file,
    setFile,
  ] =
    useState<File | null>(
      null,
    );

  const [
    previewURL,
    setPreviewURL,
  ] =
    useState("");

  const [
    result,
    setResult,
  ] =
    useState<ImageSearchResponse | null>(
      null,
    );

  const [
    busy,
    setBusy,
  ] =
    useState(false);

  const [
    error,
    setError,
  ] =
    useState("");

  /*
   * <dialog> is an external browser system,
   * so synchronization belongs in an effect.
   */
  useEffect(() => {
    const dialog =
      dialogRef.current;

    if (!dialog) {
      return;
    }

    if (
      open &&
      !dialog.open
    ) {
      dialog.showModal();

      window.setTimeout(
        () => {
          inputRef.current
            ?.focus();
        },
        0,
      );
    }

    if (
      !open &&
      dialog.open
    ) {
      dialog.close();
    }
  }, [open]);

  /*
   * Cleanup only. No synchronous React state
   * writes occur inside this effect.
   */
  useEffect(() => {
    return () => {
      if (
        previewURLRef.current
      ) {
        URL.revokeObjectURL(
          previewURLRef.current,
        );

        previewURLRef.current =
          "";
      }
    };
  }, []);

  function clearPreview() {
    if (
      previewURLRef.current
    ) {
      URL.revokeObjectURL(
        previewURLRef.current,
      );

      previewURLRef.current =
        "";
    }

    setPreviewURL("");
  }

  function reset() {
    clearPreview();

    setFile(null);
    setResult(null);
    setError("");
    setBusy(false);

    if (
      inputRef.current
    ) {
      inputRef.current.value =
        "";
    }
  }

  function close() {
    reset();
    onClose();
  }

  function handleFile(
    event:
      ChangeEvent<HTMLInputElement>,
  ) {
    const selected =
      event.target.files?.[0] ??
      null;

    setResult(null);
    setError("");

    clearPreview();

    if (!selected) {
      setFile(null);
      return;
    }

    if (
      !ACCEPTED_IMAGE_TYPES.has(
        selected.type,
      )
    ) {
      setFile(null);

      event.target.value =
        "";

      setError(
        "Choose a JPEG, PNG, WebP, or GIF image.",
      );

      return;
    }

    if (
      selected.size >
      MAX_IMAGE_BYTES
    ) {
      setFile(null);

      event.target.value =
        "";

      setError(
        "Image search accepts files up to 10 MB.",
      );

      return;
    }

    const nextPreviewURL =
      URL.createObjectURL(
        selected,
      );

    previewURLRef.current =
      nextPreviewURL;

    setFile(
      selected,
    );

    setPreviewURL(
      nextPreviewURL,
    );
  }

  async function submit(
    event:
      FormEvent<HTMLFormElement>,
  ) {
    event.preventDefault();

    if (
      !file ||
      busy
    ) {
      if (!file) {
        inputRef.current
          ?.focus();
      }

      return;
    }

    setBusy(true);
    setError("");
    setResult(null);

    const form =
      new FormData();

    form.append(
      "image",
      file,
      file.name,
    );

    try {
      const response =
        await fetch(
          "/api/storefront/search/image?limit=12",
          {
            method:
              "POST",

            body:
              form,

            cache:
              "no-store",
          },
        );

      if (
        !response.ok
      ) {
        throw new Error(
          await responseMessage(
            response,
          ),
        );
      }

      const payload =
        (await response.json()) as
          ImageSearchResponse;

      setResult(
        payload,
      );
    } catch (caught) {
      setError(
        caught instanceof
          Error
          ? caught.message
          : "Image search could not be completed.",
      );
    } finally {
      setBusy(false);
    }
  }

  const products =
    result?.data ??
    [];

  return (
    <dialog
      ref={
        dialogRef
      }
      className={
        styles.dialog
      }
      aria-labelledby="image-search-title"
      onCancel={(
        event,
      ) => {
        event.preventDefault();

        close();
      }}
      onClose={() => {
        if (open) {
          onClose();
        }
      }}
      onClick={(
        event,
      ) => {
        if (
          event.target ===
          dialogRef.current
        ) {
          close();
        }
      }}
    >
      <div
        className={
          styles.sheet
        }
      >
        <header
          className={
            styles.header
          }
        >
          <div>
            <span
              className={
                styles.eyebrow
              }
            >
              Visual discovery
            </span>

            <h2
              id="image-search-title"
            >
              Search with an image
            </h2>

            <p>
              Upload a product
              photo and we’ll
              look for visually
              similar products
              already in the
              catalog.
            </p>
          </div>

          <button
            type="button"
            className={
              styles.closeButton
            }
            aria-label="Close image search"
            onClick={
              close
            }
          >
            <Icon
              name="close"
              size={18}
            />
          </button>
        </header>

        <form
          className={
            styles.content
          }
          onSubmit={
            submit
          }
        >
          <div
            className={
              styles.uploadColumn
            }
          >
            <label
              className={
                styles.dropzone
              }
            >
              <input
                ref={
                  inputRef
                }
                type="file"
                accept="image/jpeg,image/png,image/webp,image/gif"
                className={
                  styles.fileInput
                }
                onChange={
                  handleFile
                }
              />

              {previewURL ? (
                /*
                 * Local Blob preview.
                 * It is intentionally
                 * not sent through the
                 * Next image optimizer.
                 */
                // eslint-disable-next-line @next/next/no-img-element
                <img
                  src={
                    previewURL
                  }
                  alt="Selected product search reference"
                  className={
                    styles.preview
                  }
                />
              ) : (
                <span
                  className={
                    styles.uploadPrompt
                  }
                >
                  <span
                    className={
                      styles.cameraIcon
                    }
                  >
                    <Icon
                      name="imageSearch"
                      size={28}
                    />
                  </span>

                  <strong>
                    Choose a
                    product photo
                  </strong>

                  <small>
                    JPEG, PNG,
                    WebP or GIF ·
                    maximum 10 MB.
                    On mobile, the
                    system picker
                    can offer the
                    camera or
                    photo library.
                  </small>
                </span>
              )}
            </label>

            <div
              className={
                styles.uploadActions
              }
            >
              <button
                type="button"
                className={
                  styles.secondaryButton
                }
                onClick={() =>
                  inputRef.current
                    ?.click()
                }
              >
                {file
                  ? "Choose another"
                  : "Choose image"}
              </button>

              <button
                type="submit"
                className={
                  styles.primaryButton
                }
                disabled={
                  !file ||
                  busy
                }
              >
                <Icon
                  name="search"
                  size={17}
                />

                {busy
                  ? "Searching…"
                  : "Find similar products"}
              </button>
            </div>

            {error ? (
              <div
                className={
                  styles.error
                }
                role="alert"
              >
                {error}
              </div>
            ) : null}
          </div>

          <section
            className={
              styles.results
            }
            aria-live="polite"
            aria-busy={
              busy
            }
          >
            <div
              className={
                styles.resultsHeader
              }
            >
              <div>
                <span
                  className={
                    styles.eyebrow
                  }
                >
                  Catalog matches
                </span>

                <h3>
                  {result
                    ? `${result.meta.total.toLocaleString()} result${
                        result
                          .meta
                          .total ===
                        1
                          ? ""
                          : "s"
                      }`
                    : "Results appear here"}
                </h3>
              </div>

              {result?.meta
                .showing_similar ? (
                <span
                  className={
                    styles.similarNote
                  }
                >
                  Related products
                  included
                </span>
              ) : null}
            </div>

            {busy ? (
              <div
                className={
                  styles.loadingState
                }
              >
                <span
                  className={
                    styles.spinner
                  }
                  aria-hidden="true"
                />

                <strong>
                  Comparing your
                  image
                </strong>

                <small>
                  Visual matching
                  runs separately
                  from the normal
                  storefront
                  rendering path.
                </small>
              </div>
            ) : null}

            {!busy &&
            result &&
            products.length ===
              0 ? (
              <div
                className={
                  styles.emptyState
                }
              >
                <strong>
                  No close visual
                  match yet
                </strong>

                <span>
                  Try a clearer
                  photo with the
                  product filling
                  most of the
                  frame.
                </span>
              </div>
            ) : null}

            {!busy &&
            !result ? (
              <div
                className={
                  styles.emptyState
                }
              >
                <strong>
                  Use a clear
                  reference image
                </strong>

                <span>
                  Product-only
                  photos usually
                  produce the most
                  useful catalog
                  matches.
                </span>
              </div>
            ) : null}

            {!busy &&
            products.length >
              0 ? (
              <div
                className={
                  styles.resultGrid
                }
              >
                {products.map(
                  (
                    product,
                  ) => (
                    <ResultCard
                      key={
                        product.id
                      }
                      product={
                        product
                      }
                      onNavigate={
                        close
                      }
                    />
                  ),
                )}
              </div>
            ) : null}
          </section>
        </form>
      </div>
    </dialog>
  );
}