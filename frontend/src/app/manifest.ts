import type {
  MetadataRoute,
} from "next";

import {
  siteConfig,
} from "@/lib/config/site";

export default function manifest():
  MetadataRoute.Manifest {
  return {
    name:
      siteConfig.name,

    short_name:
      "Ene Dei",

    description:
      siteConfig.description,

    start_url:
      "/",

    scope:
      "/",

    display:
      "standalone",

    background_color:
      "#FFFFFF",

    theme_color:
      siteConfig.colors.red,

    categories: [
      "shopping",
      "lifestyle",
    ],

    icons: [
      {
        src:
          "/icons/pwa/icon-192.png",

        sizes:
          "192x192",

        type:
          "image/png",

        purpose:
          "any",
      },

      {
        src:
          "/icons/pwa/icon-512.png",

        sizes:
          "512x512",

        type:
          "image/png",

        purpose:
          "any",
      },

      {
        src:
          "/icons/pwa/maskable-512.png",

        sizes:
          "512x512",

        type:
          "image/png",

        purpose:
          "maskable",
      },
    ],
  };
}