"use client";

import {
  useEffect,
  useMemo,
} from "react";

import AdminButton from "@/components/admin/ui/components/AdminButton";
import AdminCard from "@/components/admin/ui/components/AdminCard";
import AdminSectionHeader from "@/components/admin/ui/components/AdminSectionHeader";

import styles from "../css/ProductMediaUploader.module.css";

export type ProductMediaItem = {
  id: string;
  file: File;
  previewUrl: string;
  altText: string;
  isPrimary: boolean;
};

type ProductMediaUploaderProps = {
  items: ProductMediaItem[];

  disabled?: boolean;

  onChange: (
    items: ProductMediaItem[],
  ) => void;
};

const ALLOWED_TYPES = new Set([
  "image/jpeg",
  "image/png",
  "image/webp",
  "image/avif",
]);

const MAX_SIZE =
  15 * 1024 * 1024;

function makeId(): string {
  return crypto.randomUUID();
}

export default function ProductMediaUploader({
  items,
  disabled = false,
  onChange,
}: ProductMediaUploaderProps) {
  const previews = useMemo(
    () =>
      new Set(
        items.map(
          (item) => item.previewUrl,
        ),
      ),
    [items],
  );

  useEffect(() => {
    return () => {
      previews.forEach((url) => {
        URL.revokeObjectURL(url);
      });
    };
  }, [previews]);

  function addFiles(
    files: FileList | null,
  ) {
    if (!files) {
      return;
    }

    const accepted =
      Array.from(files).filter(
        (file) =>
          ALLOWED_TYPES.has(
            file.type,
          ) &&
          file.size <= MAX_SIZE,
      );

    if (accepted.length === 0) {
      return;
    }

    const alreadyHasPrimary =
      items.some(
        (item) => item.isPrimary,
      );

    const nextItems =
      accepted.map(
        (file, index) => ({
          id: makeId(),
          file,
          previewUrl:
            URL.createObjectURL(file),
          altText: "",
          isPrimary:
            !alreadyHasPrimary &&
            index === 0,
        }),
      );

    onChange([
      ...items,
      ...nextItems,
    ]);
  }

  function updateItem(
    id: string,
    patch: Partial<ProductMediaItem>,
  ) {
    onChange(
      items.map((item) =>
        item.id === id
          ? {
              ...item,
              ...patch,
            }
          : item,
      ),
    );
  }

  function makePrimary(
    id: string,
  ) {
    onChange(
      items.map((item) => ({
        ...item,
        isPrimary:
          item.id === id,
      })),
    );
  }

  function removeItem(
    id: string,
  ) {
    const removed =
      items.find(
        (item) =>
          item.id === id,
      );

    if (removed) {
      URL.revokeObjectURL(
        removed.previewUrl,
      );
    }

    const remaining =
      items.filter(
        (item) =>
          item.id !== id,
      );

    if (
      removed?.isPrimary &&
      remaining.length > 0
    ) {
      remaining[0] = {
        ...remaining[0],
        isPrimary: true,
      };
    }

    onChange(remaining);
  }

  return (
    <AdminCard className={styles.card}>
      <AdminSectionHeader
        eyebrow="Media"
        title="Product images"
        description="Images will be uploaded to object storage and attached automatically after the product is created."
      />

      <label className={styles.dropzone}>
        <input
          className={styles.fileInput}
          type="file"
          accept="image/jpeg,image/png,image/webp,image/avif"
          multiple
          disabled={disabled}
          onChange={(event) => {
            addFiles(
              event.target.files,
            );

            event.currentTarget.value =
              "";
          }}
        />

        <span className={styles.dropIcon}>
          +
        </span>

        <span className={styles.dropTitle}>
          Add product images
        </span>

        <span className={styles.dropMeta}>
          JPG, PNG, WebP or AVIF ·
          maximum 15 MiB each
        </span>
      </label>

      {items.length > 0 ? (
        <div className={styles.gallery}>
          {items.map(
            (item, index) => (
              <article
                key={item.id}
                className={
                  styles.mediaCard
                }
              >
                <div
                  className={
                    styles.imageFrame
                  }
                >
                  {/* eslint-disable-next-line @next/next/no-img-element */}
                  <img
                    className={
                      styles.image
                    }
                    src={
                      item.previewUrl
                    }
                    alt=""
                  />

                  {item.isPrimary ? (
                    <span
                      className={
                        styles.primaryBadge
                      }
                    >
                      Primary
                    </span>
                  ) : null}
                </div>

                <div
                  className={
                    styles.mediaBody
                  }
                >
                  <p
                    className={
                      styles.fileName
                    }
                    title={
                      item.file.name
                    }
                  >
                    {item.file.name}
                  </p>

                  <input
                    className={
                      styles.altInput
                    }
                    value={
                      item.altText
                    }
                    disabled={disabled}
                    placeholder="Image alt text"
                    onChange={(
                      event,
                    ) =>
                      updateItem(
                        item.id,
                        {
                          altText:
                            event
                              .target
                              .value,
                        },
                      )
                    }
                  />

                  <div
                    className={
                      styles.mediaActions
                    }
                  >
                    {!item.isPrimary ? (
                      <AdminButton
                        type="button"
                        variant="ghost"
                        disabled={
                          disabled
                        }
                        onClick={() =>
                          makePrimary(
                            item.id,
                          )
                        }
                      >
                        Make primary
                      </AdminButton>
                    ) : (
                      <span
                        className={
                          styles.order
                        }
                      >
                        Image{" "}
                        {index + 1}
                      </span>
                    )}

                    <AdminButton
                      type="button"
                      variant="ghost"
                      disabled={
                        disabled
                      }
                      onClick={() =>
                        removeItem(
                          item.id,
                        )
                      }
                    >
                      Remove
                    </AdminButton>
                  </div>
                </div>
              </article>
            ),
          )}
        </div>
      ) : null}
    </AdminCard>
  );
}