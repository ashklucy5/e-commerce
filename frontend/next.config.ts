import type { NextConfig } from "next";

const isDevelopment =
  process.env.NODE_ENV === "development";

const nextConfig: NextConfig = {
  images: {
    remotePatterns: [
      {
        protocol: "https",
        hostname: "placehold.co",
        pathname: "/**",
      },
      {
        protocol: "https",
        hostname: "loremflickr.com",
        pathname: "/**",
      },

      /*
       * Ene Dei product CDN.
       *
       * Product images uploaded to Backblaze
       * are publicly served through ImageKit.
       */
      {
        protocol: "https",
        hostname: "ik.imagekit.io",
        pathname: "/o9vicqo4r/**",
      },

      {
        protocol: "http",
        hostname: "localhost",
        port: "9000",
        pathname: "/commerce/**",
      },
    ],

    formats: [
      "image/avif",
      "image/webp",
    ],

    dangerouslyAllowLocalIP:
      isDevelopment,
  },
};

export default nextConfig;