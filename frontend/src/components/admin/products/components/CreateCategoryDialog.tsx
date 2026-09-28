"use client";

import {
  type FormEvent,
  useMemo,
  useState,
} from "react";

import {
  adminFetch,
  AdminRequestError,
} from "@/lib/admin/api";

import type {
  AdminCategory,
  AdminCreateCategoryResponse,
} from "@/lib/admin/types";

import styles from "../css/CreateCategoryDialog.module.css";

type Props = {
  tree: AdminCategory[];
  defaultParentId?: string;
  onClose: () => void;
  onCreated: (
    category: AdminCategory,
  ) => void | Promise<void>;
};

type FlatCategory = {
  id: string;
  label: string;
  rootName: string;
};

function getErrorMessage(
  value: unknown,
): string {
  if (
    value instanceof
    AdminRequestError
  ) {
    return value.message;
  }

  if (value instanceof Error) {
    return value.message;
  }

  return "The category could not be created.";
}

function flattenCategories(
  categories: AdminCategory[],
  parentNames: string[] = [],
  rootName = "",
): FlatCategory[] {
  const output:
    FlatCategory[] = [];

  for (const category of categories) {
    const currentRoot =
      rootName || category.name;

    const path = [
      ...parentNames,
      category.name,
    ];

    output.push({
      id: category.id,
      label: path.join(" › "),
      rootName: currentRoot,
    });

    if (
      category.children &&
      category.children.length > 0
    ) {
      output.push(
        ...flattenCategories(
          category.children,
          path,
          currentRoot,
        ),
      );
    }
  }

  return output;
}

function codeSegment(
  value: string,
): string {
  const words =
    value
      .trim()
      .toUpperCase()
      .replace(
        /[^A-Z0-9\s]+/g,
        " ",
      )
      .split(/\s+/)
      .filter(Boolean);

  if (words.length === 0) {
    return "GEN";
  }

  if (words.length === 1) {
    return words[0]
      .replace(
        /[^A-Z0-9]/g,
        "",
      )
      .slice(0, 3)
      .padEnd(3, "X");
  }

  const first =
    words[0][0] ?? "X";

  const last =
    words[
      words.length - 1
    ]
      .replace(
        /[^A-Z0-9]/g,
        "",
      )
      .slice(0, 2)
      .padEnd(2, "X");

  return `${first}${last}`.slice(
    0,
    3,
  );
}

export default function CreateCategoryDialog({
  tree,
  defaultParentId = "",
  onClose,
  onCreated,
}: Props) {
  const flatCategories =
    useMemo(
      () =>
        flattenCategories(
          tree,
        ),
      [tree],
    );

  const [
    name,
    setName,
  ] = useState("");

  const [
    parentId,
    setParentId,
  ] = useState(
    defaultParentId,
  );

  const [
    description,
    setDescription,
  ] = useState("");

  const [
    productReady,
    setProductReady,
  ] = useState(true);

  const [
    prefix,
    setPrefix,
  ] = useState("");

  const [
    prefixEdited,
    setPrefixEdited,
  ] = useState(false);

  const [
    busy,
    setBusy,
  ] = useState(false);

  const [
    error,
    setError,
  ] = useState("");

  const suggestedPrefix =
    useMemo(() => {
      if (
        !productReady ||
        !name.trim()
      ) {
        return "";
      }

      const parent =
        flatCategories.find(
          (category) =>
            category.id ===
            parentId,
        );

      const firstSegment =
        parent
          ? codeSegment(
              parent.rootName,
            )
          : codeSegment(
              name,
            );

      const secondSegment =
        parent
          ? codeSegment(
              name,
            )
          : "GEN";

      return `${firstSegment}-${secondSegment}`;
    }, [
      flatCategories,
      name,
      parentId,
      productReady,
    ]);

  const displayedPrefix =
    prefixEdited
      ? prefix
      : suggestedPrefix;

  async function handleSubmit(
    event:
      FormEvent<HTMLFormElement>,
  ) {
    event.preventDefault();

    if (busy) {
      return;
    }

    setError("");

    const normalizedName =
      name.trim();

    if (!normalizedName) {
      setError(
        "Category name is required.",
      );

      return;
    }

    const normalizedPrefix =
      displayedPrefix
        .trim()
        .toUpperCase();

    if (
      productReady &&
      !/^[A-Z0-9]{3}-[A-Z0-9]{3}$/.test(
        normalizedPrefix,
      )
    ) {
      setError(
        "Product code prefix must use the format AAA-BBB.",
      );

      return;
    }

    setBusy(true);

    try {
      const response =
        await adminFetch<AdminCreateCategoryResponse>(
          "/categories",
          {
            method:
              "POST",

            body:
              JSON.stringify({
                name:
                  normalizedName,

                parent_id:
                  parentId ||
                  undefined,

                description:
                  description.trim(),

                product_code_prefix:
                  productReady
                    ? normalizedPrefix
                    : undefined,
              }),
          },
        );

      await onCreated(
        response.data,
      );

      onClose();
    } catch (value) {
      setError(
        getErrorMessage(
          value,
        ),
      );
    } finally {
      setBusy(false);
    }
  }

  return (
    <div
      className={
        styles.backdrop
      }
      role="presentation"
      onMouseDown={(
        event,
      ) => {
        if (
          event.target ===
          event.currentTarget
        ) {
          onClose();
        }
      }}
    >
      <section
        className={
          styles.dialog
        }
        role="dialog"
        aria-modal="true"
        aria-labelledby="create-category-title"
      >
        <header
          className={
            styles.header
          }
        >
          <div>
            <p
              className={
                styles.eyebrow
              }
            >
              Catalog structure
            </p>

            <h2
              id="create-category-title"
              className={
                styles.title
              }
            >
              New category
            </h2>

            <p
              className={
                styles.description
              }
            >
              Add a category or
              subcategory without
              leaving the product
              form.
            </p>
          </div>

          <button
            type="button"
            className={
              styles.closeButton
            }
            aria-label="Close category dialog"
            disabled={busy}
            onClick={
              onClose
            }
          >
            ×
          </button>
        </header>

        <form
          className={
            styles.form
          }
          onSubmit={
            handleSubmit
          }
        >
          <div
            className={
              styles.field
            }
          >
            <label
              className={
                styles.label
              }
              htmlFor="category-name"
            >
              Category name
            </label>

            <input
              id="category-name"
              className={
                styles.input
              }
              value={name}
              disabled={busy}
              autoFocus
              placeholder="Example: Casual Shirts"
              onChange={(
                event,
              ) => {
                setName(
                  event.target
                    .value,
                );
              }}
            />
          </div>

          <div
            className={
              styles.field
            }
          >
            <label
              className={
                styles.label
              }
              htmlFor="category-parent"
            >
              Parent category
            </label>

            <select
              id="category-parent"
              className={
                styles.select
              }
              value={
                parentId
              }
              disabled={busy}
              onChange={(
                event,
              ) => {
                setParentId(
                  event.target
                    .value,
                );
              }}
            >
              <option value="">
                No parent —
                top-level category
              </option>

              {flatCategories.map(
                (
                  category,
                ) => (
                  <option
                    key={
                      category.id
                    }
                    value={
                      category.id
                    }
                  >
                    {
                      category.label
                    }
                  </option>
                ),
              )}
            </select>
          </div>

          <div
            className={
              styles.field
            }
          >
            <label
              className={
                styles.label
              }
              htmlFor="category-description"
            >
              Description
            </label>

            <textarea
              id="category-description"
              className={
                styles.textarea
              }
              value={
                description
              }
              disabled={busy}
              rows={4}
              placeholder="Optional internal or storefront description"
              onChange={(
                event,
              ) => {
                setDescription(
                  event.target
                    .value,
                );
              }}
            />
          </div>

          <label
            className={
              styles.checkRow
            }
          >
            <input
              type="checkbox"
              checked={
                productReady
              }
              disabled={busy}
              onChange={(
                event,
              ) => {
                setProductReady(
                  event.target
                    .checked,
                );
              }}
            />

            <span>
              <strong>
                Products can be
                added directly here
              </strong>

              <small>
                Enable this when
                this category should
                receive its own
                product-code
                namespace.
              </small>
            </span>
          </label>

          {productReady ? (
            <div
              className={
                styles.field
              }
            >
              <label
                className={
                  styles.label
                }
                htmlFor="category-prefix"
              >
                Product code
                prefix
              </label>

              <input
                id="category-prefix"
                className={
                  styles.input
                }
                value={
                  displayedPrefix
                }
                disabled={busy}
                placeholder="AAA-BBB"
                maxLength={7}
                onChange={(
                  event,
                ) => {
                  setPrefixEdited(
                    true,
                  );

                  setPrefix(
                    event.target
                      .value
                      .toUpperCase(),
                  );
                }}
              />

              <p
                className={
                  styles.hint
                }
              >
                Format AAA-BBB.
                The suggestion is
                editable before
                creation.
              </p>
            </div>
          ) : null}

          {error ? (
            <p
              className={
                styles.error
              }
              role="alert"
            >
              {error}
            </p>
          ) : null}

          <footer
            className={
              styles.actions
            }
          >
            <button
              type="button"
              className={
                styles.secondaryButton
              }
              disabled={busy}
              onClick={
                onClose
              }
            >
              Cancel
            </button>

            <button
              type="submit"
              className={
                styles.primaryButton
              }
              disabled={busy}
            >
              {busy
                ? "Creating…"
                : "Create category"}
            </button>
          </footer>
        </form>
      </section>
    </div>
  );
}