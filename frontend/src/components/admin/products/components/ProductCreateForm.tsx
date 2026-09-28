"use client";

import {
  type FormEvent,
  useState,
} from "react";

import {
  adminFetch,
  AdminRequestError,
} from "@/lib/admin/api";

import type {
  AdminAttachProductImageRequest,
  AdminCreateProductResponse,
  AdminCreateUploadResponse,
} from "@/lib/admin/types";

import AdminPageHeader from "@/components/admin/layout/components/AdminPageHeader";

import ProductIdentitySection, {
  type ProductIdentityValue,
} from "./ProductIdentitySection";

import ProductMediaUploader, {
  type ProductMediaItem,
} from "./ProductMediaUploader";

import ProductPublishPanel from "./ProductPublishPanel";

import ProductVariantList, {
  createEmptyVariant,
} from "./ProductVariantList";

import type {
  ProductVariantValue,
} from "./ProductVariantCard";

import styles from "../css/ProductCreateForm.module.css";

const initialIdentity: ProductIdentityValue = {
  name: "",
  categoryId: "",
  brand: "Ene dei",
  shortDescription: "",
  description: "",
};

type MediaUploadCheckpoint = {
  storageKey: string;
  attached: boolean;
};

type MediaUploadCheckpoints =
  Record<
    number,
    MediaUploadCheckpoint
  >;

function requestErrorMessage(
  value: unknown,
): string {
  if (
    value instanceof
    AdminRequestError
  ) {
    return value.message;
  }

  if (value instanceof Error) {
    return value.message;
  }

  return "The product could not be saved.";
}

function bdtToMinorUnits(
  rawValue: string,
): number {
  const normalized =
    rawValue
      .replace(/,/g, "")
      .trim();

  if (
    !/^\d+(?:\.\d{0,2})?$/.test(
      normalized,
    )
  ) {
    throw new Error(
      "Enter a valid selling price, for example 1250 or 1250.50.",
    );
  }

  const [
    wholePart,
    fractionalPart = "",
  ] = normalized.split(".");

  const whole =
    Number(wholePart);

  const fractional =
    Number(
      (
        fractionalPart + "00"
      ).slice(0, 2),
    );

  if (
    !Number.isSafeInteger(whole) ||
    whole < 0
  ) {
    throw new Error(
      "Selling price is outside the supported range.",
    );
  }

  const minor =
    whole * 100 +
    fractional;

  if (
    !Number.isSafeInteger(minor)
  ) {
    throw new Error(
      "Selling price is outside the supported range.",
    );
  }

  if (minor <= 0) {
    throw new Error(
      "Selling price must be greater than zero.",
    );
  }

  return minor;
}

function validateVariants(
  variants: ProductVariantValue[],
) {
  if (variants.length === 0) {
    throw new Error(
      "At least one product variant is required.",
    );
  }

  for (
    let index = 0;
    index < variants.length;
    index += 1
  ) {
    const variant =
      variants[index];

    if (
      !Number.isInteger(
        variant.minimumOrderQuantity,
      ) ||
      variant.minimumOrderQuantity <
        1
    ) {
      throw new Error(
        `Variant ${
          index + 1
        }: minimum order quantity must be at least 1.`,
      );
    }

    if (
      !Number.isInteger(
        variant.stock,
      ) ||
      variant.stock < 0
    ) {
      throw new Error(
        `Variant ${
          index + 1
        }: opening stock must be zero or greater.`,
      );
    }

    bdtToMinorUnits(
      variant.priceBdt,
    );
  }
}

export default function ProductCreateForm() {
  const [
    identity,
    setIdentity,
  ] =
    useState<ProductIdentityValue>(
      initialIdentity,
    );

  const [
    variants,
    setVariants,
  ] =
    useState<
      ProductVariantValue[]
    >([
      createEmptyVariant(),
    ]);

  const [
    media,
    setMedia,
  ] =
    useState<ProductMediaItem[]>(
      [],
    );

  const [
    status,
    setStatus,
  ] = useState("draft");

  const [
    busy,
    setBusy,
  ] = useState(false);

  const [
    progress,
    setProgress,
  ] = useState("");

  const [
    error,
    setError,
  ] = useState("");

  const [
    createdCode,
    setCreatedCode,
  ] = useState("");

  const [
    createdProductId,
    setCreatedProductId,
  ] = useState("");

  const [
    mediaUploadCheckpoints,
    setMediaUploadCheckpoints,
  ] =
    useState<MediaUploadCheckpoints>(
      {},
    );

  const [
    imageRetryNeeded,
    setImageRetryNeeded,
  ] = useState(false);

  async function uploadMedia(
    productId: string,
    existingCheckpoints:
      MediaUploadCheckpoints,
  ): Promise<MediaUploadCheckpoints> {
    const checkpoints:
      MediaUploadCheckpoints = {
        ...existingCheckpoints,
      };

    for (
      let index = 0;
      index < media.length;
      index += 1
    ) {
      const item =
        media[index];

      const existing =
        checkpoints[index];

      /*
       * This image already completed
       * successfully on a previous run.
       *
       * Never upload or attach it again.
       */
      if (existing?.attached) {
        continue;
      }

      let storageKey =
        existing?.storageKey ??
        "";

      /*
       * Only request a new signed URL
       * when this image has not already
       * reached object storage.
       *
       * If the MinIO PUT succeeded but
       * product attachment failed,
       * storageKey is preserved and the
       * retry skips the duplicate PUT.
       */
      if (!storageKey) {
        setProgress(
          `Preparing image ${
            index + 1
          } of ${
            media.length
          }…`,
        );

        const signed =
          await adminFetch<AdminCreateUploadResponse>(
            "/uploads",
            {
              method:
                "POST",

              body:
                JSON.stringify({
                  purpose:
                    "product_image",

                  filename:
                    item.file
                      .name,

                  content_type:
                    item.file
                      .type,

                  content_length:
                    item.file
                      .size,
                }),
            },
          );

        /*
         * Current backend contract:
         *
         * {
         *   filename,
         *   purpose,
         *   upload: {
         *     provider,
         *     key,
         *     method,
         *     url,
         *     headers,
         *     expires_at
         *   }
         * }
         */
        const upload =
          signed.upload;

        if (
          !upload?.url ||
          !upload?.key
        ) {
          throw new Error(
            "The upload service returned an invalid upload target.",
          );
        }

        setProgress(
          `Uploading image ${
            index + 1
          } of ${
            media.length
          }…`,
        );

        const uploadResponse =
          await fetch(
            upload.url,
            {
              method:
                upload.method ||
                "PUT",

              headers: {
                ...(
                  upload.headers ??
                  {}
                ),
              },

              body:
                item.file,
            },
          );

        if (
          !uploadResponse.ok
        ) {
          throw new Error(
            `Image upload failed for ${item.file.name} (${uploadResponse.status}).`,
          );
        }

        storageKey =
          upload.key;

        /*
         * Important recovery checkpoint:
         *
         * The file now exists in MinIO.
         * Persist that fact immediately
         * before attempting attachment.
         */
        checkpoints[index] = {
          storageKey,
          attached: false,
        };

        setMediaUploadCheckpoints(
          {
            ...checkpoints,
          },
        );
      }

      const attachRequest:
        AdminAttachProductImageRequest =
        {
          storage_key:
            storageKey,

          /*
           * Images currently belong to
           * the product itself.
           *
           * We do not invent a
           * variant-image association.
           */
          variant_id:
            "",

          alt_text:
            item.altText
              .trim() ||
            identity.name
              .trim(),

          sort_order:
            index,

          is_primary:
            item.isPrimary,
        };

      setProgress(
        `Attaching image ${
          index + 1
        } of ${
          media.length
        }…`,
      );

      await adminFetch<unknown>(
        `/products/${productId}/images`,
        {
          method:
            "POST",

          body:
            JSON.stringify(
              attachRequest,
            ),
        },
      );

      /*
       * Attachment succeeded.
       *
       * A future retry must completely
       * skip this image.
       */
      checkpoints[index] = {
        storageKey,
        attached: true,
      };

      setMediaUploadCheckpoints(
        {
          ...checkpoints,
        },
      );
    }

    return checkpoints;
  }

  async function handleSubmit(
    event:
      FormEvent<HTMLFormElement>,
  ) {
    event.preventDefault();

    if (
      busy ||
      createdCode
    ) {
      return;
    }

    setError("");
    setProgress("");
    setImageRetryNeeded(
      false,
    );

    setMediaUploadCheckpoints(
      {},
    );

    const productName =
      identity.name.trim();

    if (!productName) {
      setError(
        "Product name is required.",
      );

      return;
    }

    if (
      !identity.categoryId
    ) {
      setError(
        "Choose a category that accepts products.",
      );

      return;
    }

    try {
      validateVariants(
        variants,
      );
    } catch (value) {
      setError(
        requestErrorMessage(
          value,
        ),
      );

      return;
    }

    setBusy(true);

    setProgress(
      "Creating product…",
    );

    let createdProduct:
      | AdminCreateProductResponse["data"]
      | null = null;

    try {
      const created =
        await adminFetch<AdminCreateProductResponse>(
          "/products",
          {
            method:
              "POST",

            body:
              JSON.stringify({
                name:
                  productName,

                category_id:
                  identity.categoryId,

                brand:
                  identity.brand
                    .trim(),

                short_description:
                  identity
                    .shortDescription
                    .trim(),

                description:
                  identity.description
                    .trim(),

                status,

                is_featured:
                  false,

                variants:
                  variants.map(
                    (
                      variant,
                    ) => ({
                      /*
                       * Generated by
                       * the backend.
                       */
                      sku: "",

                      color_name:
                        variant.colorName
                          .trim(),

                      /*
                       * Hex is optional
                       * in the current
                       * Admin UX.
                       */
                      color_hex:
                        "",

                      size:
                        variant.size
                          .trim(),

                      minimum_order_quantity:
                        variant.minimumOrderQuantity,

                      /*
                       * B2B quantity
                       * policy:
                       * any whole
                       * quantity at or
                       * above MOQ.
                       */
                      order_increment:
                        1,

                      price_amount:
                        bdtToMinorUnits(
                          variant.priceBdt,
                        ),

                      currency:
                        "BDT",

                      stock:
                        variant.stock,

                      reorder_level:
                        0,

                      price_tiers:
                        [],
                    }),
                  ),
              }),
          },
        );

      createdProduct =
        created.data;

      setCreatedProductId(
        createdProduct.id,
      );

      setCreatedCode(
        createdProduct
          .product_code,
      );

      if (
        media.length > 0
      ) {
        const checkpoints =
          await uploadMedia(
            createdProduct.id,
            {},
          );

        setMediaUploadCheckpoints(
          checkpoints,
        );
      }

      setImageRetryNeeded(
        false,
      );

      setProgress(
        media.length > 0
          ? `Saved ${createdProduct.product_code} with ${media.length} image${
              media.length === 1
                ? ""
                : "s"
            }.`
          : `Saved ${createdProduct.product_code}.`,
      );
    } catch (value) {
      const message =
        requestErrorMessage(
          value,
        );

      if (createdProduct) {
        setImageRetryNeeded(
          media.length > 0,
        );

        setError(
          `${message} Product ${createdProduct.product_code} was already created. Do not save it again. Use Retry remaining images.`,
        );
      } else {
        setError(
          message,
        );
      }

      setProgress("");
    } finally {
      setBusy(false);
    }
  }

  async function handleRetryMedia() {
    if (
      busy ||
      !createdProductId ||
      !createdCode ||
      media.length === 0
    ) {
      return;
    }

    setBusy(true);

    setError("");

    setProgress(
      "Retrying remaining images…",
    );

    setImageRetryNeeded(
      false,
    );

    try {
      const checkpoints =
        await uploadMedia(
          createdProductId,
          mediaUploadCheckpoints,
        );

      setMediaUploadCheckpoints(
        checkpoints,
      );

      setImageRetryNeeded(
        false,
      );

      setProgress(
        `All ${media.length} image${
          media.length === 1
            ? ""
            : "s"
        } are attached to ${createdCode}.`,
      );
    } catch (value) {
      setImageRetryNeeded(
        true,
      );

      setError(
        `${requestErrorMessage(
          value,
        )} Product ${createdCode} already exists. Retry only the remaining images.`,
      );

      setProgress("");
    } finally {
      setBusy(false);
    }
  }

  const attachedMediaCount =
    Object.values(
      mediaUploadCheckpoints,
    ).filter(
      (
        checkpoint,
      ) =>
        checkpoint.attached,
    ).length;

  const remainingMediaCount =
    Math.max(
      media.length -
        attachedMediaCount,
      0,
    );

  const formLocked =
    busy ||
    Boolean(createdCode);

  return (
    <form
      className={
        styles.page
      }
      onSubmit={
        handleSubmit
      }
    >
      <AdminPageHeader
        eyebrow="Catalog operations"
        title="Add product"
        description="Add one product manually. IDs, product codes and SKUs are generated automatically."
      />

      <div
        className={
          styles.layout
        }
      >
        <div
          className={
            styles.main
          }
        >
          <ProductIdentitySection
            value={
              identity
            }
            onChange={
              setIdentity
            }
            disabled={
              formLocked
            }
          />

          <ProductVariantList
            variants={
              variants
            }
            onChange={
              setVariants
            }
            disabled={
              formLocked
            }
          />

          <ProductMediaUploader
            items={
              media
            }
            onChange={
              setMedia
            }
            disabled={
              formLocked
            }
          />
        </div>

        <aside
          className={
            styles.side
          }
        >
          <ProductPublishPanel
            status={
              status
            }
            busy={
              formLocked
            }
            progress={
              progress
            }
            error={
              error
            }
            createdCode={
              createdCode
            }
            onStatusChange={
              setStatus
            }
          />

          {imageRetryNeeded &&
          createdCode &&
          remainingMediaCount >
            0 ? (
            <div
              className={
                styles.retryPanel
              }
            >
              <div>
                <strong
                  className={
                    styles.retryTitle
                  }
                >
                  Product already
                  created
                </strong>

                <p
                  className={
                    styles.retryText
                  }
                >
                  {
                    attachedMediaCount
                  }{" "}
                  of {media.length}{" "}
                  image
                  {media.length ===
                  1
                    ? ""
                    : "s"}{" "}
                  attached.{" "}
                  {
                    remainingMediaCount
                  }{" "}
                  remaining.
                </p>

                <p
                  className={
                    styles.retryCode
                  }
                >
                  {createdCode}
                </p>
              </div>

              <button
                type="button"
                className={
                  styles.retryButton
                }
                disabled={
                  busy
                }
                onClick={
                  handleRetryMedia
                }
              >
                {busy
                  ? "Retrying…"
                  : "Retry remaining images"}
              </button>
            </div>
          ) : null}
        </aside>
      </div>
    </form>
  );
}