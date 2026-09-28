"use client";

import Image from "next/image";

import {
  useEffect,
  useState,
} from "react";

import {
  adminFetch,
  AdminRequestError,
} from "@/lib/admin/api";

import type {
  AdminCatalogProductDetail,
  AdminCatalogProductDetailResponse,
  AdminCatalogVariant,
  AdminProductStatus,
  AdminUpdateCatalogProductRequest,
} from "@/lib/admin/catalog-types";

import {
  formatMoney,
} from "@/lib/money/format";

import ProductCategoryPicker from "./ProductCategoryPicker";

import styles from "../css/ProductsManagement.module.css";

type ProductDetailDrawerProps = {
  productId: string;

  onClose: () => void;

  onProductUpdated: (
    product:
      AdminCatalogProductDetail,
  ) => void;
};

type EditDraft = {
  categoryId: string;

  name: string;
  brand: string;

  shortDescription: string;
  description: string;

  status:
    AdminProductStatus;

  isFeatured: boolean;
};

function getErrorMessage(
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

  return "The product could not be loaded.";
}

function variantAvailable(
  variant:
    AdminCatalogVariant,
): number | null {
  const candidates = [
    variant.available_quantity,
    variant.available_stock,
    variant.stock,
    variant.quantity_on_hand,
  ];

  for (const value of candidates) {
    if (
      typeof value ===
      "number"
    ) {
      return value;
    }
  }

  return null;
}

function variantPrice(
  variant:
    AdminCatalogVariant,
): string {
  if (
    typeof variant.price_amount !==
    "number"
  ) {
    return "—";
  }

  return formatMoney(
    variant.price_amount,
    variant.currency ??
      "BDT",
  );
}

function statusClass(
  status: string,
): string {
  switch (status) {
    case "active":
      return styles.statusActive;

    case "archived":
      return styles.statusArchived;

    default:
      return styles.statusDraft;
  }
}

export default function ProductDetailDrawer({
  productId,
  onClose,
  onProductUpdated,
}: ProductDetailDrawerProps) {
  const [
    product,
    setProduct,
  ] =
    useState<AdminCatalogProductDetail | null>(
      null,
    );

  const [
    loading,
    setLoading,
  ] = useState(true);

  const [
    error,
    setError,
  ] = useState("");

  const [
    editing,
    setEditing,
  ] = useState(false);

  const [
    draft,
    setDraft,
  ] =
    useState<EditDraft | null>(
      null,
    );

  const [
    saving,
    setSaving,
  ] = useState(false);

  const [
    saveError,
    setSaveError,
  ] = useState("");

  useEffect(() => {
    let cancelled = false;

    adminFetch<AdminCatalogProductDetailResponse>(
      `/products/${encodeURIComponent(
        productId,
      )}`,
    )
      .then(
        (response) => {
          if (cancelled) {
            return;
          }

          setProduct(
            response.data,
          );

          setLoading(false);
        },
      )
      .catch(
        (value: unknown) => {
          if (cancelled) {
            return;
          }

          setError(
            getErrorMessage(
              value,
            ),
          );

          setLoading(false);
        },
      );

    return () => {
      cancelled = true;
    };
  }, [productId]);

  useEffect(() => {
    const previousOverflow =
      document.body.style
        .overflow;

    document.body.style.overflow =
      "hidden";

    function handleKeyDown(
      event:
        KeyboardEvent,
    ) {
      if (
        event.key ===
        "Escape"
      ) {
        onClose();
      }
    }

    window.addEventListener(
      "keydown",
      handleKeyDown,
    );

    return () => {
      document.body.style.overflow =
        previousOverflow;

      window.removeEventListener(
        "keydown",
        handleKeyDown,
      );
    };
  }, [onClose]);

  async function reload() {
    setLoading(true);
    setError("");

    try {
      const response =
        await adminFetch<AdminCatalogProductDetailResponse>(
          `/products/${encodeURIComponent(
            productId,
          )}`,
        );

      setProduct(
        response.data,
      );
    } catch (value) {
      setError(
        getErrorMessage(
          value,
        ),
      );
    } finally {
      setLoading(false);
    }
  }

  function beginEditing() {
    if (!product) {
      return;
    }

    setDraft({
      categoryId:
        product.category_id,

      name:
        product.name,

      brand:
        product.brand ??
        "",

      shortDescription:
        product.short_description ??
        "",

      description:
        product.description ??
        "",

      status:
        product.status,

      isFeatured:
        product.is_featured,
    });

    setSaveError("");
    setEditing(true);
  }

  function cancelEditing() {
    if (saving) {
      return;
    }

    setEditing(false);
    setDraft(null);
    setSaveError("");
  }

  async function saveProduct() {
    if (
      !product ||
      !draft ||
      saving
    ) {
      return;
    }

    const name =
      draft.name.trim();

    if (!name) {
      setSaveError(
        "Product name is required.",
      );

      return;
    }

    if (
      !draft.categoryId
    ) {
      setSaveError(
        "Choose a product-ready category.",
      );

      return;
    }

    if (
      draft.status ===
        "archived" &&
      product.status !==
        "archived"
    ) {
      const confirmed =
        window.confirm(
          "Archive this product? It will no longer be an active storefront product.",
        );

      if (!confirmed) {
        return;
      }
    }

    const request:
      AdminUpdateCatalogProductRequest =
      {
        category_id:
          draft.categoryId,

        name,

        brand:
          draft.brand.trim(),

        short_description:
          draft.shortDescription.trim(),

        description:
          draft.description.trim(),

        status:
          draft.status,

        is_featured:
          draft.isFeatured,
      };

    setSaving(true);
    setSaveError("");

    try {
      const response =
        await adminFetch<AdminCatalogProductDetailResponse>(
          `/products/${encodeURIComponent(
            product.id,
          )}`,
          {
            method: "PATCH",

            body:
              JSON.stringify(
                request,
              ),
          },
        );

      setProduct(
        response.data,
      );

      onProductUpdated(
        response.data,
      );

      setEditing(false);
      setDraft(null);
    } catch (value) {
      setSaveError(
        getErrorMessage(
          value,
        ),
      );
    } finally {
      setSaving(false);
    }
  }

  const images =
    [...(
      product?.images ??
      []
    )].sort(
      (a, b) => {
        if (
          a.is_primary !==
          b.is_primary
        ) {
          return a.is_primary
            ? -1
            : 1;
        }

        return (
          a.sort_order -
          b.sort_order
        );
      },
    );

  return (
    <div
      className={
        styles.drawerBackdrop
      }
      onMouseDown={(
        event,
      ) => {
        if (
          event.target ===
          event.currentTarget &&
          !saving
        ) {
          onClose();
        }
      }}
    >
      <section
        className={
          styles.drawer
        }
        role="dialog"
        aria-modal="true"
        aria-label="Product details"
      >
        <header
          className={
            styles.drawerHeader
          }
        >
          <div>
            <span>
              Product inspection
            </span>

            <strong>
              {product
                ? product.product_code
                : "Loading…"}
            </strong>
          </div>

          <button
            type="button"
            className={
              styles.drawerClose
            }
            onClick={
              onClose
            }
            disabled={
              saving
            }
            aria-label="Close product details"
          >
            ×
          </button>
        </header>

        {loading ? (
          <div
            className={
              styles.drawerLoading
            }
          >
            <span
              className={
                styles.loader
              }
            />

            <strong>
              Loading product
            </strong>

            <p>
              Reading variants,
              inventory and media.
            </p>
          </div>
        ) : error ? (
          <div
            className={
              styles.drawerError
            }
          >
            <strong>
              Product could not
              be loaded
            </strong>

            <p>
              {error}
            </p>

            <button
              type="button"
              onClick={() =>
                void reload()
              }
            >
              Retry
            </button>
          </div>
        ) : product ? (
          <div
            className={
              styles.drawerBody
            }
          >
            <section
              className={
                styles.productOverview
              }
            >
              <div
                className={
                  styles.overviewImage
                }
              >
                {images[0]?.url ? (
                  <Image
                    src={
                      images[0]
                        .url
                    }
                    alt={
                      images[0]
                        .alt_text ??
                      product.name
                    }
                    width={
                      180
                    }
                    height={
                      210
                    }
                    sizes="180px"
                  />
                ) : (
                  <span>
                    {product.name
                      .charAt(0)
                      .toUpperCase()}
                  </span>
                )}
              </div>

              <div
                className={
                  styles.overviewIdentity
                }
              >
                <div
                  className={
                    styles.overviewBadges
                  }
                >
                  <span
                    className={`${styles.statusBadge} ${statusClass(
                      product.status,
                    )}`}
                  >
                    {
                      product.status
                    }
                  </span>

                  {product.is_featured && (
                    <span
                      className={
                        styles.featuredMark
                      }
                    >
                      Featured
                    </span>
                  )}
                </div>

                <h2>
                  {
                    product.name
                  }
                </h2>

                <p>
                  {
                    product.category_name
                  }

                  {product.brand
                    ? ` · ${product.brand}`
                    : ""}
                </p>

                <code>
                  {
                    product.product_code
                  }
                </code>

                <div
                  className={
                    styles.overviewActions
                  }
                >
                  {product.status ===
                    "active" && (
                    <a
                      href={`/product/${product.slug}`}
                      target="_blank"
                      rel="noreferrer"
                    >
                      Open storefront
                      <span
                        aria-hidden="true"
                      >
                        ↗
                      </span>
                    </a>
                  )}

                  {!editing && (
                    <button
                      type="button"
                      onClick={
                        beginEditing
                      }
                    >
                      Edit product
                    </button>
                  )}
                </div>
              </div>
            </section>

            <div
              className={
                styles.metricGrid
              }
            >
              <div>
                <span>
                  Variants
                </span>

                <strong>
                  {
                    product.variant_count
                  }
                </strong>

                <small>
                  {
                    product.active_variant_count
                  }{" "}
                  active
                </small>
              </div>

              <div>
                <span>
                  Available stock
                </span>

                <strong>
                  {
                    product.available_stock
                  }
                </strong>

                <small>
                  units
                </small>
              </div>

              <div>
                <span>
                  Images
                </span>

                <strong>
                  {
                    product.images
                      .length
                  }
                </strong>

                <small>
                  attached
                </small>
              </div>
            </div>

            {editing &&
              draft && (
                <section
                  className={
                    styles.editPanel
                  }
                >
                  <div
                    className={
                      styles.sectionHeading
                    }
                  >
                    <div>
                      <span>
                        Catalog
                        record
                      </span>

                      <h3>
                        Edit
                        product
                      </h3>
                    </div>

                    <p>
                      Product-level
                      fields use the
                      real Admin
                      PATCH API.
                    </p>
                  </div>

                  <div
                    className={
                      styles.editGrid
                    }
                  >
                    <label
                      className={
                        styles.editField
                      }
                    >
                      <span>
                        Product
                        name
                      </span>

                      <input
                        value={
                          draft.name
                        }
                        disabled={
                          saving
                        }
                        onChange={(
                          event,
                        ) =>
                          setDraft(
                            (
                              current,
                            ) =>
                              current
                                ? {
                                    ...current,
                                    name:
                                      event
                                        .target
                                        .value,
                                  }
                                : current,
                          )
                        }
                      />
                    </label>

                    <label
                      className={
                        styles.editField
                      }
                    >
                      <span>
                        Brand
                      </span>

                      <input
                        value={
                          draft.brand
                        }
                        disabled={
                          saving
                        }
                        onChange={(
                          event,
                        ) =>
                          setDraft(
                            (
                              current,
                            ) =>
                              current
                                ? {
                                    ...current,
                                    brand:
                                      event
                                        .target
                                        .value,
                                  }
                                : current,
                          )
                        }
                      />
                    </label>

                    <div
                      className={
                        styles.categoryEdit
                      }
                    >
                      <ProductCategoryPicker
                        value={
                          draft.categoryId
                        }
                        disabled={
                          saving
                        }
                        onChange={(
                          categoryId,
                        ) =>
                          setDraft(
                            (
                              current,
                            ) =>
                              current
                                ? {
                                    ...current,
                                    categoryId,
                                  }
                                : current,
                          )
                        }
                      />
                    </div>

                    <label
                      className={
                        styles.editField
                      }
                    >
                      <span>
                        Status
                      </span>

                      <select
                        value={
                          draft.status
                        }
                        disabled={
                          saving
                        }
                        onChange={(
                          event,
                        ) =>
                          setDraft(
                            (
                              current,
                            ) =>
                              current
                                ? {
                                    ...current,

                                    status:
                                      event
                                        .target
                                        .value as AdminProductStatus,
                                  }
                                : current,
                          )
                        }
                      >
                        <option value="draft">
                          Draft
                        </option>

                        <option value="active">
                          Active
                        </option>

                        <option value="archived">
                          Archived
                        </option>
                      </select>
                    </label>

                    <label
                      className={
                        styles.featuredToggle
                      }
                    >
                      <input
                        type="checkbox"
                        checked={
                          draft.isFeatured
                        }
                        disabled={
                          saving
                        }
                        onChange={(
                          event,
                        ) =>
                          setDraft(
                            (
                              current,
                            ) =>
                              current
                                ? {
                                    ...current,

                                    isFeatured:
                                      event
                                        .target
                                        .checked,
                                  }
                                : current,
                          )
                        }
                      />

                      <span>
                        <strong>
                          Featured
                          product
                        </strong>

                        <small>
                          Use only
                          when the
                          product
                          should be
                          highlighted
                          by catalog
                          merchandising.
                        </small>
                      </span>
                    </label>

                    <label
                      className={`${styles.editField} ${styles.fullField}`}
                    >
                      <span>
                        Short
                        description
                      </span>

                      <textarea
                        rows={
                          3
                        }
                        value={
                          draft.shortDescription
                        }
                        disabled={
                          saving
                        }
                        onChange={(
                          event,
                        ) =>
                          setDraft(
                            (
                              current,
                            ) =>
                              current
                                ? {
                                    ...current,

                                    shortDescription:
                                      event
                                        .target
                                        .value,
                                  }
                                : current,
                          )
                        }
                      />
                    </label>

                    <label
                      className={`${styles.editField} ${styles.fullField}`}
                    >
                      <span>
                        Long
                        description
                      </span>

                      <textarea
                        rows={
                          7
                        }
                        value={
                          draft.description
                        }
                        disabled={
                          saving
                        }
                        onChange={(
                          event,
                        ) =>
                          setDraft(
                            (
                              current,
                            ) =>
                              current
                                ? {
                                    ...current,

                                    description:
                                      event
                                        .target
                                        .value,
                                  }
                                : current,
                          )
                        }
                      />
                    </label>
                  </div>

                  {saveError && (
                    <div
                      className={
                        styles.inlineError
                      }
                      role="alert"
                    >
                      {
                        saveError
                      }
                    </div>
                  )}

                  <div
                    className={
                      styles.editActions
                    }
                  >
                    <button
                      type="button"
                      className={
                        styles.cancelButton
                      }
                      onClick={
                        cancelEditing
                      }
                      disabled={
                        saving
                      }
                    >
                      Cancel
                    </button>

                    <button
                      type="button"
                      className={
                        styles.saveButton
                      }
                      onClick={() =>
                        void saveProduct()
                      }
                      disabled={
                        saving
                      }
                    >
                      {saving
                        ? "Saving…"
                        : "Save changes"}
                    </button>
                  </div>
                </section>
              )}

            {!editing && (
              <section
                className={
                  styles.descriptionPanel
                }
              >
                <div
                  className={
                    styles.sectionHeading
                  }
                >
                  <div>
                    <span>
                      Content
                    </span>

                    <h3>
                      Product
                      description
                    </h3>
                  </div>
                </div>

                <div
                  className={
                    styles.descriptionGrid
                  }
                >
                  <div>
                    <span>
                      Short
                      description
                    </span>

                    <p>
                      {product.short_description ||
                        "No short description."}
                    </p>
                  </div>

                  <div>
                    <span>
                      Long
                      description
                    </span>

                    <p>
                      {product.description ||
                        "No long description."}
                    </p>
                  </div>
                </div>
              </section>
            )}

            <section
              className={
                styles.detailSection
              }
            >
              <div
                className={
                  styles.sectionHeading
                }
              >
                <div>
                  <span>
                    Sellable units
                  </span>

                  <h3>
                    Variants
                  </h3>
                </div>

                <p>
                  SKU, price, MOQ
                  and inventory are
                  read from the
                  backend.
                </p>
              </div>

              {product.variants
                .length === 0 ? (
                <div
                  className={
                    styles.smallEmpty
                  }
                >
                  No variants
                  returned.
                </div>
              ) : (
                <div
                  className={
                    styles.variantList
                  }
                >
                  {product.variants.map(
                    (
                      variant,
                    ) => {
                      const available =
                        variantAvailable(
                          variant,
                        );

                      return (
                        <article
                          key={
                            variant.id
                          }
                          className={
                            styles.variantCard
                          }
                        >
                          <div
                            className={
                              styles.variantIdentity
                            }
                          >
                            {variant.color_hex && (
                              <span
                                className={
                                  styles.colorDot
                                }
                                style={{
                                  backgroundColor:
                                    variant.color_hex,
                                }}
                              />
                            )}

                            <div>
                              <strong>
                                {variant.color_name ||
                                  "Variant"}

                                {variant.size
                                  ? ` · ${variant.size}`
                                  : ""}
                              </strong>

                              <code>
                                {
                                  variant.sku
                                }
                              </code>
                            </div>
                          </div>

                          <div
                            className={
                              styles.variantMetric
                            }
                          >
                            <span>
                              Price
                            </span>

                            <strong>
                              {variantPrice(
                                variant,
                              )}
                            </strong>
                          </div>

                          <div
                            className={
                              styles.variantMetric
                            }
                          >
                            <span>
                              MOQ
                            </span>

                            <strong>
                              {typeof variant.minimum_order_quantity ===
                              "number"
                                ? variant.minimum_order_quantity
                                : "—"}
                            </strong>
                          </div>

                          <div
                            className={
                              styles.variantMetric
                            }
                          >
                            <span>
                              Available
                            </span>

                            <strong>
                              {available ??
                                "—"}
                            </strong>
                          </div>

                          {typeof variant.is_active ===
                            "boolean" && (
                            <span
                              className={
                                variant.is_active
                                  ? styles.variantActive
                                  : styles.variantInactive
                              }
                            >
                              {variant.is_active
                                ? "Active"
                                : "Inactive"}
                            </span>
                          )}
                        </article>
                      );
                    },
                  )}
                </div>
              )}
            </section>

            <section
              className={
                styles.detailSection
              }
            >
              <div
                className={
                  styles.sectionHeading
                }
              >
                <div>
                  <span>
                    Object storage
                  </span>

                  <h3>
                    Product
                    images
                  </h3>
                </div>

                <p>
                  {
                    images.length
                  }{" "}
                  attached
                </p>
              </div>

              {images.length ===
              0 ? (
                <div
                  className={
                    styles.smallEmpty
                  }
                >
                  No product images
                  attached.
                </div>
              ) : (
                <div
                  className={
                    styles.imageGrid
                  }
                >
                  {images.map(
                    (
                      image,
                    ) => (
                      <article
                        key={
                          image.id
                        }
                        className={
                          styles.imageCard
                        }
                      >
                        <div
                          className={
                            styles.imagePreview
                          }
                        >
                          <Image
                            src={
                              image.url
                            }
                            alt={
                              image.alt_text ??
                              product.name
                            }
                            width={
                              170
                            }
                            height={
                              190
                            }
                            sizes="170px"
                          />

                          {image.is_primary && (
                            <span>
                              Primary
                            </span>
                          )}
                        </div>

                        <div
                          className={
                            styles.imageMeta
                          }
                        >
                          <strong>
                            {image.alt_text ||
                              "Product image"}
                          </strong>

                          <span>
                            Sort{" "}
                            {
                              image.sort_order
                            }
                          </span>

                          {image.variant_id && (
                            <small>
                              Variant
                              image
                            </small>
                          )}
                        </div>
                      </article>
                    ),
                  )}
                </div>
              )}
            </section>
          </div>
        ) : null}
      </section>
    </div>
  );
}