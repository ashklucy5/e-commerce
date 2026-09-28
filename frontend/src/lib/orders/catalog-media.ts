import "server-only";

import { commerceFetch } from "@/lib/api/server";
import type {
  ProductDetail,
  ProductDetailResponse,
  ProductImage,
} from "@/lib/api/contracts/catalog";

type CatalogSearchItem = {
  id: string;
  name: string;
  slug: string;
  primary_image_url?: string | null;
  matched_sku?: string | null;
};

type CatalogSearchResponse = {
  data: CatalogSearchItem[];
};

export type OrderCatalogMedia = {
  image_url?: string;
  product_slug?: string;
};

type OrderItemMediaInput = {
  sku: string;
  productName: string;
  variantId?: string;
};

function normalize(value: string | null | undefined) {
  return String(value ?? "").trim().toLocaleLowerCase();
}

function clean(value: string | null | undefined) {
  const normalized = String(value ?? "").trim();
  return normalized || undefined;
}

function orderedImages(images: ProductImage[]) {
  return [...images].sort((left, right) => {
    if (left.is_primary !== right.is_primary) {
      return left.is_primary ? -1 : 1;
    }

    if (left.sort_order !== right.sort_order) {
      return left.sort_order - right.sort_order;
    }

    return left.id.localeCompare(right.id);
  });
}

function selectProductImage(product: ProductDetail, variantId?: string) {
  const images = orderedImages(
    Array.isArray(product.images) ? product.images : [],
  );

  const normalizedVariantId = normalize(variantId);

  if (normalizedVariantId) {
    const variantImage = images.find(
      (image) =>
        normalize(image.variant_id) === normalizedVariantId &&
        clean(image.url),
    );

    if (variantImage) {
      return clean(variantImage.url);
    }
  }

  const primaryImage = images.find(
    (image) =>
      image.is_primary &&
      clean(image.url),
  );

  if (primaryImage) {
    return clean(primaryImage.url);
  }

  const productImage = images.find(
    (image) =>
      !clean(image.variant_id) &&
      clean(image.url),
  );

  if (productImage) {
    return clean(productImage.url);
  }

  return images
    .map((image) => clean(image.url))
    .find(Boolean);
}

async function loadProductDetail(
  slug: string,
  variantId?: string,
): Promise<OrderCatalogMedia> {
  const cleanSlug = clean(slug);

  if (!cleanSlug) {
    return {};
  }

  try {
    const response =
      await commerceFetch<ProductDetailResponse>(
        `/api/v1/products/${encodeURIComponent(cleanSlug)}`,
        {
          next: {
            revalidate: 300,
            tags: [
              `order-product-media:${cleanSlug}`,
            ],
          },
        },
      );

    return {
      image_url:
        selectProductImage(
          response.data,
          variantId,
        ),

      product_slug:
        clean(response.data.slug) ??
        cleanSlug,
    };
  } catch {
    return {
      product_slug: cleanSlug,
    };
  }
}

async function searchCatalog(
  queryValue: string,
) {
  const query = clean(queryValue);

  if (!query) {
    return [] as CatalogSearchItem[];
  }

  const params =
    new URLSearchParams({
      q: query,
      limit: "6",
    });

  try {
    const response =
      await commerceFetch<CatalogSearchResponse>(
        `/api/v1/search/products?${params.toString()}`,
        {
          next: {
            revalidate: 300,
            tags: [
              `order-catalog-search:${normalize(query)}`,
            ],
          },
        },
      );

    return Array.isArray(response.data)
      ? response.data
      : [];
  } catch {
    return [] as CatalogSearchItem[];
  }
}

async function mediaFromProduct(
  product: CatalogSearchItem,
  variantId?: string,
) {
  const directImage =
    clean(product.primary_image_url);

  const slug =
    clean(product.slug);

  if (directImage) {
    return {
      image_url: directImage,
      product_slug: slug,
    } satisfies OrderCatalogMedia;
  }

  if (!slug) {
    return {};
  }

  return loadProductDetail(
    slug,
    variantId,
  );
}

export async function catalogMediaForOrderItem(
  input: OrderItemMediaInput,
): Promise<OrderCatalogMedia> {
  const sku =
    clean(input.sku);

  const productName =
    clean(input.productName);

  if (sku) {
    const products =
      await searchCatalog(sku);

    const exactSku =
      products.find(
        (product) =>
          normalize(
            product.matched_sku,
          ) === normalize(sku),
      );

    const exactName =
      productName
        ? products.find(
            (product) =>
              normalize(
                product.name,
              ) ===
              normalize(
                productName,
              ),
          )
        : undefined;

    const matched =
      exactSku ??
      exactName;

    if (matched) {
      return mediaFromProduct(
        matched,
        input.variantId,
      );
    }
  }

  if (!productName) {
    return {};
  }

  const products =
    await searchCatalog(
      productName,
    );

  const exactName =
    products.find(
      (product) =>
        normalize(
          product.name,
        ) ===
        normalize(
          productName,
        ),
    );

  if (!exactName) {
    return {};
  }

  return mediaFromProduct(
    exactName,
    input.variantId,
  );
}

export async function catalogMediaForOrderProductName(
  productName: string,
): Promise<OrderCatalogMedia> {
  const name =
    clean(productName);

  if (!name) {
    return {};
  }

  const products =
    await searchCatalog(name);

  const exactName =
    products.find(
      (product) =>
        normalize(
          product.name,
        ) ===
        normalize(name),
    );

  if (!exactName) {
    return {};
  }

  return mediaFromProduct(
    exactName,
  );
}