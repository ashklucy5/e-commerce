export const siteConfig = {
  name: "Ene Dei",
  slogan: "Precision Commerce",

  description:
    "Ene Dei is a B2B commerce platform for discovering, sourcing, and ordering business-ready products with transparent availability and dependable support.",

  url:
    process.env.SITE_URL?.replace(/\/+$/, "") ||
    "http://localhost:3000",

  colors: {
    red: "#E60023",
    black: "#111113",
    white: "#FFFFFF",
    success: "#16803A",
  },

  assets: {
    logo: "/ene-dei-precision-commerce.webp",
    symbol: "/Ene-Dei-Symbol-Fixed.svg",
    logoDark: "/Ene-Dei-Logo-Fixed-Dark.svg",
  },
} as const;
