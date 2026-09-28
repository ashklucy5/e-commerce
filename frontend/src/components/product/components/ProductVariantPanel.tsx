"use client";

import {
  useMemo,
  useState,
} from "react";

import {
  useRouter,
} from "next/navigation";

import {
  notifyCartUpdated,
} from "@/components/commerce/CartBadge";

import {
  Icon,
} from "@/components/ui/Icon";

import type {
  Cart,
} from "@/lib/api/contracts/commerce";

import type {
  ProductVariant,
} from "@/lib/api/contracts/catalog";

import {
  formatMoney,
} from "@/lib/money/format";

import styles from "../css/ProductVariantPanel.module.css";

type Props = {
  productName: string;

  variants:
    ProductVariant[];

  selectedVariant:
    ProductVariant;

  quantity: number;

  onSelectVariant: (
    variant: ProductVariant,
  ) => void;

  onQuantityChange: (
    quantity: number,
  ) => void;
};

type ApiErrorPayload = {
  error?: {
    code?: string;
    message?: string;
  };
};

type ColorOption = {
  key: string;

  name: string;

  hex?: string | null;

  variants:
    ProductVariant[];
};

function getEffectiveUnitPrice(
  variant: ProductVariant,
  quantity: number,
) {
  let price =
    variant.price_amount;

  const tiers =
    [...variant.price_tiers]
      .sort(
        (
          left,
          right,
        ) =>
          left.min_quantity -
          right.min_quantity,
      );

  for (
    const tier
    of tiers
  ) {
    if (
      quantity >=
      tier.min_quantity
    ) {
      price =
        tier.unit_price_amount;
    }
  }

  return price;
}

async function readError(
  response: Response,
) {
  try {
    const payload =
      (await response.json()) as
        ApiErrorPayload;

    return (
      payload.error?.message ??
      "Unable to complete this action."
    );
  } catch {
    return "Unable to complete this action.";
  }
}

function colorKey(
  variant: ProductVariant,
) {
  const name =
    variant.color_name
      ?.trim()
      .toLowerCase();

  if (name) {
    return `name:${name}`;
  }

  const hex =
    variant.color_hex
      ?.trim()
      .toLowerCase();

  if (hex) {
    return `hex:${hex}`;
  }

  return "default";
}

function displayColor(
  variant: ProductVariant,
) {
  return (
    variant.color_name
      ?.trim() ||
    "Default"
  );
}

function displaySize(
  variant: ProductVariant,
) {
  return (
    variant.size
      ?.trim() ||
    ""
  );
}

function stockLabel(
  quantity: number,
) {
  return `${quantity.toLocaleString()} available`;
}

export function ProductVariantPanel({
  productName,
  variants,
  selectedVariant,
  quantity,
  onSelectVariant,
  onQuantityChange,
}: Props) {
  const router =
    useRouter();

  const [
    busy,
    setBusy,
  ] =
    useState<
      "cart" |
      "buy" |
      null
    >(null);

  const [
    message,
    setMessage,
  ] =
    useState("");

  const [
    error,
    setError,
  ] =
    useState("");

  const colorOptions =
    useMemo(
      () => {
        const map =
          new Map<
            string,
            ColorOption
          >();

        for (
          const variant
          of variants
        ) {
          const key =
            colorKey(
              variant,
            );

          const existing =
            map.get(
              key,
            );

          if (existing) {
            existing.variants.push(
              variant,
            );

            continue;
          }

          map.set(
            key,
            {
              key,

              name:
                displayColor(
                  variant,
                ),

              hex:
                variant.color_hex,

              variants: [
                variant,
              ],
            },
          );
        }

        return Array.from(
          map.values(),
        );
      },
      [
        variants,
      ],
    );

  const sizeOptions =
    useMemo(
      () => {
        const sizes =
          new Set<string>();

        for (
          const variant
          of variants
        ) {
          const size =
            displaySize(
              variant,
            );

          if (size) {
            sizes.add(
              size,
            );
          }
        }

        return Array.from(
          sizes,
        );
      },
      [
        variants,
      ],
    );

  const selectedColorKey =
    colorKey(
      selectedVariant,
    );

  const selectedSize =
    displaySize(
      selectedVariant,
    );

  const sortedTiers =
    useMemo(
      () =>
        [
          ...selectedVariant
            .price_tiers,
        ].sort(
          (
            left,
            right,
          ) =>
            left.min_quantity -
            right.min_quantity,
        ),
      [
        selectedVariant,
      ],
    );

  const effectiveUnitPrice =
    getEffectiveUnitPrice(
      selectedVariant,
      quantity,
    );

  const subtotal =
    effectiveUnitPrice *
    quantity;

  const minimumQuantity =
    Math.max(
      1,
      selectedVariant
        .minimum_order_quantity,
    );

  const availableQuantity =
    Math.max(
      0,
      selectedVariant
        .available_quantity,
    );

  const belowMOQ =
    quantity <
    minimumQuantity;

  const aboveStock =
    quantity >
    availableQuantity;

  const canBuy =
    selectedVariant.in_stock &&
    availableQuantity > 0 &&
    quantity > 0 &&
    !belowMOQ &&
    !aboveStock;

  const compareAt =
    selectedVariant
      .compare_at_price_amount;

  const discounted =
    typeof compareAt ===
      "number" &&
    compareAt >
      effectiveUnitPrice;

  const discountPercent =
    discounted &&
    compareAt
      ? Math.round(
          ((compareAt -
            effectiveUnitPrice) /
            compareAt) *
            100,
        )
      : 0;

  function clearStatus() {
    setMessage("");
    setError("");
  }

  function updateQuantity(
    next: number,
  ) {
    if (
      !Number.isFinite(
        next,
      )
    ) {
      return;
    }

    onQuantityChange(
      Math.max(
        0,
        Math.floor(
          next,
        ),
      ),
    );

    clearStatus();
  }

  function selectColor(
    option: ColorOption,
  ) {
    const currentSize =
      selectedSize;

    const preferred =
      option.variants.find(
        (variant) =>
          variant.in_stock &&
          displaySize(
            variant,
          ) ===
            currentSize,
      ) ??
      option.variants.find(
        (variant) =>
          variant.in_stock,
      ) ??
      option.variants[0];

    if (
      preferred
    ) {
      onSelectVariant(
        preferred,
      );

      clearStatus();
    }
  }

  function selectSize(
    size: string,
  ) {
    const matchingColor =
      variants.filter(
        (variant) =>
          colorKey(
            variant,
          ) ===
          selectedColorKey,
      );

    const preferred =
      matchingColor.find(
        (variant) =>
          displaySize(
            variant,
          ) ===
            size &&
          variant.in_stock,
      ) ??
      matchingColor.find(
        (variant) =>
          displaySize(
            variant,
          ) === size,
      ) ??
      variants.find(
        (variant) =>
          displaySize(
            variant,
          ) ===
            size &&
          variant.in_stock,
      );

    if (
      preferred
    ) {
      onSelectVariant(
        preferred,
      );

      clearStatus();
    }
  }

  function sizeAvailable(
    size: string,
  ) {
    return variants.some(
      (variant) =>
        colorKey(
          variant,
        ) ===
          selectedColorKey &&
        displaySize(
          variant,
        ) ===
          size &&
        variant.in_stock,
    );
  }

  async function addToCart() {
    if (
      !canBuy ||
      busy
    ) {
      return;
    }

    setBusy(
      "cart",
    );

    clearStatus();

    try {
      const response =
        await fetch(
          "/api/storefront/cart/items",
          {
            method:
              "POST",

            headers: {
              "Content-Type":
                "application/json",
            },

            body:
              JSON.stringify({
                variant_id:
                  selectedVariant.id,

                quantity,
              }),
          },
        );

      if (
        !response.ok
      ) {
        throw new Error(
          await readError(
            response,
          ),
        );
      }

      const payload =
        (await response.json()) as {
          data: Cart;
        };

      notifyCartUpdated(
        payload.data,
      );

      setMessage(
        `${productName} added to cart.`,
      );
    } catch (
      caught
    ) {
      setError(
        caught instanceof Error
          ? caught.message
          : "Unable to add this product to cart.",
      );
    } finally {
      setBusy(
        null,
      );
    }
  }

  async function buyNow() {
    if (
      !canBuy ||
      busy
    ) {
      return;
    }

    setBusy(
      "buy",
    );

    clearStatus();

    try {
      const response =
        await fetch(
          "/api/storefront/checkout/buy-now",
          {
            method:
              "POST",

            headers: {
              "Content-Type":
                "application/json",
            },

            body:
              JSON.stringify({
                variant_id:
                  selectedVariant.id,

                quantity,
              }),
          },
        );

      if (
        !response.ok
      ) {
        throw new Error(
          await readError(
            response,
          ),
        );
      }

      router.push(
        "/checkout",
      );

      router.refresh();
    } catch (
      caught
    ) {
      setError(
        caught instanceof Error
          ? caught.message
          : "Unable to start checkout.",
      );

      setBusy(
        null,
      );
    }
  }

  return (
    <section
      className={
        styles.panel
      }
      aria-label="Product purchase options"
    >
      <div
        className={
          styles.priceHeader
        }
      >
        <div>
          <span
            className={
              styles.priceLabel
            }
          >
            Unit price
          </span>

          <div
            className={
              styles.priceLine
            }
          >
            <strong
              className={
                styles.price
              }
            >
              {formatMoney(
                effectiveUnitPrice,
                selectedVariant
                  .currency,
              )}
            </strong>

            {discounted &&
            compareAt ? (
              <del
                className={
                  styles.comparePrice
                }
              >
                {formatMoney(
                  compareAt,
                  selectedVariant
                    .currency,
                )}
              </del>
            ) : null}

            {discountPercent >
            0 ? (
              <span
                className={
                  styles.discount
                }
              >
                -
                {
                  discountPercent
                }
                %
              </span>
            ) : null}
          </div>
        </div>

        <span
          className={
            selectedVariant
              .in_stock
              ? styles.stock
              : styles.stockUnavailable
          }
        >
          <span
            aria-hidden="true"
            className={
              styles.stockDot
            }
          />

          {selectedVariant
            .in_stock
            ? "In stock"
            : "Out of stock"}
        </span>
      </div>

      <div
        className={
          styles.quickFacts
        }
      >
        <div>
          <span>
            MOQ
          </span>

          <strong>
            {
              minimumQuantity
            }{" "}
            units
          </strong>
        </div>

        <div>
          <span>
            Stock
          </span>

          <strong>
            {stockLabel(
              availableQuantity,
            )}
          </strong>
        </div>

        <div>
          <span>
            SKU
          </span>

          <strong>
            {
              selectedVariant
                .sku
            }
          </strong>
        </div>
      </div>

      {colorOptions.length >
      1 ||
      colorOptions[0]?.name !==
        "Default" ? (
        <div
          className={
            styles.optionSection
          }
        >
          <div
            className={
              styles.optionHeading
            }
          >
            <strong>
              Color
            </strong>

            <span>
              {
                displayColor(
                  selectedVariant,
                )
              }
            </span>
          </div>

          <div
            className={
              styles.colors
            }
          >
            {colorOptions.map(
              (
                option,
              ) => {
                const active =
                  option.key ===
                  selectedColorKey;

                const available =
                  option.variants.some(
                    (
                      variant,
                    ) =>
                      variant.in_stock,
                  );

                return (
                  <button
                    key={
                      option.key
                    }
                    type="button"
                    className={`${styles.colorOption} ${
                      active
                        ? styles.colorOptionActive
                        : ""
                    }`}
                    disabled={
                      !available
                    }
                    aria-label={`Select ${option.name}`}
                    aria-pressed={
                      active
                    }
                    onClick={() =>
                      selectColor(
                        option,
                      )
                    }
                  >
                    <span
                      className={
                        styles.swatch
                      }
                      style={{
                        background:
                          option.hex ||
                          "#d6d6da",
                      }}
                      aria-hidden="true"
                    />

                    <span>
                      {
                        option.name
                      }
                    </span>
                  </button>
                );
              },
            )}
          </div>
        </div>
      ) : null}

      {sizeOptions.length >
      0 ? (
        <div
          className={
            styles.optionSection
          }
        >
          <div
            className={
              styles.optionHeading
            }
          >
            <strong>
              Size
            </strong>

            <span>
              {selectedSize ||
                "Choose"}
            </span>
          </div>

          <div
            className={
              styles.sizes
            }
          >
            {sizeOptions.map(
              (
                size,
              ) => {
                const active =
                  size ===
                  selectedSize;

                const available =
                  sizeAvailable(
                    size,
                  );

                return (
                  <button
                    key={
                      size
                    }
                    type="button"
                    className={`${styles.sizeOption} ${
                      active
                        ? styles.sizeOptionActive
                        : ""
                    }`}
                    disabled={
                      !available
                    }
                    aria-pressed={
                      active
                    }
                    onClick={() =>
                      selectSize(
                        size,
                      )
                    }
                  >
                    {size}
                  </button>
                );
              },
            )}
          </div>
        </div>
      ) : null}

      {sortedTiers.length >
      0 ? (
        <div
          className={
            styles.optionSection
          }
        >
          <div
            className={
              styles.optionHeading
            }
          >
            <strong>
              Volume pricing
            </strong>

            <span>
              Applied automatically
            </span>
          </div>

          <div
            className={
              styles.tiers
            }
          >
            <div
              className={`${styles.tier} ${
                quantity >=
                  minimumQuantity &&
                (
                  sortedTiers
                    .length ===
                    0 ||
                  quantity <
                    sortedTiers[0]
                      .min_quantity
                )
                  ? styles.tierActive
                  : ""
              }`}
            >
              <span>
                {
                  minimumQuantity
                }
                +
              </span>

              <strong>
                {formatMoney(
                  selectedVariant
                    .price_amount,
                  selectedVariant
                    .currency,
                )}
              </strong>

              <small>
                / unit
              </small>
            </div>

            {sortedTiers.map(
              (
                tier,
              ) => {
                const active =
                  quantity >=
                    tier.min_quantity &&
                  !sortedTiers.some(
                    (
                      candidate,
                    ) =>
                      candidate.min_quantity >
                        tier.min_quantity &&
                      quantity >=
                        candidate.min_quantity,
                  );

                return (
                  <div
                    key={
                      tier.min_quantity
                    }
                    className={`${styles.tier} ${
                      active
                        ? styles.tierActive
                        : ""
                    }`}
                  >
                    <span>
                      {
                        tier.min_quantity
                      }
                      +
                    </span>

                    <strong>
                      {formatMoney(
                        tier.unit_price_amount,
                        selectedVariant
                          .currency,
                      )}
                    </strong>

                    <small>
                      / unit
                    </small>
                  </div>
                );
              },
            )}
          </div>
        </div>
      ) : null}

      <div
        className={
          styles.quantitySection
        }
      >
        <div
          className={
            styles.optionHeading
          }
        >
          <strong>
            Quantity
          </strong>

          <span>
            Any whole number
            at or above MOQ
          </span>
        </div>

        <div
          className={
            styles.quantityRow
          }
        >
          <div
            className={
              styles.quantityControl
            }
          >
            <button
              type="button"
              aria-label="Decrease quantity"
              onClick={() =>
                updateQuantity(
                  Math.max(
                    minimumQuantity,
                    quantity -
                      1,
                  ),
                )
              }
            >
              <Icon
                name="minus"
                size={16}
              />
            </button>

            <input
              type="number"
              inputMode="numeric"
              min={
                minimumQuantity
              }
              max={
                availableQuantity
              }
              step={1}
              value={
                quantity
              }
              aria-label="Order quantity"
              onChange={(
                event,
              ) =>
                updateQuantity(
                  Number(
                    event.target
                      .value,
                  ),
                )
              }
            />

            <button
              type="button"
              aria-label="Increase quantity"
              onClick={() =>
                updateQuantity(
                  Math.min(
                    availableQuantity,
                    quantity +
                      1,
                  ),
                )
              }
            >
              <Icon
                name="plus"
                size={16}
              />
            </button>
          </div>

          <div
            className={
              styles.quantityInfo
            }
          >
            <span>
              Minimum{" "}
              <strong>
                {
                  minimumQuantity
                }
              </strong>
            </span>

            <span>
              Available{" "}
              <strong>
                {
                  availableQuantity
                }
              </strong>
            </span>
          </div>
        </div>

        {belowMOQ ? (
          <p
            className={
              styles.validationError
            }
          >
            Minimum order is{" "}
            {
              minimumQuantity
            }{" "}
            units.
          </p>
        ) : null}

        {aboveStock ? (
          <p
            className={
              styles.validationError
            }
          >
            Only{" "}
            {
              availableQuantity
            }{" "}
            units are currently
            available.
          </p>
        ) : null}
      </div>

      <div
        className={
          styles.summary
        }
      >
        <div>
          <span>
            Unit price
          </span>

          <strong>
            {formatMoney(
              effectiveUnitPrice,
              selectedVariant
                .currency,
            )}
          </strong>
        </div>

        <div>
          <span>
            Estimated subtotal
          </span>

          <strong
            className={
              styles.subtotal
            }
          >
            {formatMoney(
              subtotal,
              selectedVariant
                .currency,
            )}
          </strong>
        </div>
      </div>

      {message ||
      error ? (
        <div
          className={
            error
              ? styles.messageError
              : styles.message
          }
          role="status"
        >
          {error ||
            message}
        </div>
      ) : null}

      <div
        className={
          styles.actions
        }
      >
        <button
          type="button"
          className={
            styles.addToCart
          }
          disabled={
            !canBuy ||
            busy !== null
          }
          onClick={() =>
            void addToCart()
          }
        >
          <Icon
            name="cart"
            size={19}
          />

          <span>
            {busy ===
            "cart"
              ? "Adding…"
              : "Add to cart"}
          </span>
        </button>

        <button
          type="button"
          className={
            styles.buyNow
          }
          disabled={
            !canBuy ||
            busy !== null
          }
          onClick={() =>
            void buyNow()
          }
        >
          <span>
            {busy ===
            "buy"
              ? "Opening checkout…"
              : "Buy now"}
          </span>
        </button>
      </div>
    </section>
  );
}