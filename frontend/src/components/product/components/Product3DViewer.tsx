"use client";

import {
  createElement,
} from "react";

import Script from "next/script";

import type {
  ProductModel3DMedia,
} from "@/lib/api/contracts/catalog";

import styles from "../css/ProductMediaGallery.module.css";

type Props = {
  productName: string;

  media:
    ProductModel3DMedia;
};

export function Product3DViewer({
  productName,
  media,
}: Props) {
  return (
    <>
      <Script
        id="ene-dei-model-viewer"
        type="module"
        strategy="lazyOnload"
        src="https://unpkg.com/@google/model-viewer/dist/model-viewer.min.js"
      />

      <div
        className={
          styles.modelViewer
        }
      >
        {createElement(
          "model-viewer",
          {
            src:
              media.glb_url,

            poster:
              media.poster_url ??
              undefined,

            alt:
              `${productName} 3D model`,

            "camera-controls":
              "",

            "auto-rotate":
              "",

            "shadow-intensity":
              "1",

            exposure:
              "1",

            "interaction-prompt":
              "auto",

            "touch-action":
              "pan-y",

            style: {
              width:
                "100%",

              height:
                "100%",

              background:
                "transparent",
            },
          },
        )}

        <div
          className={
            styles.interactionHint
          }
        >
          <span
            className={
              styles.rotateIcon
            }
            aria-hidden="true"
          >
            ↻
          </span>

          Drag to rotate
        </div>
      </div>
    </>
  );
}