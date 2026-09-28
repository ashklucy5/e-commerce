// Location: src/components/account/components/ProductRequestCenterClient.tsx

"use client";

import Link from "next/link";

import {
  type ChangeEvent,
  type ClipboardEvent,
  type DragEvent,
  type FormEvent,
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";

import { Icon } from "@/components/ui/Icon";

import {
  useCustomerAuth,
} from "@/lib/account/use-customer-auth";

import type {
  ProductRequest,
  ProductRequestDataResponse,
  ProductRequestListResponse,
  ProductRequestMessage,
  ProductRequestMessagesResponse,
  SourcingConfirmation,
  SourcingConfirmationResponse,
  SourcingOffer,
  SourcingOffersResponse,
} from "@/lib/api/contracts/product-request";

import {
  formatMoney,
} from "@/lib/money/format";

import {
  buildCustomerSourcingTimeline,
  type CustomerSourcingTimelineItem,
} from "@/lib/product-requests/customer-sourcing-timeline";

import styles from "../css/ProductRequestCenter.module.css";

/* =========================================================
   TYPES
   ========================================================= */

type CreateForm = {
  requested_product_name: string;
  description: string;
  requested_quantity: string;
  external_url: string;
  requirements: string;
};

type RequestAttachment = {
  id: string;

  kind:
    | "image"
    | "link";

  name: string;
  url: string;

  mime_type?: string;
  size?: number;
};

type PreviewImage = {
  url: string;
  name: string;
};

type AttachmentTarget =
  | "create"
  | "message";

type UnknownRecord =
  Record<string, unknown>;

/* =========================================================
   CONSTANTS
   ========================================================= */

const MAX_ATTACHMENTS = 4;

const MAX_SOURCE_IMAGE_BYTES =
  8 * 1024 * 1024;

/*
 * Product Request attachments are intentionally
 * compressed much more aggressively than Support
 * attachments.
 *
 * The Product Request backend currently has a much
 * smaller JSON attachment allowance.
 */
const MAX_IMAGE_DATA_URL_LENGTH =
  28_000;

const MAX_ATTACHMENT_JSON_BYTES =
  118 * 1024;

const emptyCreateForm: CreateForm = {
  requested_product_name: "",
  description: "",
  requested_quantity: "1",
  external_url: "",
  requirements: "",
};

const statusLabels:
  Record<string, string> = {
    pending_review:
      "Under review",

    on_hold:
      "On hold",

    accepted:
      "Accepted for sourcing",

    negotiating:
      "Negotiating",

    agreed:
      "Agreement confirmed",

    converted_to_order:
      "Order created",

    cancelled:
      "Cancelled",

    sent:
      "Awaiting response",

    customer_accepted:
      "Accepted",

    customer_rejected:
      "Declined",

    finalized:
      "Finalized",

    confirmed:
      "Confirmed",

    order_created:
      "Order created",
  };

/* =========================================================
   BASIC HELPERS
   ========================================================= */

function makeAttachmentID() {
  if (
    typeof crypto !==
      "undefined" &&
    typeof crypto.randomUUID ===
      "function"
  ) {
    return crypto.randomUUID();
  }

  return `${Date.now()}-${Math.random()
    .toString(36)
    .slice(2)}`;
}

function statusLabel(
  value: string,
) {
  return (
    statusLabels[value] ??
    value.replaceAll(
      "_",
      " ",
    )
  );
}

function statusClass(
  value: string,
) {
  switch (value) {
    case "pending_review":
      return styles.statusPending;

    case "on_hold":
      return styles.statusHold;

    case "accepted":
      return styles.statusAccepted;

    case "negotiating":
    case "sent":
      return styles.statusNegotiating;

    case "agreed":
    case "customer_accepted":
    case "finalized":
    case "confirmed":
      return styles.statusAgreed;

    case "converted_to_order":
    case "order_created":
      return styles.statusConverted;

    case "cancelled":
    case "customer_rejected":
      return styles.statusCancelled;

    default:
      return styles.statusNeutral;
  }
}

function formatDate(
  value?: string,
) {
  if (!value) {
    return "—";
  }

  const date =
    new Date(value);

  if (
    Number.isNaN(
      date.getTime(),
    )
  ) {
    return "—";
  }

  return new Intl.DateTimeFormat(
    "en",
    {
      dateStyle:
        "medium",

      timeStyle:
        "short",
    },
  ).format(date);
}

function formatShortDate(
  value?: string,
) {
  if (!value) {
    return "—";
  }

  const date =
    new Date(value);

  if (
    Number.isNaN(
      date.getTime(),
    )
  ) {
    return "—";
  }

  return new Intl.DateTimeFormat(
    "en",
    {
      month: "short",
      day: "numeric",
    },
  ).format(date);
}

function formatSpecificationValue(
  value: unknown,
) {
  if (
    value === null ||
    value === undefined
  ) {
    return "—";
  }

  if (
    typeof value === "string" ||
    typeof value === "number" ||
    typeof value === "boolean"
  ) {
    return String(value);
  }

  try {
    return JSON.stringify(value);
  } catch {
    return String(value);
  }
}

function asRecord(
  value: unknown,
): UnknownRecord | null {
  if (
    !value ||
    typeof value !==
      "object" ||
    Array.isArray(value)
  ) {
    return null;
  }

  return value as UnknownRecord;
}

function readRequirements(
  request:
    ProductRequest | null,
) {
  if (!request) {
    return "";
  }

  /*
   * Read through an unknown record here because the API
   * may return customer_requirements either as a structured
   * JSON object or, for older data, as a string.
   */
  const record =
    request as unknown as UnknownRecord;

  const raw: unknown =
    record[
      "customer_requirements"
    ];

  if (
    typeof raw ===
    "string"
  ) {
    return raw.trim();
  }

  const object =
    asRecord(raw);

  if (!object) {
    return "";
  }

  const notes: unknown =
    object["notes"];

  if (
    typeof notes ===
    "string"
  ) {
    return notes.trim();
  }

  return "";
}

async function readError(
  response: Response,
  fallback: string,
) {
  try {
    const payload =
      (await response.json()) as {
        error?: {
          message?: string;
        };
      };

    return (
      payload.error
        ?.message
        ?.trim() ||
      fallback
    );
  } catch {
    return fallback;
  }
}

async function readOptionalJSON<T>(
  url: string,
): Promise<T | null> {
  const response =
    await fetch(
      url,
      {
        cache:
          "no-store",
      },
    );

  if (
    response.status ===
    404
  ) {
    return null;
  }

  if (
    !response.ok
  ) {
    throw new Error(
      await readError(
        response,
        "Unable to load request data.",
      ),
    );
  }

  return (
    await response.json()
  ) as T;
}

function normaliseURL(
  input: string,
) {
  const trimmed =
    input.trim();

  if (!trimmed) {
    return null;
  }

  try {
    const parsed =
      new URL(trimmed);

    if (
      parsed.protocol !==
        "http:" &&
      parsed.protocol !==
        "https:"
    ) {
      return null;
    }

    return parsed.toString();
  } catch {
    return null;
  }
}

/* =========================================================
   ATTACHMENT HELPERS
   ========================================================= */

function parseAttachments(
  input: unknown,
): RequestAttachment[] {
  if (!input) {
    return [];
  }

  let value = input;

  if (
    typeof value ===
    "string"
  ) {
    const trimmed =
      value.trim();

    if (!trimmed) {
      return [];
    }

    if (
      trimmed.startsWith("[")
    ) {
      try {
        value =
          JSON.parse(
            trimmed,
          );
      } catch {
        return [];
      }
    } else if (
      trimmed.startsWith(
        "data:image/",
      )
    ) {
      return [
        {
          id:
            makeAttachmentID(),

          kind:
            "image",

          name:
            "Reference image",

          url:
            trimmed,
        },
      ];
    } else {
      const url =
        normaliseURL(
          trimmed,
        );

      if (!url) {
        return [];
      }

      return [
        {
          id:
            makeAttachmentID(),

          kind:
            "link",

          name:
            url,

          url,
        },
      ];
    }
  }

  if (
    !Array.isArray(
      value,
    )
  ) {
    return [];
  }

  const result:
    RequestAttachment[] =
      [];

  for (
    const raw of value
  ) {
    if (
      typeof raw ===
      "string"
    ) {
      if (
        raw.startsWith(
          "data:image/",
        )
      ) {
        result.push({
          id:
            makeAttachmentID(),

          kind:
            "image",

          name:
            "Reference image",

          url:
            raw,
        });

        continue;
      }

      const url =
        normaliseURL(raw);

      if (url) {
        result.push({
          id:
            makeAttachmentID(),

          kind:
            "link",

          name:
            url,

          url,
        });
      }

      continue;
    }

    const object =
      asRecord(raw);

    if (!object) {
      continue;
    }

    const rawURL =
      object.url ??
      object.href ??
      object.src ??
      object.public_url ??
      object.download_url;

    if (
      typeof rawURL !==
      "string"
    ) {
      continue;
    }

    const rawKind =
      object.kind ??
      object.type;

    const mime =
      typeof object.mime_type ===
      "string"
        ? object.mime_type
        : typeof object.mime ===
            "string"
          ? object.mime
          : undefined;

    const isImage =
      rawURL.startsWith(
        "data:image/",
      ) ||
      mime?.startsWith(
        "image/",
      ) ||
      rawKind ===
        "image";

    const validURL =
      isImage
        ? rawURL
        : normaliseURL(
            rawURL,
          );

    if (!validURL) {
      continue;
    }

    const nameValue =
      object.name ??
      object.filename ??
      object.title;

    result.push({
      id:
        typeof object.id ===
        "string"
          ? object.id
          : makeAttachmentID(),

      kind:
        isImage
          ? "image"
          : "link",

      name:
        typeof nameValue ===
        "string"
          ? nameValue
          : isImage
            ? "Reference image"
            : validURL,

      url:
        validURL,

      mime_type:
        mime,

      size:
        typeof object.size ===
        "number"
          ? object.size
          : undefined,
    });
  }

  return result;
}

function entityAttachments(
  value: unknown,
) {
  const object =
    asRecord(value);

  return parseAttachments(
    object?.attachments,
  );
}

function serialiseAttachments(
  attachments:
    RequestAttachment[],
) {
  return attachments.map(
    (attachment) => ({
      id:
        attachment.id,

      type:
        attachment.kind,

      kind:
        attachment.kind,

      name:
        attachment.name,

      url:
        attachment.url,

      mime_type:
        attachment.mime_type,

      size:
        attachment.size,
    }),
  );
}

function attachmentPayloadBytes(
  attachments:
    RequestAttachment[],
) {
  const encoded =
    JSON.stringify(
      serialiseAttachments(
        attachments,
      ),
    );

  return new TextEncoder()
    .encode(encoded)
    .byteLength;
}

function attachmentFallback(
  attachments:
    RequestAttachment[],
) {
  const imageCount =
    attachments.filter(
      (attachment) =>
        attachment.kind ===
        "image",
    ).length;

  const linkCount =
    attachments.filter(
      (attachment) =>
        attachment.kind ===
        "link",
    ).length;

  if (
    imageCount > 0 &&
    linkCount > 0
  ) {
    return "Shared references.";
  }

  if (
    imageCount === 1
  ) {
    return "Shared an image.";
  }

  if (
    imageCount > 1
  ) {
    return `Shared ${imageCount} images.`;
  }

  return "Shared a link.";
}

/* =========================================================
   IMAGE COMPRESSION
   ========================================================= */

function readFileAsDataURL(
  file: File,
) {
  return new Promise<string>(
    (
      resolve,
      reject,
    ) => {
      const reader =
        new FileReader();

      reader.onload =
        () => {
          if (
            typeof reader.result ===
            "string"
          ) {
            resolve(
              reader.result,
            );

            return;
          }

          reject(
            new Error(
              "Unable to read image.",
            ),
          );
        };

      reader.onerror =
        () => {
          reject(
            new Error(
              "Unable to read image.",
            ),
          );
        };

      reader.readAsDataURL(
        file,
      );
    },
  );
}

function loadBrowserImage(
  src: string,
) {
  return new Promise<HTMLImageElement>(
    (
      resolve,
      reject,
    ) => {
      const image =
        new Image();

      image.onload =
        () =>
          resolve(
            image,
          );

      image.onerror =
        () =>
          reject(
            new Error(
              "Unable to process image.",
            ),
          );

      image.src =
        src;
    },
  );
}

async function imageToAttachment(
  file: File,
): Promise<RequestAttachment> {
  if (
    !file.type.startsWith(
      "image/",
    )
  ) {
    throw new Error(
      "Only image attachments are supported.",
    );
  }

  if (
    file.size >
    MAX_SOURCE_IMAGE_BYTES
  ) {
    throw new Error(
      "Images must be 8 MB or smaller.",
    );
  }

  const source =
    await readFileAsDataURL(
      file,
    );

  const image =
    await loadBrowserImage(
      source,
    );

  let maxDimension =
    900;

  const qualities = [
    0.72,
    0.58,
    0.46,
    0.36,
    0.28,
  ];

  let output = "";

  for (
    let attempt = 0;
    attempt < 6;
    attempt += 1
  ) {
    const scale =
      Math.min(
        1,
        maxDimension /
          Math.max(
            image.naturalWidth,
            image.naturalHeight,
          ),
      );

    const width =
      Math.max(
        1,
        Math.round(
          image.naturalWidth *
            scale,
        ),
      );

    const height =
      Math.max(
        1,
        Math.round(
          image.naturalHeight *
            scale,
        ),
      );

    const canvas =
      document.createElement(
        "canvas",
      );

    canvas.width =
      width;

    canvas.height =
      height;

    const context =
      canvas.getContext(
        "2d",
      );

    if (!context) {
      throw new Error(
        "Unable to prepare image.",
      );
    }

    context.fillStyle =
      "#ffffff";

    context.fillRect(
      0,
      0,
      width,
      height,
    );

    context.drawImage(
      image,
      0,
      0,
      width,
      height,
    );

    for (
      const quality of qualities
    ) {
      const candidate =
        canvas.toDataURL(
          "image/jpeg",
          quality,
        );

      if (
        candidate.length <=
        MAX_IMAGE_DATA_URL_LENGTH
      ) {
        output =
          candidate;

        break;
      }
    }

    if (output) {
      break;
    }

    maxDimension =
      Math.max(
        360,
        Math.round(
          maxDimension *
            0.76,
        ),
      );
  }

  if (!output) {
    throw new Error(
      "This image could not be compressed enough. Try cropping it first.",
    );
  }

  return {
    id:
      makeAttachmentID(),

    kind:
      "image",

    name:
      file.name ||
      `reference-${Date.now()}.jpg`,

    url:
      output,

    mime_type:
      "image/jpeg",

    size:
      Math.round(
        output.length *
          0.75,
      ),
  };
}

/* =========================================================
   MESSAGE UI HELPERS
   ========================================================= */

function customerInitials(
  name?: string,
) {
  const parts =
    (name ?? "")
      .trim()
      .split(/\s+/)
      .filter(Boolean);

  if (
    parts.length ===
    0
  ) {
    return "U";
  }

  if (
    parts.length ===
    1
  ) {
    return parts[0]
      .slice(0, 2)
      .toUpperCase();
  }

  return (
    parts[0][0] +
    parts[
      parts.length - 1
    ][0]
  ).toUpperCase();
}

function MessageText({
  text,
}: {
  text: string;
}) {
  const parts =
    text.split(
      /(https?:\/\/[^\s]+)/gi,
    );

  return (
    <>
      {parts.map(
        (
          part,
          index,
        ) => {
          const url =
            normaliseURL(
              part,
            );

          if (!url) {
            return (
              <span
                key={
                  index
                }
              >
                {part}
              </span>
            );
          }

          return (
            <a
              key={
                index
              }
              href={url}
              target="_blank"
              rel="noopener noreferrer"
            >
              {part}
            </a>
          );
        },
      )}
    </>
  );
}

function PaperclipIcon() {
  return (
    <svg
      viewBox="0 0 24 24"
      width="18"
      height="18"
      aria-hidden="true"
    >
      <path
        d="M8.5 12.5 14.8 6.2a3 3 0 0 1 4.2 4.2l-8.1 8.1a5 5 0 0 1-7.1-7.1l8-8"
        fill="none"
        stroke="currentColor"
        strokeWidth="1.8"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
    </svg>
  );
}

function ImageIcon() {
  return (
    <svg
      viewBox="0 0 24 24"
      width="18"
      height="18"
      aria-hidden="true"
    >
      <rect
        x="3.5"
        y="4.5"
        width="17"
        height="15"
        rx="3"
        fill="none"
        stroke="currentColor"
        strokeWidth="1.6"
      />

      <circle
        cx="9"
        cy="10"
        r="1.6"
        fill="currentColor"
      />

      <path
        d="m6 17 4.1-4 2.7 2.5 2.2-2.1 3 3.6"
        fill="none"
        stroke="currentColor"
        strokeWidth="1.6"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
    </svg>
  );
}

function MessageAttachments({
  value,
  onPreview,
}: {
  value: unknown;

  onPreview: (
    attachment:
      RequestAttachment,
  ) => void;
}) {
  const attachments =
    parseAttachments(
      value,
    );

  if (
    attachments.length ===
    0
  ) {
    return null;
  }

  return (
    <div
      className={
        styles.messageAttachments
      }
    >
      {attachments.map(
        (attachment) => {
          if (
            attachment.kind ===
            "image"
          ) {
            return (
              <button
                key={
                  attachment.id
                }
                type="button"
                className={
                  styles.messageImageButton
                }
                onClick={() =>
                  onPreview(
                    attachment,
                  )
                }
              >
                <img
                  src={
                    attachment.url
                  }
                  alt={
                    attachment.name
                  }
                  className={
                    styles.messageImage
                  }
                  loading="lazy"
                  decoding="async"
                  fetchPriority="low"
                  draggable={false}
                />
              </button>
            );
          }

          return (
            <a
              key={
                attachment.id
              }
              href={
                attachment.url
              }
              target="_blank"
              rel="noopener noreferrer"
              className={
                styles.messageLink
              }
            >
              <span
                className={
                  styles.messageLinkIcon
                }
                aria-hidden="true"
              >
                ↗
              </span>

              <span>
                {
                  attachment.name
                }
              </span>
            </a>
          );
        },
      )}
    </div>
  );
}

/* =========================================================
   COMPONENT
   ========================================================= */

export function ProductRequestCenterClient() {
  const {
    customer,
  } =
    useCustomerAuth();

  const [
    requests,
    setRequests,
  ] =
    useState<
      ProductRequest[]
    >([]);

  const [
    selectedID,
    setSelectedID,
  ] =
    useState("");

  const [
    selected,
    setSelected,
  ] =
    useState<
      ProductRequest | null
    >(null);

  const [
    messages,
    setMessages,
  ] =
    useState<
      ProductRequestMessage[]
    >([]);

  const [
    offers,
    setOffers,
  ] =
    useState<
      SourcingOffer[]
    >([]);

  const [
    confirmation,
    setConfirmation,
  ] =
    useState<
      SourcingConfirmation | null
    >(null);

  const [
    createForm,
    setCreateForm,
  ] =
    useState<CreateForm>(
      emptyCreateForm,
    );

  const [
    createAttachments,
    setCreateAttachments,
  ] =
    useState<
      RequestAttachment[]
    >([]);

  const [
    message,
    setMessage,
  ] =
    useState("");

  const [
    messageAttachments,
    setMessageAttachments,
  ] =
    useState<
      RequestAttachment[]
    >([]);

  const [
    showCreate,
    setShowCreate,
  ] =
    useState(false);

  const [
    mobileDetailOpen,
    setMobileDetailOpen,
  ] =
    useState(false);

  const [
    mobileInfoOpen,
    setMobileInfoOpen,
  ] =
    useState(false);

  const [
    attachmentMenuOpen,
    setAttachmentMenuOpen,
  ] =
    useState(false);

  const [
    linkEditorOpen,
    setLinkEditorOpen,
  ] =
    useState(false);

  const [
    linkInput,
    setLinkInput,
  ] =
    useState("");

  const [
    previewImage,
    setPreviewImage,
  ] =
    useState<
      PreviewImage | null
    >(null);

  const [
    loading,
    setLoading,
  ] =
    useState(true);

  const [
    detailLoading,
    setDetailLoading,
  ] =
    useState(false);

  const [
    attachmentBusy,
    setAttachmentBusy,
  ] =
    useState(false);

  const [
    busy,
    setBusy,
  ] =
    useState("");

  const [
    error,
    setError,
  ] =
    useState("");

  const [
    signedOut,
    setSignedOut,
  ] =
    useState(false);

  const shellRef =
    useRef<HTMLDivElement | null>(
      null,
    );

  const createFileInputRef =
    useRef<HTMLInputElement | null>(
      null,
    );

  const messageFileInputRef =
    useRef<HTMLInputElement | null>(
      null,
    );

  const attachmentMenuRef =
    useRef<HTMLDivElement | null>(
      null,
    );

  const messagesEndRef =
    useRef<HTMLDivElement | null>(
      null,
    );

  const threadRef =
    useRef<HTMLDivElement | null>(
      null,
    );

  const stickToBottomRef =
    useRef(true);

  const forceBottomRef =
    useRef(false);

  /* =======================================================
     DERIVED VALUES
     ======================================================= */

  const customerAvatarText =
    customerInitials(
      customer?.full_name,
    );

  const selectedReferences =
    useMemo(
      () =>
        entityAttachments(
          selected,
        ),
      [
        selected,
      ],
    );

  const selectedRequirements =
    useMemo(
      () =>
        readRequirements(
          selected,
        ),
      [
        selected,
      ],
    );

  const timelineItems =
    useMemo(
      () =>
        selected
          ? buildCustomerSourcingTimeline(
              selected,
              messages,
              offers,
              confirmation,
            )
          : [],
      [
        selected,
        messages,
        offers,
        confirmation,
      ],
    );

  const lastTimelineID =
    timelineItems[
      timelineItems.length - 1
    ]?.id ?? "";

  /* =======================================================
     DESKTOP VIEWPORT HEIGHT
     ======================================================= */

  useEffect(() => {
    const viewport =
      window.visualViewport;

    let frame = 0;

    function syncHeight() {
      window.cancelAnimationFrame(
        frame,
      );

      frame =
        window.requestAnimationFrame(
          () => {
            const shell =
              shellRef.current;

            if (!shell) {
              return;
            }

            const viewportHeight =
              viewport?.height ??
              window.innerHeight;

            const viewportTop =
              viewport?.offsetTop ??
              0;

            const rect =
              shell.getBoundingClientRect();

            const shellTop =
              Math.max(
                rect.top -
                  viewportTop,
                0,
              );

            const availableHeight =
              Math.max(
                420,
                viewportHeight -
                  shellTop -
                  6,
              );

            shell.style.setProperty(
              "--request-shell-height",
              `${Math.round(
                availableHeight,
              )}px`,
            );

            shell.style.setProperty(
              "--request-viewport-height",
              `${Math.round(
                viewportHeight,
              )}px`,
            );

            shell.style.setProperty(
              "--request-viewport-top",
              `${Math.round(
                viewportTop,
              )}px`,
            );
          },
        );
    }

    syncHeight();

    window.addEventListener(
      "resize",
      syncHeight,
    );

    viewport?.addEventListener(
      "resize",
      syncHeight,
    );

    viewport?.addEventListener(
      "scroll",
      syncHeight,
    );

    return () => {
      window.cancelAnimationFrame(
        frame,
      );

      window.removeEventListener(
        "resize",
        syncHeight,
      );

      viewport?.removeEventListener(
        "resize",
        syncHeight,
      );

      viewport?.removeEventListener(
        "scroll",
        syncHeight,
      );
    };
  }, []);

  /* =======================================================
     LOAD REQUEST LIST
     ======================================================= */

  const loadRequests =
    useCallback(
      async (
        preferredID = "",
        silent = false,
      ) => {
        if (!silent) {
          setLoading(
            true,
          );

          setError("");
        }

        try {
          const response =
            await fetch(
              "/api/storefront/account/requests?limit=100&offset=0",
              {
                cache:
                  "no-store",
              },
            );

          if (
            response.status ===
            401
          ) {
            setSignedOut(
              true,
            );

            return;
          }

          if (
            !response.ok
          ) {
            throw new Error(
              await readError(
                response,
                "Unable to load product requests.",
              ),
            );
          }

          setSignedOut(
            false,
          );

          const payload =
            (await response.json()) as
              ProductRequestListResponse;

          const items =
            Array.isArray(
              payload.data,
            )
              ? [
                  ...payload.data,
                ]
              : [];

          items.sort(
            (
              a,
              b,
            ) =>
              new Date(
                b.updated_at,
              ).getTime() -
              new Date(
                a.updated_at,
              ).getTime(),
          );

          setRequests(
            items,
          );

          setSelectedID(
            (
              current,
            ) => {
              const desired =
                preferredID ||
                current;

              if (
                desired &&
                items.some(
                  (
                    item,
                  ) =>
                    item.id ===
                    desired,
                )
              ) {
                return desired;
              }

              return (
                items[0]
                  ?.id ??
                ""
              );
            },
          );
        } catch (
          caught
        ) {
          if (!silent) {
            setError(
              caught instanceof
                Error
                ? caught.message
                : "Unable to load product requests.",
            );
          }
        } finally {
          if (!silent) {
            setLoading(
              false,
            );
          }
        }
      },
      [],
    );

  /* =======================================================
     LOAD REQUEST DETAIL
     ======================================================= */

  const loadDetail =
    useCallback(
      async (
        requestID:
          string,
        silent = false,
      ) => {
        if (
          !requestID
        ) {
          setSelected(
            null,
          );

          setMessages(
            [],
          );

          setOffers(
            [],
          );

          setConfirmation(
            null,
          );

          return;
        }

        if (!silent) {
          setDetailLoading(
            true,
          );

          setError("");
        }

        try {
          const encoded =
            encodeURIComponent(
              requestID,
            );

          const [
            detailPayload,
            messagePayload,
            offerPayload,
            confirmationPayload,
          ] =
            await Promise.all([
              readOptionalJSON<
                ProductRequestDataResponse<ProductRequest>
              >(
                `/api/storefront/account/requests/${encoded}`,
              ),

              readOptionalJSON<
                ProductRequestMessagesResponse
              >(
                `/api/storefront/account/requests/${encoded}/messages?limit=200&offset=0`,
              ),

              readOptionalJSON<
                SourcingOffersResponse
              >(
                `/api/storefront/account/requests/${encoded}/offers`,
              ),

              readOptionalJSON<
                SourcingConfirmationResponse
              >(
                `/api/storefront/account/requests/${encoded}/confirmation`,
              ),
            ]);

          setSelected(
            detailPayload
              ?.data ??
              null,
          );

          setMessages(
            messagePayload
              ?.data ??
              [],
          );

          setOffers(
            offerPayload
              ?.data ??
              [],
          );

          setConfirmation(
            confirmationPayload
              ?.data ??
              null,
          );
        } catch (
          caught
        ) {
          if (!silent) {
            setError(
              caught instanceof
                Error
                ? caught.message
                : "Unable to load this product request.",
            );
          }
        } finally {
          if (!silent) {
            setDetailLoading(
              false,
            );
          }
        }
      },
      [],
    );

  /* =======================================================
     INITIAL LOAD
     ======================================================= */

  useEffect(
    () => {
      void loadRequests();
    },
    [
      loadRequests,
    ],
  );

  useEffect(
    () => {
      void loadDetail(
        selectedID,
      );
    },
    [
      loadDetail,
      selectedID,
    ],
  );

  /* =======================================================
     POLLING
     ======================================================= */

  useEffect(() => {
    if (!selectedID) {
      return;
    }

    const interval =
      window.setInterval(
        () => {
          if (
            document.visibilityState !==
            "visible"
          ) {
            return;
          }

          void Promise.all([
            loadDetail(
              selectedID,
              true,
            ),

            loadRequests(
              selectedID,
              true,
            ),
          ]);
        },
        20000,
      );

    return () => {
      window.clearInterval(
        interval,
      );
    };
  }, [
    loadDetail,
    loadRequests,
    selectedID,
  ]);

  /* =======================================================
     KEEP CHAT AT THE BOTTOM WITHOUT HIJACKING MANUAL SCROLL
     ======================================================= */

  useEffect(() => {
    if (!lastTimelineID) {
      return;
    }

    if (
      !stickToBottomRef.current &&
      !forceBottomRef.current
    ) {
      return;
    }

    forceBottomRef.current = false;

    const frame =
      window.requestAnimationFrame(
        () => {
          messagesEndRef
            .current
            ?.scrollIntoView({
              block: "end",
              behavior: "auto",
            });
        },
      );

    return () =>
      window.cancelAnimationFrame(
        frame,
      );
  }, [
    lastTimelineID,
  ]);

  /* =======================================================
     ATTACHMENT MENU OUTSIDE CLICK
     ======================================================= */

  useEffect(() => {
    function handlePointerDown(
      event: PointerEvent,
    ) {
      if (
        !attachmentMenuOpen &&
        !linkEditorOpen
      ) {
        return;
      }

      const element =
        attachmentMenuRef.current;

      if (
        element &&
        !element.contains(
          event.target as Node,
        )
      ) {
        setAttachmentMenuOpen(
          false,
        );

        setLinkEditorOpen(
          false,
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
  }, [
    attachmentMenuOpen,
    linkEditorOpen,
  ]);

  /* =======================================================
     MODAL / LIGHTBOX LOCK
     ======================================================= */

  useEffect(() => {
    if (
      !showCreate &&
      !previewImage
    ) {
      return;
    }

    const previousOverflow =
      document.body.style
        .overflow;

    document.body.style.overflow =
      "hidden";

    function handleKeyDown(
      event: KeyboardEvent,
    ) {
      if (
        event.key !==
        "Escape"
      ) {
        return;
      }

      if (previewImage) {
        setPreviewImage(
          null,
        );

        return;
      }

      if (
        busy !== "create" &&
        !attachmentBusy
      ) {
        setShowCreate(
          false,
        );
      }
    }

    document.addEventListener(
      "keydown",
      handleKeyDown,
    );

    return () => {
      document.body.style.overflow =
        previousOverflow;

      document.removeEventListener(
        "keydown",
        handleKeyDown,
      );
    };
  }, [
    attachmentBusy,
    busy,
    previewImage,
    showCreate,
  ]);

  /* =======================================================
     FORM HELPERS
     ======================================================= */

  function updateCreateField<
    K extends keyof CreateForm,
  >(
    key: K,
    value:
      CreateForm[K],
  ) {
    setCreateForm(
      (
        current,
      ) => ({
        ...current,

        [key]:
          value,
      }),
    );

    setError("");
  }

  function openNewRequest() {
    setCreateForm(
      emptyCreateForm,
    );

    setCreateAttachments(
      [],
    );

    setError("");

    setShowCreate(
      true,
    );
  }

  function closeNewRequest() {
    if (
      busy ===
        "create" ||
      attachmentBusy
    ) {
      return;
    }

    setShowCreate(
      false,
    );

    setCreateForm(
      emptyCreateForm,
    );

    setCreateAttachments(
      [],
    );

    setError("");
  }

  function openRequest(
    requestID: string,
  ) {
    setSelectedID(
      requestID,
    );

    setMobileDetailOpen(
      true,
    );

    setMobileInfoOpen(
      false,
    );

    stickToBottomRef.current = true;
    forceBottomRef.current = true;

    setMessage("");

    setMessageAttachments(
      [],
    );

    setAttachmentMenuOpen(
      false,
    );

    setLinkEditorOpen(
      false,
    );

    setError("");
  }

  /* =======================================================
     ATTACH IMAGE
     ======================================================= */

  async function addImageFiles(
    files: File[],
    target:
      AttachmentTarget,
  ) {
    if (
      files.length ===
      0
    ) {
      return;
    }

    const current =
      target ===
      "create"
        ? createAttachments
        : messageAttachments;

    const remaining =
      MAX_ATTACHMENTS -
      current.length;

    if (
      remaining <= 0
    ) {
      setError(
        `You can attach up to ${MAX_ATTACHMENTS} references at a time.`,
      );

      return;
    }

    setAttachmentBusy(
      true,
    );

    setError("");

    try {
      const next = [
        ...current,
      ];

      for (
        const file of files.slice(
          0,
          remaining,
        )
      ) {
        const attachment =
          await imageToAttachment(
            file,
          );

        const candidate = [
          ...next,
          attachment,
        ];

        if (
          attachmentPayloadBytes(
            candidate,
          ) >
          MAX_ATTACHMENT_JSON_BYTES
        ) {
          throw new Error(
            "These reference images exceed the Product Request attachment limit. Remove an image or use a smaller crop.",
          );
        }

        next.push(
          attachment,
        );
      }

      if (
        target ===
        "create"
      ) {
        setCreateAttachments(
          next,
        );
      } else {
        setMessageAttachments(
          next,
        );
      }
    } catch (
      caught
    ) {
      setError(
        caught instanceof
          Error
          ? caught.message
          : "Unable to attach image.",
      );
    } finally {
      setAttachmentBusy(
        false,
      );
    }
  }

  function handleFileChange(
    event:
      ChangeEvent<HTMLInputElement>,
    target:
      AttachmentTarget,
  ) {
    const files =
      Array.from(
        event.target
          .files ??
          [],
      );

    event.target.value =
      "";

    setAttachmentMenuOpen(
      false,
    );

    void addImageFiles(
      files,
      target,
    );
  }

  function handlePaste(
    event:
      ClipboardEvent<HTMLTextAreaElement>,
    target:
      AttachmentTarget,
  ) {
    const files =
      Array.from(
        event.clipboardData
          .items,
      )
        .filter(
          (item) =>
            item.kind ===
              "file" &&
            item.type.startsWith(
              "image/",
            ),
        )
        .map(
          (item) =>
            item.getAsFile(),
        )
        .filter(
          (
            file,
          ): file is File =>
            Boolean(
              file,
            ),
        );

    if (
      files.length ===
      0
    ) {
      return;
    }

    event.preventDefault();

    void addImageFiles(
      files,
      target,
    );
  }

  function handleDrop(
    event:
      DragEvent<HTMLElement>,
    target:
      AttachmentTarget,
  ) {
    const files =
      Array.from(
        event.dataTransfer
          .files,
      ).filter(
        (file) =>
          file.type.startsWith(
            "image/",
          ),
      );

    if (
      files.length ===
      0
    ) {
      return;
    }

    event.preventDefault();

    void addImageFiles(
      files,
      target,
    );
  }

  function removeAttachment(
    attachmentID: string,
    target:
      AttachmentTarget,
  ) {
    if (
      target ===
      "create"
    ) {
      setCreateAttachments(
        (
          current,
        ) =>
          current.filter(
            (
              attachment,
            ) =>
              attachment.id !==
              attachmentID,
          ),
      );

      return;
    }

    setMessageAttachments(
      (
        current,
      ) =>
        current.filter(
          (
            attachment,
          ) =>
            attachment.id !==
            attachmentID,
        ),
    );
  }

  function addMessageLink() {
    const url =
      normaliseURL(
        linkInput,
      );

    if (!url) {
      setError(
        "Enter a valid http:// or https:// link.",
      );

      return;
    }

    if (
      messageAttachments.length >=
      MAX_ATTACHMENTS
    ) {
      setError(
        `You can attach up to ${MAX_ATTACHMENTS} references at a time.`,
      );

      return;
    }

    const candidate:
      RequestAttachment[] =
        [
          ...messageAttachments,

          {
            id:
              makeAttachmentID(),

            kind:
              "link",

            name:
              url,

            url,
          },
        ];

    if (
      attachmentPayloadBytes(
        candidate,
      ) >
      MAX_ATTACHMENT_JSON_BYTES
    ) {
      setError(
        "These attachments exceed the Product Request attachment limit.",
      );

      return;
    }

    setMessageAttachments(
      candidate,
    );

    setLinkInput("");

    setLinkEditorOpen(
      false,
    );

    setAttachmentMenuOpen(
      false,
    );

    setError("");
  }

  /* =======================================================
     CREATE REQUEST
     ======================================================= */

  async function createRequest(
    event:
      FormEvent<HTMLFormElement>,
  ) {
    event.preventDefault();

    if (
      busy ||
      attachmentBusy
    ) {
      return;
    }

    const quantity =
      Number(
        createForm
          .requested_quantity,
      );

    if (
      !createForm
        .requested_product_name
        .trim() ||
      !createForm
        .description
        .trim() ||
      !Number.isInteger(
        quantity,
      ) ||
      quantity <= 0
    ) {
      setError(
        "Add the product name, description, and a valid requested quantity.",
      );

      return;
    }

    const externalURL =
      createForm
        .external_url
        .trim();

    if (
      externalURL &&
      !normaliseURL(
        externalURL,
      )
    ) {
      setError(
        "Enter a valid reference URL.",
      );

      return;
    }

    setBusy(
      "create",
    );

    setError("");

    try {
      const response =
        await fetch(
          "/api/storefront/account/requests",
          {
            method:
              "POST",

            headers: {
              "Content-Type":
                "application/json",
            },

            body:
              JSON.stringify({
                requested_product_name:
                  createForm
                    .requested_product_name
                    .trim(),

                description:
                  createForm
                    .description
                    .trim(),

                requested_quantity:
                  quantity,

                customer_requirements:
                  createForm
                    .requirements
                    .trim()
                    ? {
                        notes:
                          createForm
                            .requirements
                            .trim(),
                      }
                    : {},

                external_url:
                  externalURL,

                attachments:
                  serialiseAttachments(
                    createAttachments,
                  ),
              }),
          },
        );

      if (
        !response.ok
      ) {
        throw new Error(
          await readError(
            response,
            "Unable to create your product request.",
          ),
        );
      }

      const payload =
        (await response.json()) as
          ProductRequestDataResponse<ProductRequest>;

      setCreateForm(
        emptyCreateForm,
      );

      setCreateAttachments(
        [],
      );

      setShowCreate(
        false,
      );

      setSelectedID(
        payload.data.id,
      );

      setMobileDetailOpen(
        true,
      );

      await Promise.all([
        loadRequests(
          payload.data.id,
          true,
        ),

        loadDetail(
          payload.data.id,
        ),
      ]);
    } catch (
      caught
    ) {
      setError(
        caught instanceof
          Error
          ? caught.message
          : "Unable to create your product request.",
      );
    } finally {
      setBusy("");
    }
  }

  /* =======================================================
     SEND CUSTOMER MESSAGE
     ======================================================= */

  async function sendMessage(
    event:
      FormEvent<HTMLFormElement>,
  ) {
    event.preventDefault();

    if (
      !selected ||
      !selected.can_message ||
      busy ||
      attachmentBusy
    ) {
      return;
    }

    const typed =
      message.trim();

    if (
      !typed &&
      messageAttachments.length ===
        0
    ) {
      return;
    }

    /*
     * Keep a non-empty body for attachment-only messages.
     */
    const body =
      typed ||
      attachmentFallback(
        messageAttachments,
      );

    setBusy(
      "message",
    );

    forceBottomRef.current = true;

    setError("");

    try {
      const response =
        await fetch(
          `/api/storefront/account/requests/${encodeURIComponent(
            selected.id,
          )}/messages`,
          {
            method:
              "POST",

            headers: {
              "Content-Type":
                "application/json",
            },

            body:
              JSON.stringify({
                message:
                  body,

                attachments:
                  serialiseAttachments(
                    messageAttachments,
                  ),
              }),
          },
        );

      if (
        !response.ok
      ) {
        throw new Error(
          await readError(
            response,
            "Unable to send your message.",
          ),
        );
      }

      setMessage("");

      setMessageAttachments(
        [],
      );

      setAttachmentMenuOpen(
        false,
      );

      setLinkEditorOpen(
        false,
      );

      await Promise.all([
        loadDetail(
          selected.id,
        ),

        loadRequests(
          selected.id,
          true,
        ),
      ]);
    } catch (
      caught
    ) {
      setError(
        caught instanceof
          Error
          ? caught.message
          : "Unable to send your message.",
      );
    } finally {
      setBusy("");
    }
  }

  /* =======================================================
     OFFER ACTION
     ======================================================= */

  async function respondToOffer(
    offerID: string,
    action:
      | "accept"
      | "reject",
  ) {
    if (
      !selected ||
      busy
    ) {
      return;
    }

    setBusy(
      `${action}:${offerID}`,
    );

    forceBottomRef.current = true;

    setError("");

    try {
      const response =
        await fetch(
          `/api/storefront/account/requests/${encodeURIComponent(
            selected.id,
          )}/offers/${encodeURIComponent(
            offerID,
          )}/${action}`,
          {
            method:
              "POST",
          },
        );

      if (
        !response.ok
      ) {
        throw new Error(
          await readError(
            response,
            action ===
              "accept"
              ? "Unable to accept this offer."
              : "Unable to decline this offer.",
          ),
        );
      }

      await Promise.all([
        loadDetail(
          selected.id,
        ),

        loadRequests(
          selected.id,
          true,
        ),
      ]);
    } catch (
      caught
    ) {
      setError(
        caught instanceof
          Error
          ? caught.message
          : "Unable to update this offer.",
      );
    } finally {
      setBusy("");
    }
  }

  /* =======================================================
     REQUEST CONTEXT + UNIFIED SOURCING TIMELINE
     ======================================================= */

  function renderRequestContext() {
    if (!selected) {
      return null;
    }

    return (
      <div className={styles.requestContextContent}>
        <div className={styles.requestContextHeading}>
          <div>
            <span className={styles.cardEyebrow}>Your request</span>
            <h2>{selected.requested_product_name}</h2>
          </div>

          <span
            className={`${styles.requestStatus} ${statusClass(
              selected.status,
            )}`}
          >
            {statusLabel(selected.status)}
          </span>
        </div>

        <p className={styles.requestContextDescription}>
          {selected.description}
        </p>

        {selectedRequirements ? (
          <div className={styles.requestContextSection}>
            <span>Requirements</span>
            <p>{selectedRequirements}</p>
          </div>
        ) : null}

        {selectedReferences.length > 0 ? (
          <div className={styles.requestContextSection}>
            <span>References</span>

            <MessageAttachments
              value={selected.attachments}
              onPreview={(attachment) =>
                setPreviewImage({
                  url: attachment.url,
                  name: attachment.name,
                })
              }
            />
          </div>
        ) : null}

        {selected.external_url ? (
          <a
            className={styles.requestContextLink}
            href={selected.external_url}
            target="_blank"
            rel="noopener noreferrer"
          >
            <span>Open product reference</span>
            <span aria-hidden="true">↗</span>
          </a>
        ) : null}

        <dl className={styles.requestContextFacts}>
          <div>
            <dt>Request</dt>
            <dd>{selected.request_number}</dd>
          </div>

          <div>
            <dt>Quantity</dt>
            <dd>{selected.requested_quantity.toLocaleString()}</dd>
          </div>

          <div>
            <dt>Created</dt>
            <dd>{formatShortDate(selected.created_at)}</dd>
          </div>

          <div>
            <dt>Conversation</dt>
            <dd>{selected.can_message ? "Open" : "Closed"}</dd>
          </div>
        </dl>

        {selected.status_reason ? (
          <div className={styles.requestContextNotice}>
            <strong>Status note</strong>
            <span>{selected.status_reason}</span>
          </div>
        ) : null}
      </div>
    );
  }

  function renderTimelineItem(
    item: CustomerSourcingTimelineItem,
  ) {
    if (!selected) {
      return null;
    }

    if (item.kind === "request") {
      const references = entityAttachments(item.request);
      const requirements = readRequirements(item.request);

      return (
        <article
          key={item.id}
          className={styles.timelineRequestCard}
        >
          <div className={styles.timelineCardTopline}>
            <div>
              <span className={styles.cardEyebrow}>Request submitted</span>
              <strong>{item.request.requested_product_name}</strong>
            </div>

            <time dateTime={item.at}>{formatDate(item.at)}</time>
          </div>

          <p>{item.request.description}</p>

          <div className={styles.timelineRequestMeta}>
            <span>
              Quantity <strong>{item.request.requested_quantity.toLocaleString()}</strong>
            </span>

            {requirements ? <span>{requirements}</span> : null}
          </div>

          {references.length > 0 ? (
            <MessageAttachments
              value={item.request.attachments}
              onPreview={(attachment) =>
                setPreviewImage({
                  url: attachment.url,
                  name: attachment.name,
                })
              }
            />
          ) : null}

          {item.request.external_url ? (
            <a
              className={styles.timelineReferenceLink}
              href={item.request.external_url}
              target="_blank"
              rel="noopener noreferrer"
            >
              Product reference <span aria-hidden="true">↗</span>
            </a>
          ) : null}
        </article>
      );
    }

    if (item.kind === "message") {
      const support = item.message.author_type === "support";

      return (
        <div
          key={item.id}
          className={`${styles.messageLine} ${
            support ? styles.messageLineSupport : styles.messageLineCustomer
          } ${styles.timelineMessageLine}`}
        >
          {support ? (
            <span className={styles.messageAvatar}>
              <Icon name="request" size={13} />
            </span>
          ) : null}

          <div className={styles.messageGroup}>
            <div
              className={`${styles.messageBubble} ${
                support ? styles.supportMessage : styles.customerMessage
              }`}
            >
              {item.message.body ? (
                <p>
                  <MessageText text={item.message.body} />
                </p>
              ) : null}

              <MessageAttachments
                value={item.message.attachments}
                onPreview={(attachment) =>
                  setPreviewImage({
                    url: attachment.url,
                    name: attachment.name,
                  })
                }
              />
            </div>

            <span className={styles.messageMeta}>
              {support
                ? item.message.support_actor_name || "Ene Dei Sourcing"
                : "You"}
              {" · "}
              {formatDate(item.message.created_at)}
            </span>
          </div>

          {!support ? renderCustomerAvatar() : null}
        </div>
      );
    }

    if (item.kind === "offer_sent") {
      const { offer } = item;
      const quantity = offer.quoted_quantity ?? selected.requested_quantity;
      const total = offer.unit_price * quantity + offer.shipping_price;
      const actionable = offer.status === "sent";
      const attachments = entityAttachments(offer);
      const specifications = Object.entries(offer.offered_specifications ?? {});

      return (
        <div
          key={item.id}
          className={styles.timelineCommercial}
        >
          <article
            className={`${styles.timelineOfferCard} ${
              actionable ? styles.timelineOfferCardActive : ""
            }`}
          >
            <div className={styles.timelineOfferTopline}>
              <div>
                <span className={styles.cardEyebrow}>
                  Commercial offer · Offer {item.offerNumber}
                </span>
                <h3>{offer.product_name}</h3>
              </div>

              <span
                className={`${styles.offerStatus} ${statusClass(offer.status)}`}
              >
                {statusLabel(offer.status)}
              </span>
            </div>

            {offer.description ? (
              <p className={styles.timelineOfferDescription}>
                {offer.description}
              </p>
            ) : null}

            {specifications.length > 0 ? (
              <dl className={styles.timelineSpecifications}>
                {specifications.map(([key, value]) => (
                  <div key={key}>
                    <dt>{key.replaceAll("_", " ")}</dt>
                    <dd>{formatSpecificationValue(value)}</dd>
                  </div>
                ))}
              </dl>
            ) : null}

            {attachments.length > 0 ? (
              <MessageAttachments
                value={offer.attachments}
                onPreview={(attachment) =>
                  setPreviewImage({
                    url: attachment.url,
                    name: attachment.name,
                  })
                }
              />
            ) : null}

            <dl className={styles.timelineOfferFacts}>
              <div>
                <dt>Quantity</dt>
                <dd>{quantity.toLocaleString()}</dd>
              </div>
              <div>
                <dt>MOQ</dt>
                <dd>
                  {offer.minimum_order_quantity?.toLocaleString() ?? "—"}
                </dd>
              </div>
              <div>
                <dt>Unit price</dt>
                <dd>{formatMoney(offer.unit_price, offer.currency)}</dd>
              </div>
              <div>
                <dt>Shipping</dt>
                <dd>{formatMoney(offer.shipping_price, offer.currency)}</dd>
              </div>
              <div className={styles.timelineOfferTotal}>
                <dt>Total</dt>
                <dd>{formatMoney(total, offer.currency)}</dd>
              </div>
            </dl>

            {offer.expires_at ? (
              <small className={styles.timelineOfferExpiry}>
                Expires {formatDate(offer.expires_at)}
              </small>
            ) : null}

            {actionable ? (
              <div className={styles.timelineOfferActions}>
                <button
                  type="button"
                  className={styles.secondaryButton}
                  disabled={Boolean(busy)}
                  onClick={() => void respondToOffer(offer.id, "reject")}
                >
                  {busy === `reject:${offer.id}` ? "Declining…" : "Decline"}
                </button>

                <button
                  type="button"
                  className={styles.primaryButton}
                  disabled={Boolean(busy)}
                  onClick={() => void respondToOffer(offer.id, "accept")}
                >
                  {busy === `accept:${offer.id}` ? "Accepting…" : "Accept offer"}
                </button>
              </div>
            ) : null}
          </article>

          <time dateTime={item.at}>{formatDate(item.at)}</time>
        </div>
      );
    }

    if (item.kind === "offer_response") {
      const accepted = item.response === "accepted";

      return (
        <div
          key={item.id}
          className={`${styles.timelineEvent} ${
            accepted ? styles.timelineEventGood : styles.timelineEventDanger
          }`}
        >
          <span className={styles.timelineEventIcon} aria-hidden="true">
            {accepted ? "✓" : "×"}
          </span>

          <div>
            <strong>
              {accepted
                ? `You accepted Offer ${item.offerNumber}`
                : `You declined Offer ${item.offerNumber}`}
            </strong>
            <span>
              {accepted
                ? "The sourcing team will review and finalize the accepted terms."
                : "Negotiation can continue with a revised offer."}
            </span>
          </div>

          <time dateTime={item.at}>{formatDate(item.at)}</time>
        </div>
      );
    }

    if (item.kind === "agreement") {
      const { confirmation: agreement } = item;

      return (
        <div
          key={item.id}
          className={styles.timelineCommercial}
        >
          <article className={styles.timelineAgreementCard}>
            <div className={styles.timelineAgreementHeading}>
              <span className={styles.timelineAgreementIcon} aria-hidden="true">
                ✓
              </span>

              <div>
                <span className={styles.cardEyebrow}>Agreement confirmed</span>
                <h3>{agreement.accepted_product_name}</h3>
              </div>

              <span className={styles.confirmationBadge}>Locked</span>
            </div>

            <p>
              These commercial terms are finalized and cannot be changed during checkout.
            </p>

            <dl className={styles.timelineOfferFacts}>
              <div>
                <dt>Quantity</dt>
                <dd>{agreement.quantity.toLocaleString()}</dd>
              </div>
              <div>
                <dt>MOQ</dt>
                <dd>{agreement.minimum_order_quantity.toLocaleString()}</dd>
              </div>
              <div>
                <dt>Unit price</dt>
                <dd>
                  {formatMoney(
                    agreement.unit_price_snapshot,
                    agreement.currency,
                  )}
                </dd>
              </div>
              <div>
                <dt>Shipping</dt>
                <dd>
                  {formatMoney(
                    agreement.shipping_price_snapshot,
                    agreement.currency,
                  )}
                </dd>
              </div>
              <div className={styles.timelineOfferTotal}>
                <dt>Total</dt>
                <dd>{formatMoney(agreement.total_amount, agreement.currency)}</dd>
              </div>
            </dl>

            {!agreement.created_order_id ? (
              <Link
                className={styles.checkoutButton}
                href={`/account/request/${selected.id}/checkout`}
              >
                Continue to checkout
                <Icon name="chevronRight" size={14} />
              </Link>
            ) : null}
          </article>

          <time dateTime={item.at}>{formatDate(item.at)}</time>
        </div>
      );
    }

    return (
      <div
        key={item.id}
        className={`${styles.timelineEvent} ${styles.timelineEventGood}`}
      >
        <span className={styles.timelineEventIcon} aria-hidden="true">✓</span>

        <div>
          <strong>Sourcing order created</strong>
          <span>Your sourcing agreement has been converted into an order.</span>
        </div>

        <Link
          className={styles.timelineOrderLink}
          href={`/account/orders/${item.confirmation.created_order_id}`}
        >
          View order
          <Icon name="chevronRight" size={13} />
        </Link>
      </div>
    );
  }

  /* =======================================================
     AVATAR
     ======================================================= */

  function renderCustomerAvatar() {
    if (
      customer?.avatar_url
    ) {
      return (
        <span
          className={
            styles.customerAvatar
          }
        >
          <img
            src={
              customer.avatar_url
            }
            alt=""
          />
        </span>
      );
    }

    return (
      <span
        className={
          styles.customerAvatar
        }
      >
        {
          customerAvatarText
        }
      </span>
    );
  }

  /* =======================================================
     DRAFT ATTACHMENTS
     ======================================================= */

  function renderDraftAttachments(
    attachments:
      RequestAttachment[],
    target:
      AttachmentTarget,
  ) {
    if (
      attachments.length ===
      0
    ) {
      return null;
    }

    return (
      <div
        className={
          styles.draftAttachments
        }
      >
        {attachments.map(
          (attachment) => (
            <div
              key={
                attachment.id
              }
              className={
                styles.draftAttachment
              }
            >
              {attachment.kind ===
              "image" ? (
                <img
                  src={
                    attachment.url
                  }
                  alt=""
                />
              ) : (
                <span
                  className={
                    styles.draftLinkIcon
                  }
                  aria-hidden="true"
                >
                  ↗
                </span>
              )}

              <span
                className={
                  styles.draftAttachmentName
                }
              >
                {
                  attachment.name
                }
              </span>

              <button
                type="button"
                aria-label={`Remove ${attachment.name}`}
                onClick={() =>
                  removeAttachment(
                    attachment.id,
                    target,
                  )
                }
              >
                ×
              </button>
            </div>
          ),
        )}
      </div>
    );
  }

  /* =======================================================
     SIGNED OUT
     ======================================================= */

  if (
    signedOut
  ) {
    return (
      <main
        className={
          styles.page
        }
      >
        <section
          className={
            styles.signedOutState
          }
        >
          <span
            className={
              styles.largeRequestIcon
            }
          >
            <Icon
              name="request"
              size={25}
            />
          </span>

          <h1>
            Sign in to manage product requests.
          </h1>

          <p>
            Your sourcing conversations, offers and agreements are tied to your account.
          </p>

          <Link
            className={
              styles.primaryButton
            }
            href="/account/sign-in?next=%2Faccount%2Frequest"
          >
            Sign in
          </Link>
        </section>
      </main>
    );
  }

  /* =======================================================
     MAIN UI
     ======================================================= */

  return (
    <main
      className={
        styles.page
      }
    >
      <div
        ref={shellRef}
        className={
          styles.shell
        }
      >
        {/* =================================================
            COMPACT DESKTOP HEADER
            ================================================= */}

        <header
          className={
            styles.topBar
          }
        >
          <div
            className={
              styles.topIdentity
            }
          >
            <Link
              href="/account"
              className={
                styles.accountBack
              }
            >
              <span
                aria-hidden="true"
              >
                ←
              </span>

              Account
            </Link>

            <div
              className={
                styles.topBarText
              }
            >
              <span
                className={
                  styles.eyebrow
                }
              >
                Product sourcing
              </span>

              <h1>
                Product Request
              </h1>

              <p>
                Request products beyond the catalog and agree sourcing terms with our team.
              </p>
            </div>
          </div>

          <button
            type="button"
            className={
              styles.primaryButton
            }
            onClick={
              openNewRequest
            }
          >
            <Icon
              name="plus"
              size={15}
            />

            New request
          </button>
        </header>

        {/* =================================================
            ERROR
            ================================================= */}

        {error &&
        !showCreate ? (
          <div
            className={
              styles.errorBanner
            }
            role="alert"
          >
            <span>
              {error}
            </span>

            <button
              type="button"
              aria-label="Dismiss error"
              onClick={() =>
                setError("")
              }
            >
              ×
            </button>
          </div>
        ) : null}

        {/* =================================================
            WORKSPACE
            ================================================= */}

        <div
          className={`${styles.requestWorkspace} ${
            mobileDetailOpen
              ? styles.mobileDetailOpen
              : ""
          }`}
        >
          {/* ===============================================
              REQUEST LIST
              =============================================== */}

          <aside
            className={
              styles.requestRail
            }
          >
            <div
              className={
                styles.railHead
              }
            >
              <div>
                <strong>
                  Requests
                </strong>

                <span>
                  Latest activity first
                </span>
              </div>

              <span
                className={
                  styles.countBadge
                }
              >
                {
                  requests.length
                }
              </span>
            </div>

            <div
              className={
                styles.requestList
              }
            >
              {loading ? (
                <div
                  className={
                    styles.railLoading
                  }
                >
                  Loading requests…
                </div>
              ) : null}

              {!loading &&
              requests.length ===
                0 ? (
                <div
                  className={
                    styles.emptyRail
                  }
                >
                  <span
                    className={
                      styles.largeRequestIcon
                    }
                  >
                    <Icon
                      name="request"
                      size={20}
                    />
                  </span>

                  <strong>
                    No requests yet
                  </strong>

                  <p>
                    Request something when you cannot find it in the catalog.
                  </p>

                  <button
                    type="button"
                    className={
                      styles.secondaryButton
                    }
                    onClick={
                      openNewRequest
                    }
                  >
                    New request
                  </button>
                </div>
              ) : null}

              {requests.map(
                (
                  request,
                ) => {
                  const active =
                    request.id ===
                    selectedID;

                  return (
                    <button
                      key={
                        request.id
                      }
                      type="button"
                      className={`${styles.requestItem} ${
                        active
                          ? styles.requestItemActive
                          : ""
                      }`}
                      onClick={() =>
                        openRequest(
                          request.id,
                        )
                      }
                    >
                      <span
                        className={
                          styles.requestItemTop
                        }
                      >
                        <span
                          className={
                            styles.requestItemTitle
                          }
                        >
                          {
                            request.requested_product_name
                          }
                        </span>

                        <small>
                          {formatShortDate(
                            request.updated_at,
                          )}
                        </small>
                      </span>

                      <span
                        className={
                          styles.requestNumber
                        }
                      >
                        {
                          request.request_number
                        }
                      </span>

                      <span
                        className={
                          styles.requestItemBottom
                        }
                      >
                        <span
                          className={`${styles.requestStatus} ${statusClass(
                            request.status,
                          )}`}
                        >
                          {statusLabel(
                            request.status,
                          )}
                        </span>

                        <span
                          className={
                            styles.requestMeta
                          }
                        >
                          {request.requested_quantity.toLocaleString()} units
                        </span>
                      </span>
                    </button>
                  );
                },
              )}
            </div>
          </aside>

          {/* ===============================================
              REQUEST DETAIL
              =============================================== */}

          <section
            className={
              styles.detailColumn
            }
          >
            <button
              type="button"
              className={
                styles.mobileBack
              }
              onClick={() => {
                setMobileInfoOpen(false);
                setMobileDetailOpen(false);
              }}
            >
              <Icon
                name="arrowLeft"
                size={14}
              />

              Requests
            </button>

            {!selectedID &&
            !loading ? (
              <div
                className={
                  styles.emptyDetail
                }
              >
                <span
                  className={
                    styles.largeRequestIcon
                  }
                >
                  <Icon
                    name="request"
                    size={25}
                  />
                </span>

                <h2>
                  Source something new.
                </h2>

                <p>
                  Create a request, share product references, talk with the sourcing team and review commercial offers here.
                </p>

                <button
                  type="button"
                  className={
                    styles.primaryButton
                  }
                  onClick={
                    openNewRequest
                  }
                >
                  New product request
                </button>
              </div>
            ) : null}

            {detailLoading ? (
              <div
                className={
                  styles.loadingDetail
                }
              >
                Loading request…
              </div>
            ) : null}

            {selected &&
            !detailLoading ? (
              <>
                {/* =========================================
                    REQUEST HEADER
                    ========================================= */}

                <header
                  className={
                    styles.detailHeader
                  }
                >
                  <span
                    className={
                      styles.detailHeaderAvatar
                    }
                  >
                    <Icon
                      name="request"
                      size={18}
                    />
                  </span>

                  <div
                    className={
                      styles.detailHeaderText
                    }
                  >
                    <strong>
                      {
                        selected.requested_product_name
                      }
                    </strong>

                    <span>
                      {
                        selected.request_number
                      }
                      {" · "}
                      {selected.requested_quantity.toLocaleString()} units
                    </span>
                  </div>

                  <span
                    className={`${styles.detailStatus} ${statusClass(
                      selected.status,
                    )}`}
                  >
                    {statusLabel(
                      selected.status,
                    )}
                  </span>

                  <button
                    type="button"
                    className={styles.mobileDetailsButton}
                    onClick={() => setMobileInfoOpen(true)}
                  >
                    Details
                  </button>
                </header>

                {/* =========================================
                    SOURCING NEGOTIATION WORKSPACE

                    Desktop: request context + conversation.
                    Mobile: conversation first; details on demand.
                    ========================================= */}

                <div className={styles.detailContent}>
                  <aside className={styles.requestContextPanel}>
                    {renderRequestContext()}
                  </aside>

                  <section
                    className={`${styles.conversationCard} ${styles.sourcingConversationCard}`}
                  >
                    <div className={styles.conversationHeader}>
                      <div>
                        <span className={styles.cardEyebrow}>
                          Sourcing conversation
                        </span>

                        <h2>Ene Dei sourcing team</h2>
                      </div>

                      <span
                        className={`${styles.conversationState} ${
                          selected.can_message ? "" : styles.conversationStateClosed
                        }`}
                      >
                        {selected.can_message ? "Open" : "Read only"}
                      </span>
                    </div>

                    <div
                      ref={threadRef}
                      className={`${styles.thread} ${styles.sourcingTimeline}`}
                      aria-live="polite"
                      onScroll={(event) => {
                        const element = event.currentTarget;
                        const distanceFromBottom =
                          element.scrollHeight -
                          element.scrollTop -
                          element.clientHeight;

                        stickToBottomRef.current = distanceFromBottom < 120;
                      }}
                    >
                      {selected.status_reason &&
                      (selected.status === "on_hold" ||
                        selected.status === "cancelled") ? (
                        <div className={styles.timelineStatusNotice}>
                          <strong>{statusLabel(selected.status)}</strong>
                          <span>{selected.status_reason}</span>
                        </div>
                      ) : null}

                      <div className={styles.timelineStream}>
                        {timelineItems.map(renderTimelineItem)}

                        <div ref={messagesEndRef} />
                      </div>
                    </div>

                    {selected.can_message ? (
                      <form
                        className={styles.composer}
                        onSubmit={sendMessage}
                        onDragOver={(event) => event.preventDefault()}
                        onDrop={(event) => handleDrop(event, "message")}
                      >
                        {renderDraftAttachments(
                          messageAttachments,
                          "message",
                        )}

                        <input
                          ref={messageFileInputRef}
                          className={styles.hiddenInput}
                          type="file"
                          accept="image/*"
                          multiple
                          onChange={(event) =>
                            handleFileChange(event, "message")
                          }
                        />

                        <div
                          ref={attachmentMenuRef}
                          className={styles.composerShell}
                        >
                          {attachmentMenuOpen ? (
                            <div className={styles.attachmentMenu}>
                              <button
                                type="button"
                                disabled={
                                  messageAttachments.length >= MAX_ATTACHMENTS
                                }
                                onClick={() =>
                                  messageFileInputRef.current?.click()
                                }
                              >
                                <span className={styles.attachmentMenuIcon}>
                                  <ImageIcon />
                                </span>

                                <span>
                                  <strong>Photo or screenshot</strong>
                                  <small>Add a product reference</small>
                                </span>
                              </button>

                              <button
                                type="button"
                                disabled={
                                  messageAttachments.length >= MAX_ATTACHMENTS
                                }
                                onClick={() => {
                                  setAttachmentMenuOpen(false);
                                  setLinkEditorOpen(true);
                                }}
                              >
                                <span className={styles.attachmentMenuIcon}>↗</span>

                                <span>
                                  <strong>Add link</strong>
                                  <small>Share a web reference</small>
                                </span>
                              </button>
                            </div>
                          ) : null}

                          {linkEditorOpen ? (
                            <div className={styles.linkPopover}>
                              <strong>Add reference link</strong>

                              <div>
                                <input
                                  type="url"
                                  autoFocus
                                  value={linkInput}
                                  placeholder="https://..."
                                  onChange={(event) =>
                                    setLinkInput(event.target.value)
                                  }
                                  onKeyDown={(event) => {
                                    if (event.key === "Enter") {
                                      event.preventDefault();
                                      addMessageLink();
                                    }

                                    if (event.key === "Escape") {
                                      setLinkEditorOpen(false);
                                    }
                                  }}
                                />

                                <button
                                  type="button"
                                  onClick={addMessageLink}
                                >
                                  Add
                                </button>
                              </div>
                            </div>
                          ) : null}

                          <div className={styles.composerBox}>
                            <button
                              type="button"
                              className={styles.composerAttachmentButton}
                              aria-label="Add attachment"
                              disabled={attachmentBusy}
                              onClick={() => {
                                setLinkEditorOpen(false);
                                setAttachmentMenuOpen((current) => !current);
                              }}
                            >
                              <PaperclipIcon />
                            </button>

                            <textarea
                              className={styles.composerTextarea}
                              value={message}
                              rows={1}
                              maxLength={5000}
                              placeholder="Write a message…"
                              onPaste={(event) =>
                                handlePaste(event, "message")
                              }
                              onChange={(event) =>
                                setMessage(event.target.value)
                              }
                            />

                            <button
                              type="submit"
                              className={styles.sendButton}
                              disabled={
                                Boolean(busy) ||
                                attachmentBusy ||
                                (!message.trim() &&
                                  messageAttachments.length === 0)
                              }
                            >
                              {busy === "message" ? "Sending…" : "Send"}
                            </button>
                          </div>
                        </div>
                      </form>
                    ) : (
                      <div className={styles.closedNote}>
                        {selected.status === "converted_to_order"
                          ? "This sourcing conversation is complete."
                          : "This request conversation is closed."}
                      </div>
                    )}
                  </section>
                </div>

                {mobileInfoOpen ? (
                  <div
                    className={styles.mobileInfoOverlay}
                    role="presentation"
                    onMouseDown={(event) => {
                      if (event.target === event.currentTarget) {
                        setMobileInfoOpen(false);
                      }
                    }}
                  >
                    <aside
                      className={styles.mobileInfoSheet}
                      role="dialog"
                      aria-modal="true"
                      aria-label="Product request details"
                    >
                      <div className={styles.mobileInfoHandle} />

                      <header className={styles.mobileInfoHeader}>
                        <div>
                          <span className={styles.cardEyebrow}>Request details</span>
                          <h2>{selected.requested_product_name}</h2>
                        </div>

                        <button
                          type="button"
                          aria-label="Close request details"
                          onClick={() => setMobileInfoOpen(false)}
                        >
                          ×
                        </button>
                      </header>

                      {renderRequestContext()}
                    </aside>
                  </div>
                ) : null}
              </>
            ) : null}
          </section>
        </div>
      </div>

      {/* ===================================================
          CREATE REQUEST MODAL
          =================================================== */}

      {showCreate ? (
        <div
          className={
            styles.createOverlay
          }
          role="dialog"
          aria-modal="true"
          aria-label="New product request"
          onMouseDown={(
            event,
          ) => {
            if (
              event.target ===
              event.currentTarget
            ) {
              closeNewRequest();
            }
          }}
        >
          <section
            className={
              styles.createPanel
            }
            onDragOver={(
              event,
            ) =>
              event.preventDefault()
            }
            onDrop={(
              event,
            ) =>
              handleDrop(
                event,
                "create",
              )
            }
          >
            <header
              className={
                styles.createHeader
              }
            >
              <div
                className={
                  styles.createHeaderText
                }
              >
                <span
                  className={
                    styles.eyebrow
                  }
                >
                  New sourcing request
                </span>

                <h2>
                  What are you looking for?
                </h2>

                <p>
                  Give us the product details and add photos or screenshots that help identify it.
                </p>
              </div>

              <button
                type="button"
                className={
                  styles.closeButton
                }
                aria-label="Close"
                onClick={
                  closeNewRequest
                }
              >
                ×
              </button>
            </header>

            <form
              className={
                styles.createForm
              }
              onSubmit={
                createRequest
              }
            >
              <div
                className={
                  styles.formRow
                }
              >
                <label
                  className={
                    styles.field
                  }
                >
                  <span>
                    Product name
                  </span>

                  <input
                    value={
                      createForm.requested_product_name
                    }
                    maxLength={
                      180
                    }
                    placeholder="Product, model, brand or exact item"
                    onChange={(
                      event,
                    ) =>
                      updateCreateField(
                        "requested_product_name",
                        event.target.value,
                      )
                    }
                  />
                </label>

                <label
                  className={`${styles.field} ${styles.quantityField}`}
                >
                  <span>
                    Quantity
                  </span>

                  <input
                    value={
                      createForm.requested_quantity
                    }
                    type="number"
                    min={1}
                    step={1}
                    inputMode="numeric"
                    onChange={(
                      event,
                    ) =>
                      updateCreateField(
                        "requested_quantity",
                        event.target.value,
                      )
                    }
                  />
                </label>
              </div>

              <label
                className={`${styles.field} ${styles.wide}`}
              >
                <span>
                  Description
                </span>

                <textarea
                  value={
                    createForm.description
                  }
                  maxLength={
                    5000
                  }
                  placeholder="Describe the exact item, intended use, acceptable alternatives, condition, region or anything else that helps us source it."
                  onPaste={(
                    event,
                  ) =>
                    handlePaste(
                      event,
                      "create",
                    )
                  }
                  onChange={(
                    event,
                  ) =>
                    updateCreateField(
                      "description",
                      event.target.value,
                    )
                  }
                />
              </label>

              <div
                className={
                  styles.formRow
                }
              >
                <label
                  className={
                    styles.field
                  }
                >
                  <span>
                    Reference URL{" "}
                    <small>
                      optional
                    </small>
                  </span>

                  <input
                    value={
                      createForm.external_url
                    }
                    type="url"
                    inputMode="url"
                    placeholder="https://..."
                    onChange={(
                      event,
                    ) =>
                      updateCreateField(
                        "external_url",
                        event.target.value,
                      )
                    }
                  />
                </label>

                <label
                  className={
                    styles.field
                  }
                >
                  <span>
                    Requirements{" "}
                    <small>
                      optional
                    </small>
                  </span>

                  <input
                    value={
                      createForm.requirements
                    }
                    placeholder="Color, size, packaging, deadline…"
                    onChange={(
                      event,
                    ) =>
                      updateCreateField(
                        "requirements",
                        event.target.value,
                      )
                    }
                  />
                </label>
              </div>

              {/* ===========================================
                  CREATE REQUEST IMAGE SECTION
                  =========================================== */}

              <div
                className={
                  styles.referenceUpload
                }
              >
                <div
                  className={
                    styles.referenceUploadHeading
                  }
                >
                  <div>
                    <strong>
                      Reference images
                    </strong>

                    <span>
                      Optional · up to {MAX_ATTACHMENTS} compressed images
                    </span>
                  </div>

                  <button
                    type="button"
                    className={
                      styles.secondaryButton
                    }
                    disabled={
                      attachmentBusy ||
                      createAttachments.length >=
                        MAX_ATTACHMENTS
                    }
                    onClick={() =>
                      createFileInputRef.current?.click()
                    }
                  >
                    <ImageIcon />

                    Add images
                  </button>
                </div>

                <input
                  ref={
                    createFileInputRef
                  }
                  className={
                    styles.hiddenInput
                  }
                  type="file"
                  accept="image/*"
                  multiple
                  onChange={(
                    event,
                  ) =>
                    handleFileChange(
                      event,
                      "create",
                    )
                  }
                />

                {createAttachments.length ===
                0 ? (
                  <button
                    type="button"
                    className={
                      styles.uploadSurface
                    }
                    onClick={() =>
                      createFileInputRef.current?.click()
                    }
                  >
                    <span
                      className={
                        styles.uploadIcon
                      }
                    >
                      <ImageIcon />
                    </span>

                    <span
                      className={
                        styles.uploadCopy
                      }
                    >
                      <strong>
                        Add product photos or screenshots
                      </strong>

                      <span>
                        Click here, paste a screenshot into the description, or drag images onto this form.
                      </span>
                    </span>
                  </button>
                ) : (
                  <div
                    className={
                      styles.attachmentGrid
                    }
                  >
                    {createAttachments.map(
                      (
                        attachment,
                      ) => (
                        <div
                          key={
                            attachment.id
                          }
                          className={
                            styles.attachmentPreview
                          }
                        >
                          <button
                            type="button"
                            className={
                              styles.attachmentPreviewImage
                            }
                            onClick={() =>
                              setPreviewImage({
                                url:
                                  attachment.url,

                                name:
                                  attachment.name,
                              })
                            }
                          >
                            <img
                              src={
                                attachment.url
                              }
                              alt={
                                attachment.name
                              }
                            />
                          </button>

                          <button
                            type="button"
                            className={
                              styles.attachmentRemove
                            }
                            aria-label={`Remove ${attachment.name}`}
                            onClick={() =>
                              removeAttachment(
                                attachment.id,
                                "create",
                              )
                            }
                          >
                            ×
                          </button>
                        </div>
                      ),
                    )}
                  </div>
                )}
              </div>

              {error ? (
                <p
                  className={
                    styles.createError
                  }
                  role="alert"
                >
                  {error}
                </p>
              ) : null}

              <div
                className={
                  styles.formActions
                }
              >
                <button
                  type="button"
                  className={
                    styles.secondaryButton
                  }
                  disabled={
                    busy ===
                      "create" ||
                    attachmentBusy
                  }
                  onClick={
                    closeNewRequest
                  }
                >
                  Cancel
                </button>

                <button
                  type="submit"
                  className={
                    styles.primaryButton
                  }
                  disabled={
                    busy ===
                      "create" ||
                    attachmentBusy
                  }
                >
                  {busy ===
                  "create"
                    ? "Submitting…"
                    : attachmentBusy
                      ? "Preparing image…"
                      : "Submit request"}
                </button>
              </div>
            </form>
          </section>
        </div>
      ) : null}

      {/* ===================================================
          IMAGE LIGHTBOX
          =================================================== */}

      {previewImage ? (
        <div
          className={
            styles.imageLightbox
          }
          role="dialog"
          aria-modal="true"
          aria-label="Image preview"
          onClick={() =>
            setPreviewImage(
              null,
            )
          }
        >
          <button
            type="button"
            className={
              styles.imageLightboxClose
            }
            aria-label="Close image preview"
            onClick={() =>
              setPreviewImage(
                null,
              )
            }
          >
            ×
          </button>

          <div
            className={
              styles.imageLightboxContent
            }
            onClick={(
              event,
            ) =>
              event.stopPropagation()
            }
          >
            <img
              src={
                previewImage.url
              }
              alt={
                previewImage.name
              }
              className={
                styles.imageLightboxImage
              }
            />

            <span>
              {
                previewImage.name
              }
            </span>
          </div>
        </div>
      ) : null}
    </main>
  );
}