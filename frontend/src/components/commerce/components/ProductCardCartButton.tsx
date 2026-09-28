"use client";

import { useRouter } from "next/navigation";
import { useState } from "react";

import { notifyCartUpdated } from "@/components/commerce/CartBadge";
import { Icon } from "@/components/ui/Icon";
import type { Cart } from "@/lib/api/contracts/commerce";

type PurchaseVariant = {
  id: string;
  sku: string;
  color_name: string | null;
  size: string | null;
  minimum_order_quantity: number;
  available_quantity: number;
  in_stock: boolean;
};

type PurchaseOptionsResponse = {
  data: {
    slug: string;
    variants: PurchaseVariant[];
  };
};

type Props = {
  productName: string;
  productSlug: string;
  inStock: boolean;
  className?: string;
};

type State =
  | "idle"
  | "loading"
  | "added"
  | "error";

async function readError(response: Response) {
  try {
    const payload = (await response.json()) as {
      error?: {
        message?: string;
      };
    };

    return (
      payload.error?.message ||
      "Unable to add this product to cart."
    );
  } catch {
    return "Unable to add this product to cart.";
  }
}

export function ProductCardCartButton({
  productName,
  productSlug,
  inStock,
  className = "",
}: Props) {
  const router = useRouter();
  const [state, setState] = useState<State>("idle");

  const disabled = !inStock || state === "loading";

  async function handleClick() {
    if (disabled) {
      return;
    }

    setState("loading");

    try {
      const optionsResponse = await fetch(
        `/api/storefront/products/${encodeURIComponent(
          productSlug,
        )}/purchase-options`,
        {
          cache: "no-store",
        },
      );

      if (!optionsResponse.ok) {
        throw new Error(await readError(optionsResponse));
      }

      const options =
        (await optionsResponse.json()) as PurchaseOptionsResponse;

      const purchasable = options.data.variants.filter(
        (variant) =>
          variant.in_stock &&
          variant.available_quantity >=
            Math.max(1, variant.minimum_order_quantity),
      );

      if (purchasable.length === 0) {
        setState("error");
        return;
      }

      /*
       * A card does not expose variant choice. Never silently add an
       * arbitrary color/size when multiple valid variants exist.
       */
      if (purchasable.length > 1) {
        router.push(`/product/${productSlug}`);
        return;
      }

      const variant = purchasable[0];
      const quantity = Math.max(1, variant.minimum_order_quantity);

      const cartResponse = await fetch(
        "/api/storefront/cart/items",
        {
          method: "POST",
          headers: {
            "Content-Type": "application/json",
          },
          body: JSON.stringify({
            variant_id: variant.id,
            quantity,
          }),
        },
      );

      if (!cartResponse.ok) {
        throw new Error(await readError(cartResponse));
      }

      const payload = (await cartResponse.json()) as {
        data: Cart;
      };

      notifyCartUpdated(payload.data);
      setState("added");

      window.setTimeout(() => {
        setState("idle");
      }, 1400);
    } catch {
      setState("error");

      window.setTimeout(() => {
        setState("idle");
      }, 1800);
    }
  }

  const label = !inStock
    ? `${productName} is out of stock`
    : state === "loading"
      ? `Adding ${productName} to cart`
      : state === "added"
        ? `${productName} added to cart`
        : state === "error"
          ? `Could not add ${productName} to cart`
          : `Add ${productName} to cart`;

  return (
    <button
      type="button"
      className={className}
      aria-label={label}
      title={
        inStock
          ? "Add to cart"
          : "Out of stock"
      }
      disabled={disabled}
      data-state={state}
      onClick={() => {
        void handleClick();
      }}
    >
      {state === "added" ? (
        <Icon name="cartFilled" size={16} />
      ) : state === "error" ? (
        <Icon name="close" size={14} />
      ) : (
        <Icon name="cart" size={16} />
      )}

      <span className="sr-only" aria-live="polite">
        {state === "added"
          ? "Added to cart"
          : state === "error"
            ? "Unable to add to cart"
            : ""}
      </span>
    </button>
  );
}
