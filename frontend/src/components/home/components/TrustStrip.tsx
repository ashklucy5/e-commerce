"use client";

import {
  useRef,
} from "react";

import { Icon } from "@/components/ui/Icon";

import styles from "../css/TrustStrip.module.css";

const items = [
  {
    icon: "orders" as const,
    title: "MOQ friendly",
    text: "Product-level minimums",
  },
  {
    icon: "delivery" as const,
    title: "Order fulfillment",
    text: "Reliable delivery handoff",
  },
  {
    icon: "secureCheckout" as const,
    title: "Secure payments",
    text: "Protected checkout",
  },
  {
    icon: "support" as const,
    title: "Dedicated support",
    text: "Help for your business",
  },
];

export function TrustStrip() {
  const railRef =
    useRef<HTMLDivElement | null>(
      null,
    );

  const dragRef =
    useRef({
      active: false,
      pointerId: -1,
      startX: 0,
      startScrollLeft: 0,
      moved: false,
    });

  function handlePointerDown(
    event:
      React.PointerEvent<HTMLDivElement>,
  ) {
    /*
     * Touch and pen already get high-quality
     * native momentum scrolling.
     *
     * We only implement custom dragging for a
     * mouse, including Chrome responsive mode.
     */
    if (
      event.pointerType !==
        "mouse" ||
      event.button !== 0
    ) {
      return;
    }

    const rail =
      railRef.current;

    if (!rail) {
      return;
    }

    dragRef.current = {
      active: true,
      pointerId:
        event.pointerId,
      startX:
        event.clientX,
      startScrollLeft:
        rail.scrollLeft,
      moved: false,
    };

    rail.setPointerCapture(
      event.pointerId,
    );

    rail.dataset.dragging =
      "true";
  }

  function handlePointerMove(
    event:
      React.PointerEvent<HTMLDivElement>,
  ) {
    const rail =
      railRef.current;

    const drag =
      dragRef.current;

    if (
      !rail ||
      !drag.active ||
      drag.pointerId !==
        event.pointerId
    ) {
      return;
    }

    const distance =
      event.clientX -
      drag.startX;

    if (
      Math.abs(distance) >
      3
    ) {
      drag.moved =
        true;
    }

    rail.scrollLeft =
      drag.startScrollLeft -
      distance;
  }

  function finishDrag(
    event:
      React.PointerEvent<HTMLDivElement>,
  ) {
    const rail =
      railRef.current;

    const drag =
      dragRef.current;

    if (
      !rail ||
      !drag.active ||
      drag.pointerId !==
        event.pointerId
    ) {
      return;
    }

    drag.active =
      false;

    delete rail.dataset
      .dragging;

    if (
      rail.hasPointerCapture(
        event.pointerId,
      )
    ) {
      rail.releasePointerCapture(
        event.pointerId,
      );
    }
  }

  return (
    <section
      className={
        styles.shell
      }
      aria-label="Business shopping benefits"
    >
      <div
        ref={railRef}
        className={
          styles.rail
        }
        onPointerDown={
          handlePointerDown
        }
        onPointerMove={
          handlePointerMove
        }
        onPointerUp={
          finishDrag
        }
        onPointerCancel={
          finishDrag
        }
      >
        {items.map(
          (item) => (
            <article
              className={
                styles.item
              }
              key={
                item.title
              }
            >
              <span
                className={
                  styles.icon
                }
                aria-hidden="true"
              >
                <Icon
                  name={
                    item.icon
                  }
                  size={18}
                />
              </span>

              <span
                className={
                  styles.copy
                }
              >
                <strong>
                  {
                    item.title
                  }
                </strong>

                <small>
                  {item.text}
                </small>
              </span>
            </article>
          ),
        )}
      </div>
    </section>
  );
}