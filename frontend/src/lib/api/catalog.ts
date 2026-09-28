import { commerceFetch } from "./server";

import type {
  ActivePromotionsResponse,
  CategoryDetailResponse,
  CategoryNode,
  CategoryTreeResponse,
  ProductDetailResponse,
  ProductListResponse,
  SearchResponse,
} from "./contracts/catalog";

function sortCategories(
  categories: CategoryNode[],
): CategoryNode[] {
  return [...categories]
    .sort(
      (a, b) =>
        a.sort_order - b.sort_order,
    )
    .map((category) => ({
      ...category,

      children: sortCategories(
        category.children ?? [],
      ),
    }));
}

export async function getCategoryTree() {
  const response =
    await commerceFetch<CategoryTreeResponse>(
      "/api/v1/categories/tree",
      {
        next: {
          revalidate: 300,
          tags: ["categories"],
        },
      },
    );

  return sortCategories(response.data);
}

export async function getCategory(
  slug: string,
) {
  const response =
    await commerceFetch<CategoryDetailResponse>(
      `/api/v1/categories/${encodeURIComponent(
        slug,
      )}`,
      {
        next: {
          revalidate: 300,
          tags: [
            "categories",
            `category:${slug}`,
          ],
        },
      },
    );

  return response.data;
}

export async function getProducts(
  limit = 12,
): Promise<ProductListResponse> {
  const params = new URLSearchParams({
    limit: String(limit),
  });

  return commerceFetch<ProductListResponse>(
    `/api/v1/products?${params.toString()}`,
    {
      next: {
        revalidate: 60,
        tags: ["products"],
      },
    },
  );
}

export type ProductListSort =
  | "recommended"
  | "featured"
  | "newest"
  | "price_asc"
  | "price_desc"
  | "name_asc"
  | "name_desc";

export type ProductListOptions = {
  page?: number;
  minPriceAmount?: number;
  maxPriceAmount?: number;
  inStock?: boolean;
  sort?: ProductListSort;
};

export async function getProductsByCategory(
  categorySlug: string,
  limit = 24,
  options: ProductListOptions = {},
) {
  const params = new URLSearchParams({
    category: categorySlug,
    limit: String(limit),
    page: String(options.page ?? 1),
  });

  if (
    typeof options.minPriceAmount ===
      "number" &&
    Number.isFinite(
      options.minPriceAmount,
    )
  ) {
    params.set(
      "min_price",
      String(
        Math.max(
          0,
          Math.trunc(
            options.minPriceAmount,
          ),
        ),
      ),
    );
  }

  if (
    typeof options.maxPriceAmount ===
      "number" &&
    Number.isFinite(
      options.maxPriceAmount,
    )
  ) {
    params.set(
      "max_price",
      String(
        Math.max(
          0,
          Math.trunc(
            options.maxPriceAmount,
          ),
        ),
      ),
    );
  }

  if (
    typeof options.inStock ===
    "boolean"
  ) {
    params.set(
      "in_stock",
      String(options.inStock),
    );
  }

  if (options.sort) {
    params.set(
      "sort",
      options.sort,
    );
  }

  const requestPath =
    `/api/v1/products?${params.toString()}`;

  const response =
    await commerceFetch<ProductListResponse>(
      requestPath,
      {
        next: {
          revalidate: 60,
          tags: [
            "products",
            `category-products:${categorySlug}`,
          ],
        },
      },
    );

  return response;
}

export async function getProduct(
  slug: string,
) {
  const response =
    await commerceFetch<ProductDetailResponse>(
      `/api/v1/products/${encodeURIComponent(
        slug,
      )}`,
      {
        next: {
          revalidate: 60,
          tags: [
            "products",
            `product:${slug}`,
          ],
        },
      },
    );

  return response.data;
}

export async function searchProducts(
  query: string,
  limit = 24,
) {
  const params = new URLSearchParams({
    q: query,
    limit: String(limit),
  });

  return commerceFetch<SearchResponse>(
    `/api/v1/search/products?${params.toString()}`,
    {
      next: {
        revalidate: 30,
        tags: ["search-products"],
      },
    },
  );
}

export async function getActivePromotions() {
  const response =
    await commerceFetch<ActivePromotionsResponse>(
      "/api/v1/promotions/active",
      {
        next: {
          revalidate: 30,
          tags: ["promotions"],
        },
      },
    );

  return response.data;
}