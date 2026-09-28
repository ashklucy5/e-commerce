"use client";

import Link from "next/link";

import {
  type PointerEvent as ReactPointerEvent,
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";

import { CatalogImage } from "@/components/commerce/components/CatalogImage";
import { Icon } from "@/components/ui/Icon";

import type {
  ProductImage,
} from "@/lib/api/contracts/catalog";

import styles from "../css/ProductMediaGallery.module.css";

type Props = {
  productName:
    string;

  images:
    ProductImage[];

  selectedImageID:
    string;

  onSelectImage:
    (
      imageID:
        string,
    ) => void;

  backHref:
    string;
};

const SWIPE_THRESHOLD =
  28;

export function ProductGallery({
  productName,
  images,
  selectedImageID,
  onSelectImage,
  backHref,
}: Props) {
  const orderedImages =
    useMemo(
      () =>
        [...images].sort(
          (a, b) =>
            a.sort_order -
            b.sort_order,
        ),
      [
        images,
      ],
    );

  const selectedIndex =
    useMemo(
      () => {
        const index =
          orderedImages.findIndex(
            (image) =>
              image.id ===
              selectedImageID,
          );

        return index >=
          0
          ? index
          : 0;
      },
      [
        orderedImages,
        selectedImageID,
      ],
    );

  const selectedImage =
    orderedImages[
      selectedIndex
    ];

  const [
    fullscreen,
    setFullscreen,
  ] =
    useState(false);

  const pointerID =
    useRef<
      number | null
    >(null);

  const startX =
    useRef(0);

  const startY =
    useRef(0);

  const dragging =
    useRef(false);

  const selectIndex =
    useCallback(
      (
        index:
          number,
      ) => {
        if (
          orderedImages.length ===
          0
        ) {
          return;
        }

        const normalized =
          (
            (
              index %
              orderedImages.length
            ) +
            orderedImages.length
          ) %
          orderedImages.length;

        onSelectImage(
          orderedImages[
            normalized
          ].id,
        );
      },
      [
        onSelectImage,
        orderedImages,
      ],
    );

  const previous =
    useCallback(
      () =>
        selectIndex(
          selectedIndex -
            1,
        ),
      [
        selectIndex,
        selectedIndex,
      ],
    );

  const next =
    useCallback(
      () =>
        selectIndex(
          selectedIndex +
            1,
        ),
      [
        selectIndex,
        selectedIndex,
      ],
    );

  function pointerDown(
    event:
      ReactPointerEvent<HTMLDivElement>,
  ) {
    if (
      !event.isPrimary
    ) {
      return;
    }

    const target =
      event.target as
        HTMLElement;

    /*
     * Keep controls clickable instead of
     * turning them into gallery swipes.
     */
    if (
      target.closest(
        "button, a",
      )
    ) {
      return;
    }

    pointerID.current =
      event.pointerId;

    startX.current =
      event.clientX;

    startY.current =
      event.clientY;

    dragging.current =
      false;
  }

  function pointerMove(
    event:
      ReactPointerEvent<HTMLDivElement>,
  ) {
    if (
      pointerID.current !==
      event.pointerId
    ) {
      return;
    }

    const dx =
      event.clientX -
      startX.current;

    const dy =
      event.clientY -
      startY.current;

    if (
      !dragging.current &&
      Math.abs(dx) >
        9 &&
      Math.abs(dx) >
        Math.abs(dy)
    ) {
      dragging.current =
        true;

      event.currentTarget
        .setPointerCapture(
          event.pointerId,
        );
    }

    if (
      dragging.current
    ) {
      event.preventDefault();
    }
  }

  function pointerUp(
    event:
      ReactPointerEvent<HTMLDivElement>,
  ) {
    if (
      pointerID.current !==
      event.pointerId
    ) {
      return;
    }

    const dx =
      event.clientX -
      startX.current;

    if (
      dragging.current &&
      Math.abs(dx) >=
        SWIPE_THRESHOLD
    ) {
      if (dx < 0) {
        next();
      } else {
        previous();
      }
    }

    if (
      event.currentTarget
        .hasPointerCapture(
          event.pointerId,
        )
    ) {
      event.currentTarget
        .releasePointerCapture(
          event.pointerId,
        );
    }

    pointerID.current =
      null;

    dragging.current =
      false;
  }

  function pointerCancel(
    event:
      ReactPointerEvent<HTMLDivElement>,
  ) {
    pointerID.current =
      null;

    dragging.current =
      false;

    if (
      event.currentTarget
        .hasPointerCapture(
          event.pointerId,
        )
    ) {
      event.currentTarget
        .releasePointerCapture(
          event.pointerId,
        );
    }
  }

  useEffect(
    () => {
      if (
        !fullscreen
      ) {
        return;
      }

      function keyDown(
        event:
          KeyboardEvent,
      ) {
        if (
          event.key ===
          "Escape"
        ) {
          setFullscreen(
            false,
          );
        }

        if (
          event.key ===
          "ArrowLeft"
        ) {
          previous();
        }

        if (
          event.key ===
          "ArrowRight"
        ) {
          next();
        }
      }

      window.addEventListener(
        "keydown",
        keyDown,
      );

      return () => {
        window.removeEventListener(
          "keydown",
          keyDown,
        );
      };
    },
    [
      fullscreen,
      next,
      previous,
    ],
  );

  if (
    !selectedImage
  ) {
    return (
      <div
        className={
          styles.empty
        }
      >
        <Link
          href={
            backHref
          }
          className={
            styles.backButton
          }
          aria-label="Go back"
        >
          <Icon
            name="chevronLeft"
            size={18}
          />
        </Link>

        <span
          aria-hidden="true"
        >
          {productName
            .charAt(0)
            .toUpperCase()}
        </span>

        <small>
          Product image
          unavailable
        </small>
      </div>
    );
  }

  return (
    <>
      <div
        className={
          styles.gallery
        }
      >
        {orderedImages.length >
        1 ? (
          <div
            className={
              styles.thumbnails
            }
            aria-label="Product photos"
          >
            {orderedImages.map(
              (
                image,
                index,
              ) => {
                const active =
                  index ===
                  selectedIndex;

                return (
                  <button
                    key={
                      image.id
                    }
                    type="button"
                    className={`${styles.thumbnail} ${
                      active
                        ? styles.thumbnailActive
                        : ""
                    }`}
                    aria-label={`View product photo ${
                      index +
                      1
                    }`}
                    aria-pressed={
                      active
                    }
                    onClick={() =>
                      selectIndex(
                        index,
                      )
                    }
                  >
                    <CatalogImage
                      src={
                        image.url
                      }
                      alt=""
                      fill
                      sizes="76px"
                      className={
                        styles.thumbnailImage
                      }
                    />
                  </button>
                );
              },
            )}
          </div>
        ) : null}

        <div
          className={
            styles.stage
          }
          onPointerDown={
            pointerDown
          }
          onPointerMove={
            pointerMove
          }
          onPointerUp={
            pointerUp
          }
          onPointerCancel={
            pointerCancel
          }
        >
          <Link
            href={
              backHref
            }
            className={
              styles.backButton
            }
            aria-label="Go back"
          >
            <Icon
              name="chevronLeft"
              size={18}
            />
          </Link>

          <div
            className={
              styles.imageStage
            }
          >
            <CatalogImage
              key={
                selectedImage.id
              }
              src={
                selectedImage.url
              }
              alt={
                selectedImage.alt_text ??
                productName
              }
              fill
              priority={
                selectedIndex ===
                0
              }
              sizes="(max-width: 48rem) 100vw, (max-width: 64rem) 56vw, 47vw"
              className={
                styles.mainImage
              }
            />
          </div>

          <button
            type="button"
            className={
              styles.zoomButton
            }
            aria-label="Open product image fullscreen"
            onClick={() =>
              setFullscreen(
                true,
              )
            }
          >
            <Icon
              name="zoom"
              size={17}
            />
          </button>
        </div>
      </div>

      {fullscreen ? (
        <div
          className={
            styles.fullscreen
          }
          role="dialog"
          aria-modal="true"
          aria-label="Product image viewer"
        >
          <button
            type="button"
            className={
              styles.fullscreenClose
            }
            aria-label="Close fullscreen image"
            onClick={() =>
              setFullscreen(
                false,
              )
            }
          >
            <Icon
              name="close"
              size={18}
            />
          </button>

          <div
            className={
              styles.fullscreenImage
            }
          >
            <CatalogImage
              src={
                selectedImage.url
              }
              alt={
                selectedImage.alt_text ??
                productName
              }
              fill
              sizes="100vw"
              className={
                styles.fullscreenImageContent
              }
            />
          </div>

          {orderedImages.length >
          1 ? (
            <div
              className={
                styles.fullscreenControls
              }
            >
              <button
                type="button"
                aria-label="Previous image"
                onClick={
                  previous
                }
              >
                <Icon
                  name="chevronLeft"
                  size={18}
                />
              </button>

              <span>
                {selectedIndex +
                  1}
                /
                {
                  orderedImages.length
                }
              </span>

              <button
                type="button"
                aria-label="Next image"
                onClick={
                  next
                }
              >
                <Icon
                  name="chevronRight"
                  size={18}
                />
              </button>
            </div>
          ) : null}
        </div>
      ) : null}
    </>
  );
}