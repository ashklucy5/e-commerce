"use client";

import {
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";

import {
  adminFetch,
  AdminRequestError,
} from "@/lib/admin/api";

import type {
  AdminCategory,
  AdminCategoryTreeResponse,
} from "@/lib/admin/types";

import CreateCategoryDialog from "./CreateCategoryDialog";

import styles from "../css/ProductCategoryPicker.module.css";

type Props = {
  value: string;
  onChange: (
    categoryId: string,
  ) => void;
  disabled?: boolean;
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

  return "Categories could not be loaded.";
}

function categoryChildren(
  category:
    | AdminCategory
    | null
    | undefined,
): AdminCategory[] {
  return (
    category?.children ??
    []
  );
}

function findCategory(
  tree: AdminCategory[],
  id: string,
): AdminCategory | null {
  for (const category of tree) {
    if (
      category.id === id
    ) {
      return category;
    }

    const child =
      findCategory(
        categoryChildren(
          category,
        ),
        id,
      );

    if (child) {
      return child;
    }
  }

  return null;
}

function findPath(
  tree: AdminCategory[],
  id: string,
  currentPath: string[] = [],
): string[] | null {
  for (const category of tree) {
    const nextPath = [
      ...currentPath,
      category.id,
    ];

    if (
      category.id === id
    ) {
      return nextPath;
    }

    const childPath =
      findPath(
        categoryChildren(
          category,
        ),
        id,
        nextPath,
      );

    if (childPath) {
      return childPath;
    }
  }

  return null;
}

function buildLevels(
  tree: AdminCategory[],
  selectedPath: string[],
): AdminCategory[][] {
  const levels:
    AdminCategory[][] = [];

  let currentLevel =
    tree;

  levels.push(
    currentLevel,
  );

  for (
    let index = 0;
    index <
    selectedPath.length;
    index += 1
  ) {
    const selectedId =
      selectedPath[index];

    const selected =
      currentLevel.find(
        (category) =>
          category.id ===
          selectedId,
      );

    if (!selected) {
      break;
    }

    const children =
      categoryChildren(
        selected,
      );

    if (
      children.length === 0
    ) {
      break;
    }

    currentLevel =
      children;

    levels.push(
      currentLevel,
    );
  }

  return levels;
}

function categoryPathNames(
  tree: AdminCategory[],
  selectedPath: string[],
): string[] {
  const names:
    string[] = [];

  for (
    let index = 0;
    index <
    selectedPath.length;
    index += 1
  ) {
    const category =
      findCategory(
        tree,
        selectedPath[index],
      );

    if (category) {
      names.push(
        category.name,
      );
    }
  }

  return names;
}

export default function ProductCategoryPicker({
  value,
  onChange,
  disabled = false,
}: Props) {
  const [
    tree,
    setTree,
  ] =
    useState<AdminCategory[]>(
      [],
    );

  const [
    selectedPath,
    setSelectedPath,
  ] = useState<string[]>(
    [],
  );

  const [
    loading,
    setLoading,
  ] = useState(true);

  const [
    error,
    setError,
  ] = useState("");

  const [
    createOpen,
    setCreateOpen,
  ] = useState(false);

  /*
   * Preserve the incoming category
   * value for the initial async tree
   * load without making the effect
   * re-run every time this picker
   * itself calls onChange().
   */
  const initialValueRef =
    useRef(value);

  useEffect(() => {
    let cancelled =
      false;

    adminFetch<AdminCategoryTreeResponse>(
      "/categories/tree",
    )
      .then(
        (response) => {
          if (cancelled) {
            return;
          }

          const nextTree =
            response.data;

          setTree(
            nextTree,
          );

          const initialValue =
            initialValueRef.current;

          if (
            initialValue
          ) {
            const path =
              findPath(
                nextTree,
                initialValue,
              );

            if (path) {
              setSelectedPath(
                path,
              );
            }
          }
        },
      )
      .catch(
        (value) => {
          if (cancelled) {
            return;
          }

          setError(
            getErrorMessage(
              value,
            ),
          );
        },
      )
      .finally(() => {
        if (!cancelled) {
          setLoading(
            false,
          );
        }
      });

    return () => {
      cancelled = true;
    };
  }, []);

  async function loadTree(): Promise<
    AdminCategory[]
  > {
    setLoading(true);
    setError("");

    try {
      const response =
        await adminFetch<AdminCategoryTreeResponse>(
          "/categories/tree",
        );

      setTree(
        response.data,
      );

      return response.data;
    } catch (value) {
      setError(
        getErrorMessage(
          value,
        ),
      );

      return [];
    } finally {
      setLoading(false);
    }
  }

  const levels =
    useMemo(
      () =>
        buildLevels(
          tree,
          selectedPath,
        ),
      [
        tree,
        selectedPath,
      ],
    );

  const selectedId =
    selectedPath[
      selectedPath.length - 1
    ] ?? "";

  const selectedCategory =
    selectedId
      ? findCategory(
          tree,
          selectedId,
        )
      : null;

  const selectedNames =
    useMemo(
      () =>
        categoryPathNames(
          tree,
          selectedPath,
        ),
      [
        tree,
        selectedPath,
      ],
    );

  const selectedLabel =
    selectedNames.join(
      " › ",
    );

  function selectFromPath(
    nextPath: string[],
  ) {
    setSelectedPath(
      nextPath,
    );

    const nextId =
      nextPath[
        nextPath.length - 1
      ];

    if (!nextId) {
      onChange("");
      return;
    }

    const category =
      findCategory(
        tree,
        nextId,
      );

    if (
      category
        ?.product_code_ready
    ) {
      onChange(
        category.id,
      );
    } else {
      onChange("");
    }
  }

  function handleLevelChange(
    levelIndex: number,
    categoryId: string,
  ) {
    const basePath =
      selectedPath.slice(
        0,
        levelIndex,
      );

    if (!categoryId) {
      selectFromPath(
        basePath,
      );

      return;
    }

    selectFromPath([
      ...basePath,
      categoryId,
    ]);
  }

  async function handleCreated(
    category: AdminCategory,
  ) {
    const nextTree =
      await loadTree();

    if (
      nextTree.length === 0
    ) {
      return;
    }

    const path =
      findPath(
        nextTree,
        category.id,
      );

    if (path) {
      setSelectedPath(
        path,
      );
    }

    if (
      category.product_code_ready
    ) {
      onChange(
        category.id,
      );
    } else {
      onChange("");
    }
  }

  const defaultParentId =
    selectedCategory?.id ??
    "";

  return (
    <section
      className={
        styles.picker
      }
    >
      <div
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
            Catalog placement
          </p>

          <h3
            className={
              styles.title
            }
          >
            Category
          </h3>
        </div>

        <button
          type="button"
          className={
            styles.createButton
          }
          disabled={
            disabled ||
            loading
          }
          onClick={() => {
            setCreateOpen(
              true,
            );
          }}
        >
          + New category
        </button>
      </div>

      {loading ? (
        <div
          className={
            styles.message
          }
        >
          Loading categories…
        </div>
      ) : null}

      {!loading &&
      error ? (
        <div
          className={
            styles.error
          }
          role="alert"
        >
          <span>
            {error}
          </span>

          <button
            type="button"
            disabled={disabled}
            onClick={() => {
              void loadTree();
            }}
          >
            Retry
          </button>
        </div>
      ) : null}

      {!loading &&
      !error ? (
        <div
          className={
            styles.levels
          }
        >
          {levels.map(
            (
              categories,
              levelIndex,
            ) => {
              const currentValue =
                selectedPath[
                  levelIndex
                ] ??
                "";

              const label =
                levelIndex ===
                0
                  ? "Category"
                  : levelIndex ===
                      1
                    ? "Subcategory"
                    : `Level ${
                        levelIndex +
                        1
                      }`;

              return (
                <div
                  key={
                    levelIndex
                  }
                  className={
                    styles.field
                  }
                >
                  <label
                    className={
                      styles.label
                    }
                    htmlFor={`product-category-level-${levelIndex}`}
                  >
                    {label}
                  </label>

                  <select
                    id={`product-category-level-${levelIndex}`}
                    className={
                      styles.select
                    }
                    value={
                      currentValue
                    }
                    disabled={
                      disabled
                    }
                    onChange={(
                      event,
                    ) => {
                      handleLevelChange(
                        levelIndex,
                        event.target
                          .value,
                      );
                    }}
                  >
                    <option value="">
                      Choose{" "}
                      {label.toLowerCase()}
                    </option>

                    {categories.map(
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
                            category.name
                          }
                        </option>
                      ),
                    )}
                  </select>
                </div>
              );
            },
          )}
        </div>
      ) : null}

      {!loading &&
      !error &&
      selectedCategory ? (
        <div
          className={
            selectedCategory
              .product_code_ready
              ? styles.readyState
              : styles.pendingState
          }
        >
          <div>
            <strong>
              {selectedLabel}
            </strong>

            <p>
              {selectedCategory
                .product_code_ready
                ? "Ready for products"
                : categoryChildren(
                      selectedCategory,
                    )
                      .length >
                    0
                  ? "Choose a subcategory that accepts products."
                  : "This category needs a product-code namespace before products can be added directly here."}
            </p>
          </div>

          {selectedCategory
            .product_code_prefix ? (
            <span
              className={
                styles.prefix
              }
            >
              {
                selectedCategory.product_code_prefix
              }
            </span>
          ) : null}
        </div>
      ) : null}

      {!loading &&
      !error &&
      tree.length === 0 ? (
        <div
          className={
            styles.message
          }
        >
          No categories exist
          yet. Create the first
          category to continue.
        </div>
      ) : null}

      {createOpen ? (
        <CreateCategoryDialog
          tree={tree}
          defaultParentId={
            defaultParentId
          }
          onClose={() => {
            setCreateOpen(
              false,
            );
          }}
          onCreated={
            handleCreated
          }
        />
      ) : null}
    </section>
  );
}