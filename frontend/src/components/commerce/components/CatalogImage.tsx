import Image from "next/image";

import type {
  CSSProperties,
} from "react";

import styles from "../css/CatalogImage.module.css";

type CatalogImageProps = {
  src: string;
  alt: string;

  fill?: boolean;

  width?: number;
  height?: number;

  sizes?: string;

  /*
   * We keep the prop named "priority"
   * for our own reusable API.
   *
   * Internally on Next 16 it becomes:
   * loading="eager"
   * fetchPriority="high"
   */
  priority?: boolean;

  className?: string;

  style?: CSSProperties;

  "aria-hidden"?:
    boolean |
    "true" |
    "false";
};

const TRUSTED_REMOTE_HOSTS =
  new Set([
    "placehold.co",
    "loremflickr.com",
  ]);

function isTrustedForNextImage(
  src: string,
) {
  if (
    src.startsWith("/")
  ) {
    return true;
  }

  try {
    const url =
      new URL(src);

    if (
      TRUSTED_REMOTE_HOSTS.has(
        url.hostname.toLowerCase(),
      )
    ) {
      return true;
    }

    return (
      url.protocol ===
        "http:" &&
      url.hostname ===
        "localhost" &&
      url.port ===
        "9000" &&
      url.pathname.startsWith(
        "/commerce/",
      )
    );
  } catch {
    return false;
  }
}

export function CatalogImage({
  src,
  alt,

  fill = false,

  width,
  height,

  sizes,

  priority = false,

  className,

  style,

  "aria-hidden":
    ariaHidden,
}: CatalogImageProps) {
  /*
   * Known and controlled hosts:
   * use Next Image optimization.
   */
  if (
    isTrustedForNextImage(
      src,
    )
  ) {
    return (
      <Image
        src={src}
        alt={alt}
        fill={fill}
        width={
          fill
            ? undefined
            : width
        }
        height={
          fill
            ? undefined
            : height
        }
        sizes={sizes}
        loading={
          priority
            ? "eager"
            : "lazy"
        }
        fetchPriority={
          priority
            ? "high"
            : "auto"
        }
        className={
          className
        }
        style={style}
        aria-hidden={
          ariaHidden
        }
      />
    );
  }

  /*
   * Unknown supplier/CDN host:
   *
   * Do not crash the storefront
   * simply because Next.js did not
   * know the hostname at build time.
   */
  const fallbackStyle:
    CSSProperties =
    fill
      ? {
          ...style,

          position:
            "absolute",

          inset: 0,

          width: "100%",

          height:
            "100%",
        }
      : style ?? {};

  return (
    // eslint-disable-next-line @next/next/no-img-element
    <img
      src={src}
      alt={alt}
      width={
        fill
          ? undefined
          : width
      }
      height={
        fill
          ? undefined
          : height
      }
      sizes={sizes}
      loading={
        priority
          ? "eager"
          : "lazy"
      }
      fetchPriority={
        priority
          ? "high"
          : "auto"
      }
      decoding="async"
      referrerPolicy="no-referrer"
      className={`${styles.fallbackImage}${
        className
          ? ` ${className}`
          : ""
      }`}
      style={
        fallbackStyle
      }
      aria-hidden={
        ariaHidden
      }
    />
  );
}