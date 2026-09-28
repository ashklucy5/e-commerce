import type { Metadata } from "next";
import Link from "next/link";

import { Icon } from "@/components/ui/Icon";

export const metadata: Metadata = {
  title: "AI Shopping",

  description:
    "AI-assisted product discovery for Ene dei business buyers.",

  robots: {
    index: false,
    follow: true,
  },
};

export default function AIPage() {
  return (
    <div className="ai-page">
      <div className="site-container">
        <section className="ai-page__content">
          <span className="ai-page__icon">
            <Icon
              name="aiAssistant"
              size={34}
            />
          </span>

          <p className="catalog-page__eyebrow">
            Ene dei intelligence
          </p>

          <h1>
            Smarter product sourcing is
            coming.
          </h1>

          <p className="ai-page__description">
            Our AI-assisted product
            discovery experience will help
            business buyers find suitable
            products, compare options, and
            source more efficiently.
          </p>

          <Link
            href="/search"
            className="ai-page__action"
          >
            Browse products now
          </Link>
        </section>
      </div>
    </div>
  );
}