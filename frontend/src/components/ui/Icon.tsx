import type { CSSProperties } from "react";

const iconPaths = {
  home:
    "/icons/navigation/home.svg",

  homeFilled:
    "/icons/navigation/home-filled.svg",

  categories:
    "/icons/navigation/categories.svg",

  search:
    "/icons/navigation/search.svg",

  heart:
    "/icons/navigation/heart.svg",

  heartFilled:
    "/icons/navigation/heart-filled.svg",

  cart:
    "/icons/navigation/cart.svg",

  cartFilled:
    "/icons/navigation/cart-filled.svg",

  account:
    "/icons/navigation/account.svg",

  settings:
    "/icons/navigation/settings.svg",

  request:
    "/icons/navigation/request.svg",

  bell:
    "/icons/navigation/bell.svg",

  menu:
    "/icons/navigation/menu.svg",

  filter:
    "/icons/commerce/filter.svg",

  sort:
    "/icons/commerce/sort.svg",

  coupon:
    "/icons/commerce/coupon.svg",

  orders:
    "/icons/commerce/orders.svg",

  address:
    "/icons/commerce/address.svg",

  secureCheckout:
    "/icons/commerce/secure-checkout.svg",

  delivery:
    "/icons/commerce/delivery.svg",

  returns:
    "/icons/commerce/returns.svg",

  support:
    "/icons/commerce/support.svg",

  review:
    "/icons/commerce/review-star.svg",

  /*
   * These remain registered for existing
   * components. We are not exposing the AI
   * storefront experience in this phase.
   */
  aiAssistant:
    "/icons/product/ai-assistant.svg",

  ar:
    "/icons/product/ar.svg",

  compare:
    "/icons/product/compare.svg",

  share:
    "/icons/product/share.svg",

  sizeGuide:
    "/icons/product/size-guide.svg",

  tryOn:
    "/icons/product/try-on.svg",

  view360:
    "/icons/product/view-360.svg",

  view3d:
    "/icons/product/view-3d.svg",

  zoom:
    "/icons/product/zoom.svg",

  arrowLeft:
    "/icons/utility/arrow-left.svg",

  chevronLeft:
    "/icons/utility/chevron-left.svg",

  chevronRight:
    "/icons/utility/chevron-right.svg",

  close:
    "/icons/utility/close.svg",

  minus:
    "/icons/utility/minus.svg",

  more:
    "/icons/utility/more.svg",

  plus:
    "/icons/utility/plus.svg",

  imageSearch:
    "/icons/utility/image-search.svg",
} as const;

export type IconName =
  keyof typeof iconPaths;

type IconProps = {
  name: IconName;
  size?: number;
  className?: string;
  label?: string;
};

export function Icon({
  name,
  size = 24,
  className = "",
  label,
}: IconProps) {
  const src =
    iconPaths[name];

  const style = {
    width: size,
    height: size,

    WebkitMaskImage:
      `url("${src}")`,

    maskImage:
      `url("${src}")`,

    WebkitMaskRepeat:
      "no-repeat",

    maskRepeat:
      "no-repeat",

    WebkitMaskPosition:
      "center",

    maskPosition:
      "center",

    WebkitMaskSize:
      "contain",

    maskSize:
      "contain",

    backgroundColor:
      "currentColor",

    flex:
      "0 0 auto",
  } satisfies CSSProperties;

  return (
    <span
      className={
        className
      }
      style={style}
      role={
        label
          ? "img"
          : undefined
      }
      aria-label={
        label
      }
      aria-hidden={
        label
          ? undefined
          : true
      }
    />
  );
}