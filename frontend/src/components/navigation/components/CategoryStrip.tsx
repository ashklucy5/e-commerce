"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import {
  type MouseEvent as ReactMouseEvent,
  type PointerEvent as ReactPointerEvent,
  useEffect,
  useMemo,
  useRef,
} from "react";

import type { CategoryNode } from "@/lib/api/contracts/catalog";

import styles from "../css/CategoryStrip.module.css";

type Props = {
  categories: CategoryNode[];
};

function activeSlugFromPath(pathname: string) {
  if (!pathname.startsWith("/category/")) {
    return null;
  }

  const slug = pathname.split("/")[2];

  if (!slug) {
    return null;
  }

  try {
    return decodeURIComponent(slug);
  } catch {
    return slug;
  }
}

export function CategoryStrip({ categories }: Props) {
  const pathname = usePathname();
  const railRef = useRef<HTMLDivElement | null>(null);
  const activeRef = useRef<HTMLAnchorElement | null>(null);
  const didDragRef = useRef(false);
  const dragRef = useRef({
    pointerId: null as number | null,
    startX: 0,
    startScrollLeft: 0,
    dragging: false,
  });

  const activeSlug = useMemo(() => activeSlugFromPath(pathname), [pathname]);

  useEffect(() => {
    const rail = railRef.current;
    const active = activeRef.current;

    if (!rail || !active) {
      return;
    }

    const railRect = rail.getBoundingClientRect();
    const activeRect = active.getBoundingClientRect();
    const activeCenter =
      activeRect.left - railRect.left + rail.scrollLeft + activeRect.width / 2;
    const left = Math.max(0, activeCenter - rail.clientWidth / 2);
    const reducedMotion = window.matchMedia(
      "(prefers-reduced-motion: reduce)",
    ).matches;

    rail.scrollTo({ left, behavior: reducedMotion ? "auto" : "smooth" });
  }, [activeSlug]);

  function handlePointerDown(event: ReactPointerEvent<HTMLDivElement>) {
    if (event.pointerType !== "mouse" || event.button !== 0) {
      return;
    }

    const rail = railRef.current;

    if (!rail) {
      return;
    }

    didDragRef.current = false;
    dragRef.current = {
      pointerId: event.pointerId,
      startX: event.clientX,
      startScrollLeft: rail.scrollLeft,
      dragging: false,
    };
  }

  function handlePointerMove(event: ReactPointerEvent<HTMLDivElement>) {
    const rail = railRef.current;
    const drag = dragRef.current;

    if (
      !rail ||
      event.pointerType !== "mouse" ||
      drag.pointerId !== event.pointerId ||
      (event.buttons & 1) === 0
    ) {
      return;
    }

    const delta = event.clientX - drag.startX;

    if (!drag.dragging && Math.abs(delta) < 7) {
      return;
    }

    if (!drag.dragging) {
      drag.dragging = true;
      didDragRef.current = true;
      rail.dataset.dragging = "true";
      rail.setPointerCapture(event.pointerId);
    }

    event.preventDefault();
    rail.scrollLeft = drag.startScrollLeft - delta;
  }

  function handleClickCapture(event: ReactMouseEvent<HTMLDivElement>) {
    if (!didDragRef.current) {
      return;
    }

    event.preventDefault();
    event.stopPropagation();
    didDragRef.current = false;
  }

  function endPointer(event: ReactPointerEvent<HTMLDivElement>) {
    const rail = railRef.current;

    if (!rail || dragRef.current.pointerId !== event.pointerId) {
      return;
    }

    if (rail.hasPointerCapture(event.pointerId)) {
      rail.releasePointerCapture(event.pointerId);
    }

    rail.dataset.dragging = "false";
    dragRef.current.pointerId = null;
    dragRef.current.dragging = false;
  }

  if (categories.length === 0) {
    return null;
  }

  return (
    <nav className={styles.shell} aria-label="Product categories">
      <div className="site-container">
        <div
          ref={railRef}
          className={styles.rail}
          data-dragging="false"
          onClickCapture={handleClickCapture}
          onPointerDown={handlePointerDown}
          onPointerMove={handlePointerMove}
          onPointerUp={endPointer}
          onPointerCancel={endPointer}
        >
          {categories.map((category) => {
            const active = activeSlug === category.slug;

            return (
              <Link
                key={category.id}
                ref={active ? activeRef : undefined}
                href={`/category/${category.slug}`}
                className={`${styles.item} ${active ? styles.active : ""}`}
                aria-current={active ? "page" : undefined}
                draggable={false}
              >
                {category.name}
              </Link>
            );
          })}
        </div>
      </div>
    </nav>
  );
}
