import type { Metadata } from "next";
import { HomeCatalogFeed } from "@/components/home/components/HomeCatalogFeed";
import { HomeHero } from "@/components/home/components/HomeHero";
import { TrustStrip } from "@/components/home/components/TrustStrip";
import styles from "@/components/home/css/Home.module.css";
import { getHomeProductFeed, getHomePromotions } from "@/lib/api/home";
import type {
  HomeProductCard,
  HomeProductFeedResponse,
} from "@/lib/api/contracts/home";
import { siteConfig } from "@/lib/config/site";

const MAX_HERO_PRODUCTS = 4;

export const dynamic = "force-dynamic";

type PageProps = {
  searchParams: Promise<{
    page?: string | string[];
  }>;
};

const EMPTY_HOME_FEED: HomeProductFeedResponse = {
  data: [],
  meta: {
    page: 1,
    limit: 50,
    total: 0,
    total_pages: 0,
    has_next: false,
    has_previous: false,
    sort: "homepage",
  },
};

function parsePage(value?: string | string[]) {
  const raw = Array.isArray(value) ? value[0] : value;
  const page = Number.parseInt(raw ?? "1", 10);

  return Number.isFinite(page) && page > 0 ? page : 1;
}

function shuffleProducts(
  products: HomeProductCard[],
) {
  const shuffled =
    [...products];

  for (
    let index =
      shuffled.length - 1;
    index > 0;
    index -= 1
  ) {
    const target =
      Math.floor(
        Math.random() *
          (index + 1),
      );

    [
      shuffled[index],
      shuffled[target],
    ] = [
      shuffled[target],
      shuffled[index],
    ];
  }

  return shuffled;
}

function getHeroProducts(
  products: HomeProductCard[],
) {
  /*
   * Hero imagery must be real.
   *
   * Products without a primary image do not
   * enter the normal hero candidate pool.
   */
  const withImages =
    products.filter(
      (
        product,
      ) =>
        Boolean(
          product.primary_image_url,
        ),
    );

  /*
   * Rank by actual popularity first.
   *
   * The explicit "popular" merchandising group
   * gets preference when the backend supplies
   * enough candidates.
   */
  const ranked =
    [...withImages].sort(
      (a, b) => {
        const soldDifference =
          b.sold_quantity -
          a.sold_quantity;

        if (
          soldDifference !==
          0
        ) {
          return soldDifference;
        }

        return (
          Number(
            Boolean(
              b.is_featured,
            ),
          ) -
          Number(
            Boolean(
              a.is_featured,
            ),
          )
        );
      },
    );

  const popular =
    ranked.filter(
      (
        product,
      ) =>
        product.merchandising_group ===
        "popular",
    );

  /*
   * Randomize only among strong candidates,
   * rather than randomly selecting weak
   * products from the entire catalog.
   */
  const candidatePool =
    (
      popular.length >=
      MAX_HERO_PRODUCTS
        ? popular
        : ranked
    ).slice(
      0,
      12,
    );

  const selected =
    shuffleProducts(
      candidatePool,
    ).slice(
      0,
      MAX_HERO_PRODUCTS,
    );

  /*
   * Emergency fallback only if absolutely no
   * product in the response has an image.
   */
  if (
    selected.length === 0
  ) {
    return [...products]
      .sort(
        (a, b) =>
          b.sold_quantity -
          a.sold_quantity,
      )
      .slice(
        0,
        MAX_HERO_PRODUCTS,
      );
  }

  return selected;
}

export async function generateMetadata({ searchParams }: PageProps): Promise<Metadata> {
  const params = await searchParams;
  const page = parsePage(params.page);
  const canonical = page > 1 ? `/?page=${page}` : "/";
  const title = page > 1 ? `B2B Product Catalog — Page ${page}` : "B2B Wholesale Marketplace";
  const description =
    "Discover business-ready products on Ene Dei with transparent pricing, live catalog availability, secure checkout, and sourcing support for bulk product needs.";

  return {
    title,
    description,
    alternates: {
      canonical,
    },
    openGraph: {
      type: "website",
      url: canonical,
      siteName: siteConfig.name,
      title: `${title} | ${siteConfig.name}`,
      description,
    },
  };
}

export default async function HomePage({ searchParams }: PageProps) {
  const params = await searchParams;
  const page = parsePage(params.page);

  const structuredData = {
    "@context": "https://schema.org",
    "@graph": [
      {
        "@type": "Organization",
        "@id": `${siteConfig.url}/#organization`,
        name: siteConfig.name,
        url: siteConfig.url,
        logo: `${siteConfig.url}${siteConfig.assets.logo}`,
        description: siteConfig.description,
      },
      {
        "@type": "WebSite",
        "@id": `${siteConfig.url}/#website`,
        name: siteConfig.name,
        url: siteConfig.url,
        description: siteConfig.description,
        publisher: {
          "@id": `${siteConfig.url}/#organization`,
        },
      },
    ],
  };
  const jsonLD = JSON.stringify(structuredData).replace(/</g, "\\u003c");

  const [productFeed, promotions] = await Promise.all([
    getHomeProductFeed(page, 50).catch(() => ({
      ...EMPTY_HOME_FEED,
      meta: {
        ...EMPTY_HOME_FEED.meta,
        page,
      },
    })),
    getHomePromotions().catch(() => []),
  ]);

  return (
    <div className={styles.page}>
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{ __html: jsonLD }}
      />

      <div className="site-container">
        <HomeHero products={getHeroProducts(productFeed.data)} />

        <TrustStrip />

        <HomeCatalogFeed
          key={`home-catalog-page-${productFeed.meta.page}`}
          initialProducts={productFeed.data}
          initialMeta={productFeed.meta}
          promotions={promotions}
        />
      </div>
    </div>
  );
}
