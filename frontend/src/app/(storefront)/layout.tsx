import type {
  Metadata,
  Viewport,
} from "next";

import type {
  ReactNode,
} from "react";

import {
  SiteHeader,
} from "@/components/layout/components/SiteHeader";

import {
  MobileBottomNav,
} from "@/components/navigation/components/MobileBottomNav";

import {
  DesktopCustomerLauncher,
} from "@/components/navigation/components/DesktopCustomerLauncher";

import {
  StorefrontInstallPrompt,
} from "@/components/pwa/components/StorefrontInstallPrompt";

import {
  siteConfig,
} from "@/lib/config/site";

export const metadata:
  Metadata = {
  metadataBase:
    new URL(
      siteConfig.url,
    ),

  applicationName:
    siteConfig.name,

  title: {
    default:
      `${siteConfig.name} — ${siteConfig.slogan}`,

    template:
      `%s | ${siteConfig.name}`,
  },

  description:
    siteConfig.description,

  manifest:
    "/manifest.webmanifest",

  icons: {
  icon: [
    {
      url: "/favicon.png",
      type: "image/png",
    },

    {
      url: "/icons/pwa/icon-192.png",
      sizes: "192x192",
      type: "image/png",
    },

    {
      url: "/icons/pwa/icon-512.png",
      sizes: "512x512",
      type: "image/png",
    },
  ],

  apple: [
    {
      url: "/icons/pwa/apple-touch-icon.png",
      sizes: "180x180",
      type: "image/png",
    },
  ],
},

  appleWebApp: {
    capable:
      true,

    title:
      siteConfig.name,

    statusBarStyle:
      "default",
  },

  robots: {
    index:
      true,

    follow:
      true,
  },
};

export const viewport:
  Viewport = {
  width:
    "device-width",

  initialScale:
    1,

  viewportFit:
    "cover",

  colorScheme:
    "light",

  themeColor:
    siteConfig.colors.red,
};

type Props = {
  children:
    ReactNode;
};

export default function StorefrontLayout({
  children,
}: Props) {
  return (
    <>
      <a
        className="skip-link"
        href="#main-content"
      >
        Skip to content
      </a>

      <div
        className="app-shell"
      >
        <SiteHeader />

        <main
          id="main-content"
          className="site-main"
        >
          {children}
        </main>

        <DesktopCustomerLauncher />

        <StorefrontInstallPrompt />

        <MobileBottomNav />
      </div>
    </>
  );
}