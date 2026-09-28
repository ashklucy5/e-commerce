"use client";

import NextImage from "next/image";
import { useRouter } from "next/navigation";

import {
  type ChangeEvent,
  type FormEvent,
  type KeyboardEvent,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";

import {
  CatalogImage,
} from "@/components/commerce/components/CatalogImage";

import {
  Icon,
} from "@/components/ui/Icon";

import type {
  ImageSearchResponse,
} from "@/lib/api/contracts/image-search";

import type {
  SearchSuggestions,
  SearchSuggestionsResponse,
} from "@/lib/api/contracts/search-suggestions";

import {
  formatMoney,
} from "@/lib/money/format";

import styles from "../css/SearchAutocomplete.module.css";


/*
 * ============================================================
 * SEARCH CONFIGURATION
 * ============================================================
 */

const MAX_IMAGE_BYTES =
  10 * 1024 * 1024;

const ACCEPTED_IMAGE_TYPES =
  new Set([
    "image/jpeg",
    "image/png",
    "image/webp",
    "image/gif",
  ]);


const EMPTY:
  SearchSuggestions = {
  query: "",
  products: [],
  categories: [],
  brands: [],
};


type NavigationItem = {
  key: string;
  href: string;
};


type ErrorPayload = {
  error?: {
    code?: string;
    message?: string;
  };
};


function searchHref(
  query: string,
) {
  return `/search?q=${encodeURIComponent(
    query,
  )}`;
}


async function responseMessage(
  response: Response,
) {
  try {
    const payload =
      (await response.json()) as
        ErrorPayload;

    return (
      payload.error
        ?.message ??
      "Search could not be completed."
    );
  } catch {
    return "Search could not be completed.";
  }
}


/*
 * ============================================================
 * COMPONENT
 * ============================================================
 */

export function SearchAutocomplete() {
  const router =
    useRouter();


  const rootRef =
    useRef<HTMLDivElement | null>(
      null,
    );


  const inputRef =
    useRef<HTMLInputElement | null>(
      null,
    );


  const visualInputRef =
    useRef<HTMLInputElement | null>(
      null,
    );


  const [
    query,
    setQuery,
  ] =
    useState("");


  const [
    suggestions,
    setSuggestions,
  ] =
    useState<SearchSuggestions>(
      EMPTY,
    );


  const [
    open,
    setOpen,
  ] =
    useState(false);


  const [
    loading,
    setLoading,
  ] =
    useState(false);


  const [
    activeIndex,
    setActiveIndex,
  ] =
    useState(-1);


  /*
   * Visual-search state.
   */
  const [
    visualSearching,
    setVisualSearching,
  ] =
    useState(false);


  const [
    visualResult,
    setVisualResult,
  ] =
    useState<ImageSearchResponse | null>(
      null,
    );


  const [
    visualError,
    setVisualError,
  ] =
    useState("");


  /*
   * Header expansion state.
   *
   * We mirror this onto <html> so SiteHeader and
   * MobileBottomNav can react without converting
   * the server-rendered header into a client component.
   */
  const [
    searchFocused,
    setSearchFocused,
  ] =
    useState(false);


  function updateSearchFocus(
    focused: boolean,
  ) {
    setSearchFocused(
      focused,
    );

    if (
      typeof document ===
      "undefined"
    ) {
      return;
    }

    if (focused) {
      document.documentElement
        .setAttribute(
          "data-store-search-focused",
          "true",
        );
    } else {
      document.documentElement
        .removeAttribute(
          "data-store-search-focused",
        );
    }
  }


  /*
   * Cleanup global search state.
   */
  useEffect(() => {
    return () => {
      document.documentElement
        .removeAttribute(
          "data-store-search-focused",
        );
    };
  }, []);


  /*
   * Restore q= when already on /search.
   */
  useEffect(() => {
    if (
      typeof window ===
      "undefined"
    ) {
      return;
    }

    if (
      window.location.pathname !==
      "/search"
    ) {
      return;
    }

    const params =
      new URLSearchParams(
        window.location.search,
      );

    const current =
      params.get("q") ??
      "";

    queueMicrotask(
      () => {
        setQuery(
          current,
        );
      },
    );
  }, []);


  /*
   * Close the dropdown when the shopper
   * interacts outside search.
   */
  useEffect(() => {
    function handlePointerDown(
      event:
        PointerEvent,
    ) {
      const target =
        event.target as
          Node;

      if (
        rootRef.current &&
        !rootRef.current.contains(
          target,
        )
      ) {
        setOpen(
          false,
        );

        setActiveIndex(
          -1,
        );
      }
    }


    document.addEventListener(
      "pointerdown",
      handlePointerDown,
    );


    return () => {
      document.removeEventListener(
        "pointerdown",
        handlePointerDown,
      );
    };
  }, []);


  /*
   * ==========================================================
   * TEXT AUTOCOMPLETE
   * ==========================================================
   */

  useEffect(() => {
    const cleanQuery =
      query.trim();

    if (
      cleanQuery.length <
      2
    ) {
      return;
    }


    const controller =
      new AbortController();


    const timer =
      window.setTimeout(
        async () => {
          setLoading(
            true,
          );

          try {
            const parameters =
              new URLSearchParams({
                q: cleanQuery,
                limit: "6",
              });


            const response =
              await fetch(
                `/api/storefront/search/suggestions?${parameters.toString()}`,
                {
                  cache:
                    "no-store",

                  signal:
                    controller.signal,
                },
              );


            if (
              !response.ok
            ) {
              throw new Error(
                "Suggestion request failed.",
              );
            }


            const payload =
              (await response.json()) as
                SearchSuggestionsResponse;


            setSuggestions(
              payload.data ?? {
                ...EMPTY,

                query:
                  cleanQuery,
              },
            );


            setOpen(
              true,
            );


            setActiveIndex(
              -1,
            );
          } catch {
            if (
              controller.signal
                .aborted
            ) {
              return;
            }


            setSuggestions({
              ...EMPTY,

              query:
                cleanQuery,
            });


            setOpen(
              true,
            );
          } finally {
            if (
              !controller.signal
                .aborted
            ) {
              setLoading(
                false,
              );
            }
          }
        },

        180,
      );


    return () => {
      window.clearTimeout(
        timer,
      );

      controller.abort();
    };
  }, [query]);


  /*
   * ==========================================================
   * VISUAL SEARCH
   * ==========================================================
   */


  function openVisualPicker() {
    /*
     * Close text suggestions first.
     */
    setOpen(
      false,
    );

    setActiveIndex(
      -1,
    );

    setVisualError(
      "",
    );


    /*
     * Clicking this hidden input invokes the browser /
     * operating-system image picker.
     *
     * On supported mobile browsers this normally offers
     * camera and photo-library/gallery options.
     *
     * We deliberately DO NOT add capture="environment",
     * because that can force the camera and prevent the
     * user from choosing an existing gallery image.
     */
    visualInputRef.current
      ?.click();
  }


  async function handleVisualImage(
    event:
      ChangeEvent<HTMLInputElement>,
  ) {
    const selected =
      event.currentTarget
        .files?.[0] ??
      null;


    /*
     * Reset native input immediately so selecting the
     * same photo again later still fires onChange.
     */
    event.currentTarget.value =
      "";


    if (!selected) {
      return;
    }


    /*
     * Image mode replaces text suggestions.
     */
    setQuery(
      "",
    );

    setSuggestions(
      EMPTY,
    );

    setActiveIndex(
      -1,
    );

    setVisualResult(
      null,
    );

    setVisualError(
      "",
    );


    if (
      !ACCEPTED_IMAGE_TYPES.has(
        selected.type,
      )
    ) {
      setVisualError(
        "Choose a JPEG, PNG, WebP, or GIF image.",
      );

      setOpen(
        true,
      );

      return;
    }


    if (
      selected.size >
      MAX_IMAGE_BYTES
    ) {
      setVisualError(
        "Image search accepts files up to 10 MB.",
      );

      setOpen(
        true,
      );

      return;
    }


    setVisualSearching(
      true,
    );

    setOpen(
      true,
    );


    const form =
      new FormData();


    form.append(
      "image",
      selected,
      selected.name,
    );


    try {
      const response =
        await fetch(
          "/api/storefront/search/image?limit=12",
          {
            method:
              "POST",

            body:
              form,

            cache:
              "no-store",
          },
        );


      if (
        !response.ok
      ) {
        throw new Error(
          await responseMessage(
            response,
          ),
        );
      }


      const payload =
        (await response.json()) as
          ImageSearchResponse;


      setVisualResult(
        payload,
      );

      setOpen(
        true,
      );
    } catch (
      caught
    ) {
      setVisualError(
        caught instanceof
          Error
          ? caught.message
          : "Image search could not be completed.",
      );

      setOpen(
        true,
      );
    } finally {
      setVisualSearching(
        false,
      );
    }
  }


  /*
   * ==========================================================
   * KEYBOARD NAVIGATION
   * ==========================================================
   */

  const navigationItems =
    useMemo<
      NavigationItem[]
    >(() => {
      const items:
        NavigationItem[] =
        [];


      for (
        const product of
        suggestions.products
      ) {
        items.push({
          key:
            `product:${product.id}`,

          href:
            `/product/${product.slug}`,
        });
      }


      for (
        const category of
        suggestions.categories
      ) {
        items.push({
          key:
            `category:${category.slug}`,

          href:
            `/category/${category.slug}`,
        });
      }


      for (
        const brand of
        suggestions.brands
      ) {
        items.push({
          key:
            `brand:${brand}`,

          href:
            searchHref(
              brand,
            ),
        });
      }


      if (
        query.trim()
      ) {
        items.push({
          key:
            "search-all",

          href:
            searchHref(
              query.trim(),
            ),
        });
      }


      return items;
    }, [
      query,
      suggestions,
    ]);


  const categoryOffset =
    suggestions
      .products
      .length;


  const brandOffset =
    categoryOffset +
    suggestions
      .categories
      .length;


  const searchAllIndex =
    brandOffset +
    suggestions
      .brands
      .length;


  function navigate(
    href: string,
  ) {
    setOpen(
      false,
    );

    setActiveIndex(
      -1,
    );


    inputRef.current
      ?.blur();


    updateSearchFocus(
      false,
    );


    router.push(
      href,
    );
  }


  function submitSearch(
    event?:
      FormEvent<HTMLFormElement>,
  ) {
    event?.preventDefault();


    const cleanQuery =
      query.trim();


    if (
      !cleanQuery
    ) {
      inputRef.current
        ?.focus();

      return;
    }


    navigate(
      searchHref(
        cleanQuery,
      ),
    );
  }


  function handleQueryChange(
    value: string,
  ) {
    const cleanQuery =
      value.trim();


    /*
     * Typing switches away from prior visual results.
     */
    setVisualResult(
      null,
    );

    setVisualError(
      "",
    );


    setQuery(
      value,
    );


    if (
      cleanQuery.length <
      2
    ) {
      setSuggestions({
        ...EMPTY,

        query:
          cleanQuery,
      });


      setLoading(
        false,
      );


      setOpen(
        false,
      );


      setActiveIndex(
        -1,
      );
    }
  }


  function handleKeyDown(
    event:
      KeyboardEvent<HTMLInputElement>,
  ) {
    if (
      event.key ===
      "Escape"
    ) {
      setOpen(
        false,
      );

      setActiveIndex(
        -1,
      );


      inputRef.current
        ?.blur();


      updateSearchFocus(
        false,
      );

      return;
    }


    if (
      event.key ===
      "ArrowDown"
    ) {
      event.preventDefault();


      if (!open) {
        setOpen(
          true,
        );
      }


      setActiveIndex(
        (
          current,
        ) => {
          if (
            navigationItems.length ===
            0
          ) {
            return -1;
          }


          return (
            (
              current +
              1
            ) %
            navigationItems.length
          );
        },
      );


      return;
    }


    if (
      event.key ===
      "ArrowUp"
    ) {
      event.preventDefault();


      if (!open) {
        setOpen(
          true,
        );
      }


      setActiveIndex(
        (
          current,
        ) => {
          if (
            navigationItems.length ===
            0
          ) {
            return -1;
          }


          if (
            current <=
            0
          ) {
            return (
              navigationItems.length -
              1
            );
          }


          return (
            current -
            1
          );
        },
      );


      return;
    }


    if (
      event.key ===
        "Enter" &&
      open &&
      activeIndex >=
        0 &&
      navigationItems[
        activeIndex
      ]
    ) {
      event.preventDefault();


      navigate(
        navigationItems[
          activeIndex
        ].href,
      );
    }
  }


  const hasSuggestions =
    suggestions
      .products
      .length >
      0 ||
    suggestions
      .categories
      .length >
      0 ||
    suggestions
      .brands
      .length >
      0;


  const visualProducts =
    visualResult?.data ??
    [];


  const hasVisualState =
    visualSearching ||
    Boolean(
      visualError,
    ) ||
    visualResult !==
      null;


  const showDropdown =
    open &&
    (
      query
        .trim()
        .length >=
        2 ||
      hasVisualState
    );


  /*
   * ==========================================================
   * RENDER
   * ==========================================================
   */

  return (
    <div
      ref={rootRef}

      className={
        styles.root
      }

      data-search-focused={
        searchFocused
          ? "true"
          : "false"
      }

      onFocusCapture={() => {
        updateSearchFocus(
          true,
        );
      }}

      onBlurCapture={(
        event,
      ) => {
        const nextTarget =
          event.relatedTarget as
            | Node
            | null;


        if (
          nextTarget &&
          event.currentTarget.contains(
            nextTarget,
          )
        ) {
          return;
        }


        updateSearchFocus(
          false,
        );
      }}
    >
      <form
        className={
          styles.form
        }

        role="search"

        onSubmit={
          submitSearch
        }
      >
        {/*
         * TEXT SEARCH
         */}
        <button
          type="submit"

          className={
            styles.searchButton
          }

          aria-label="Search products"
        >
          <Icon
            name="search"

            size={18}
          />
        </button>


        <input
          ref={inputRef}

          type="search"

          value={query}

          className={
            styles.input
          }

          placeholder="Search products"

          autoComplete="off"

          spellCheck={
            false
          }

          enterKeyHint="search"

          aria-label="Search products"

          role="combobox"

          aria-autocomplete="list"

          aria-expanded={
            showDropdown
          }

          aria-controls="store-search-suggestions"

          aria-activedescendant={
            activeIndex >=
            0
              ? `search-option-${activeIndex}`
              : undefined
          }

          onChange={(
            event,
          ) =>
            handleQueryChange(
              event.target
                .value,
            )
          }

          onFocus={() => {
            updateSearchFocus(
              true,
            );


            if (
              query
                .trim()
                .length >=
              2 ||
              hasVisualState
            ) {
              setOpen(
                true,
              );
            }
          }}

          onKeyDown={
            handleKeyDown
          }
        />


        <div
          className={
            styles.controls
          }
        >
          {loading ? (
            <span
              className={
                styles.loading
              }

              aria-label="Loading suggestions"
            />
          ) : null}


          {/*
           * Hidden system image input.
           *
           * No custom dialog.
           */}
          <input
            ref={
              visualInputRef
            }

            type="file"

            accept="image/*"

            className={
              styles.visualSearchInput
            }

            aria-hidden="true"

            tabIndex={-1}

            onChange={
              handleVisualImage
            }
          />


          <button
            type="button"

            className={
              styles.imageSearchButton
            }

            aria-label="Search with a photo"

            title="Search with a photo"

            onClick={
              openVisualPicker
            }
          >
            <NextImage
              src="/icons/search/visual_search.svg"

              alt=""

              width={24}

              height={24}

              unoptimized

              aria-hidden="true"

              className={
                styles.visualSearchIcon
              }
            />
          </button>
        </div>
      </form>


      {/*
       * ======================================================
       * AUTOCOMPLETE / VISUAL RESULTS
       * ======================================================
       */}

      {showDropdown ? (
        <div
          id="store-search-suggestions"

          className={
            styles.dropdown
          }

          role="listbox"
        >
          {/*
           * VISUAL SEARCH STATE
           */}

          {hasVisualState ? (
            <section
              className={`${styles.section} ${styles.visualSection}`}
            >
              <div
                className={
                  styles.visualHeader
                }
              >
                <div>
                  <span
                    className={
                      styles.sectionTitle
                    }
                  >
                    Visual search
                  </span>

                  <strong>
                    {visualSearching
                      ? "Finding similar products…"
                      : visualResult
                        ? `${visualResult.meta.total.toLocaleString()} match${
                            visualResult.meta.total ===
                            1
                              ? ""
                              : "es"
                          }`
                        : "Image search"}
                  </strong>
                </div>


                {!visualSearching ? (
                  <button
                    type="button"

                    className={
                      styles.chooseAnother
                    }

                    onClick={
                      openVisualPicker
                    }
                  >
                    Choose another
                  </button>
                ) : null}
              </div>


              {visualSearching ? (
                <div
                  className={
                    styles.visualLoading
                  }

                  role="status"
                >
                  <span
                    className={
                      styles.visualSpinner
                    }

                    aria-hidden="true"
                  />

                  <span>
                    Comparing your
                    image with the
                    catalog…
                  </span>
                </div>
              ) : null}


              {!visualSearching &&
              visualError ? (
                <div
                  className={
                    styles.visualError
                  }

                  role="alert"
                >
                  {
                    visualError
                  }
                </div>
              ) : null}


              {!visualSearching &&
              visualResult &&
              visualProducts.length ===
                0 ? (
                <div
                  className={
                    styles.visualEmpty
                  }
                >
                  <strong>
                    No close visual
                    match yet.
                  </strong>

                  <span>
                    Try a clearer
                    product photo.
                  </span>
                </div>
              ) : null}


              {!visualSearching &&
              visualProducts.length >
                0 ? (
                <div
                  className={
                    styles.visualGrid
                  }
                >
                  {visualProducts.map(
                    (
                      product,
                    ) => (
                      <button
                        key={
                          product.id
                        }

                        type="button"

                        className={
                          styles.visualProduct
                        }

                        onClick={() =>
                          navigate(
                            `/product/${product.slug}`,
                          )
                        }
                      >
                        <span
                          className={
                            styles.visualProductMedia
                          }
                        >
                          {product.primary_image_url ? (
                            <CatalogImage
                              src={
                                product.primary_image_url
                              }

                              alt=""

                              fill

                              sizes="(max-width: 48rem) 30vw, 8rem"

                              className={
                                styles.visualProductImage
                              }
                            />
                          ) : (
                            <span
                              className={
                                styles.imageFallback
                              }

                              aria-hidden="true"
                            >
                              E
                            </span>
                          )}


                          {product.match_type ? (
                            <small
                              className={
                                styles.visualMatch
                              }
                            >
                              {product.match_type ===
                              "primary"
                                ? "Match"
                                : "Similar"}
                            </small>
                          ) : null}
                        </span>


                        <span
                          className={
                            styles.visualProductBody
                          }
                        >
                          <strong>
                            {
                              product.name
                            }
                          </strong>

                          <small>
                            {[
                              product.brand,

                              product
                                .category
                                ?.name,
                            ]
                              .filter(
                                Boolean,
                              )
                              .join(
                                " · ",
                              )}
                          </small>

                          <b>
                            {formatMoney(
                              product.price_amount,
                              product.currency,
                            )}
                          </b>
                        </span>
                      </button>
                    ),
                  )}
                </div>
              ) : null}
            </section>
          ) : null}


          {/*
           * NORMAL TEXT SEARCH
           */}

          {!hasVisualState &&
          hasSuggestions ? (
            <>
              {suggestions
                .products
                .length >
              0 ? (
                <section
                  className={
                    styles.section
                  }
                >
                  <div
                    className={
                      styles.sectionTitle
                    }
                  >
                    Products
                  </div>


                  <div
                    className={
                      styles.productList
                    }
                  >
                    {suggestions.products.map(
                      (
                        product,
                        index,
                      ) => (
                        <button
                          id={`search-option-${index}`}

                          key={
                            product.id
                          }

                          type="button"

                          role="option"

                          aria-selected={
                            activeIndex ===
                            index
                          }

                          className={[
                            styles.product,

                            activeIndex ===
                            index
                              ? styles.active
                              : "",
                          ]
                            .filter(
                              Boolean,
                            )
                            .join(
                              " ",
                            )}

                          onMouseEnter={() =>
                            setActiveIndex(
                              index,
                            )
                          }

                          onClick={() =>
                            navigate(
                              `/product/${product.slug}`,
                            )
                          }
                        >
                          <span
                            className={
                              styles.productImage
                            }
                          >
                            {product.primary_image_url ? (
                              <CatalogImage
                                src={
                                  product.primary_image_url
                                }

                                alt=""

                                width={
                                  52
                                }

                                height={
                                  58
                                }

                                sizes="52px"
                              />
                            ) : (
                              <span
                                className={
                                  styles.imageFallback
                                }

                                aria-hidden="true"
                              >
                                E
                              </span>
                            )}
                          </span>


                          <span
                            className={
                              styles.productText
                            }
                          >
                            <strong>
                              {
                                product.name
                              }
                            </strong>

                            <small>
                              {[
                                product.brand,

                                product
                                  .category
                                  ?.name,
                              ]
                                .filter(
                                  Boolean,
                                )
                                .join(
                                  " · ",
                                )}
                            </small>
                          </span>


                          <span
                            className={
                              styles.price
                            }
                          >
                            {formatMoney(
                              product.price_amount,
                              product.currency,
                            )}
                          </span>
                        </button>
                      ),
                    )}
                  </div>
                </section>
              ) : null}


              {suggestions
                .categories
                .length >
              0 ? (
                <section
                  className={
                    styles.section
                  }
                >
                  <div
                    className={
                      styles.sectionTitle
                    }
                  >
                    Categories
                  </div>


                  <div
                    className={
                      styles.compactList
                    }
                  >
                    {suggestions.categories.map(
                      (
                        category,
                        index,
                      ) => {
                        const optionIndex =
                          categoryOffset +
                          index;


                        return (
                          <button
                            id={`search-option-${optionIndex}`}

                            key={
                              category.slug
                            }

                            type="button"

                            role="option"

                            aria-selected={
                              activeIndex ===
                              optionIndex
                            }

                            className={[
                              styles.compactRow,

                              activeIndex ===
                              optionIndex
                                ? styles.active
                                : "",
                            ]
                              .filter(
                                Boolean,
                              )
                              .join(
                                " ",
                              )}

                            onMouseEnter={() =>
                              setActiveIndex(
                                optionIndex,
                              )
                            }

                            onClick={() =>
                              navigate(
                                `/category/${category.slug}`,
                              )
                            }
                          >
                            <span
                              className={
                                styles.compactIcon
                              }
                            >
                              <Icon
                                name="categories"
                                size={
                                  16
                                }
                              />
                            </span>


                            <span>
                              {
                                category.name
                              }
                            </span>


                            <span
                              aria-hidden="true"

                              className={
                                styles.arrow
                              }
                            >
                              →
                            </span>
                          </button>
                        );
                      },
                    )}
                  </div>
                </section>
              ) : null}


              {suggestions
                .brands
                .length >
              0 ? (
                <section
                  className={
                    styles.section
                  }
                >
                  <div
                    className={
                      styles.sectionTitle
                    }
                  >
                    Brands
                  </div>


                  <div
                    className={
                      styles.brandList
                    }
                  >
                    {suggestions.brands.map(
                      (
                        brand,
                        index,
                      ) => {
                        const optionIndex =
                          brandOffset +
                          index;


                        return (
                          <button
                            id={`search-option-${optionIndex}`}

                            key={
                              brand
                            }

                            type="button"

                            role="option"

                            aria-selected={
                              activeIndex ===
                              optionIndex
                            }

                            className={[
                              styles.brand,

                              activeIndex ===
                              optionIndex
                                ? styles.activeBrand
                                : "",
                            ]
                              .filter(
                                Boolean,
                              )
                              .join(
                                " ",
                              )}

                            onMouseEnter={() =>
                              setActiveIndex(
                                optionIndex,
                              )
                            }

                            onClick={() =>
                              navigate(
                                searchHref(
                                  brand,
                                ),
                              )
                            }
                          >
                            {
                              brand
                            }
                          </button>
                        );
                      },
                    )}
                  </div>
                </section>
              ) : null}
            </>
          ) : null}


          {!hasVisualState &&
          !hasSuggestions &&
          !loading ? (
            <div
              className={
                styles.empty
              }
            >
              No direct
              suggestions yet
            </div>
          ) : null}


          {!hasVisualState ? (
            <button
              id={`search-option-${searchAllIndex}`}

              type="button"

              role="option"

              aria-selected={
                activeIndex ===
                searchAllIndex
              }

              className={[
                styles.searchAll,

                activeIndex ===
                searchAllIndex
                  ? styles.searchAllActive
                  : "",
              ]
                .filter(
                  Boolean,
                )
                .join(
                  " ",
                )}

              onMouseEnter={() =>
                setActiveIndex(
                  searchAllIndex,
                )
              }

              onClick={() =>
                submitSearch()
              }
            >
              <span
                className={
                  styles.searchAllIcon
                }
              >
                <Icon
                  name="search"

                  size={17}
                />
              </span>


              <span>
                Search all results
                for{" "}

                <strong>
                  “
                  {query.trim()}
                  ”
                </strong>
              </span>


              <span
                aria-hidden="true"

                className={
                  styles.arrow
                }
              >
                →
              </span>
            </button>
          ) : null}
        </div>
      ) : null}
    </div>
  );
}