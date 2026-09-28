"use client";

import Image from "next/image";

import {
  type PointerEvent,
  useMemo,
  useRef,
  useState,
} from "react";

import type {
  ProductSpin360Media,
} from "@/lib/api/contracts/catalog";

import styles from "../css/ProductMediaGallery.module.css";

type Props = {
  productName: string;

  media:
    ProductSpin360Media;
};

const PIXELS_PER_FRAME =
  10;

export function Product360Viewer({
  productName,
  media,
}: Props) {
  const frames =
    useMemo(
      () =>
        [...media.frames].sort(
          (
            left,
            right,
          ) =>
            left.frame -
            right.frame,
        ),
      [
        media.frames,
      ],
    );

  const [
    frameIndex,
    setFrameIndex,
  ] =
    useState(0);

  const dragging =
    useRef(false);

  const lastX =
    useRef(0);

  function moveBy(
    amount: number,
  ) {
    if (
      frames.length ===
      0
    ) {
      return;
    }

    setFrameIndex(
      (current) =>
        (
          current +
          amount +
          frames.length
        ) %
        frames.length,
    );
  }

  function handlePointerDown(
    event:
      PointerEvent<HTMLDivElement>,
  ) {
    dragging.current =
      true;

    lastX.current =
      event.clientX;

    event.currentTarget.setPointerCapture(
      event.pointerId,
    );
  }

  function handlePointerMove(
    event:
      PointerEvent<HTMLDivElement>,
  ) {
    if (
      !dragging.current
    ) {
      return;
    }

    const delta =
      event.clientX -
      lastX.current;

    const steps =
      Math.trunc(
        delta /
          PIXELS_PER_FRAME,
      );

    if (
      steps === 0
    ) {
      return;
    }

    moveBy(
      -steps,
    );

    lastX.current +=
      steps *
      PIXELS_PER_FRAME;
  }

  function handlePointerUp(
    event:
      PointerEvent<HTMLDivElement>,
  ) {
    dragging.current =
      false;

    if (
      event.currentTarget.hasPointerCapture(
        event.pointerId,
      )
    ) {
      event.currentTarget.releasePointerCapture(
        event.pointerId,
      );
    }
  }

  const frame =
    frames[
      frameIndex
    ];

  if (!frame) {
    return (
      <div
        className={
          styles.mediaUnavailable
        }
      >
        360° media is not
        available.
      </div>
    );
  }

  return (
    <div
      className={
        styles.spinViewer
      }
      tabIndex={0}
      role="application"
      aria-label={`${productName} 360 degree viewer`}
      onPointerDown={
        handlePointerDown
      }
      onPointerMove={
        handlePointerMove
      }
      onPointerUp={
        handlePointerUp
      }
      onPointerCancel={
        handlePointerUp
      }
      onKeyDown={(
        event,
      ) => {
        if (
          event.key ===
          "ArrowLeft"
        ) {
          event.preventDefault();

          moveBy(-1);
        }

        if (
          event.key ===
          "ArrowRight"
        ) {
          event.preventDefault();

          moveBy(1);
        }
      }}
    >
      <Image
        key={
          frame.url
        }
        src={
          frame.url
        }
        alt={`${productName} 360° view`}
        fill
        sizes="(max-width: 899px) 100vw, 52vw"
        className={
          styles.spinImage
        }
        draggable={
          false
        }
      />

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

      <div
        className={
          styles.frameCounter
        }
      >
        {frameIndex + 1}
        {" / "}
        {frames.length}
      </div>
    </div>
  );
}