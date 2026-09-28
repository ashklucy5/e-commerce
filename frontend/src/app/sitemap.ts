import type { MetadataRoute } from "next";

import { getCategoryTree } from "@/lib/api/catalog";
import type { CategoryNode } from "@/lib/api/contracts/catalog";
import { siteConfig } from "@/lib/config/site";

function flattenCategories(
  categories: CategoryNode[],
): CategoryNode[] {
  return categories.flatMap(
    (category) => [
      category,
      ...flattenCategories(
        category.children ?? [],
      ),
    ],
  );
}

export default async function sitemap(): Promise<MetadataRoute.Sitemap> {
  const home: MetadataRoute.Sitemap[number] = {
    url: siteConfig.url,
    changeFrequency: "daily",
    priority: 1,
  };

  try {
    const categories =
      await getCategoryTree();

    return [
      home,
      ...flattenCategories(categories).map(
        (category) => ({
          url:
            `${siteConfig.url}/category/${category.slug}`,
          changeFrequency:
            "daily" as const,
          priority: 0.8,
        }),
      ),
    ];
  } catch {
    /*
     * A temporary catalog outage should not
     * make the sitemap route fail completely.
     */
    return [home];
  }
}
