"use client";

import Link from "next/link";
import { useSearchParams } from "next/navigation";

import type {
  ChangeEvent,
  ClipboardEvent,
  DragEvent,
} from "react";

import {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";

import { Icon } from "@/components/ui/Icon";

import { useCustomerAuth } from "@/lib/account/use-customer-auth";

import type {
  AccountSupportCase,
  AccountSupportCaseResponse,
  AccountSupportCasesResponse,
  AccountSupportMessage,
  AccountSupportMessagesResponse,
} from "@/lib/api/contracts/account";

import styles from "../css/SupportCenter.module.css";

type ConversationMode =
  | "conversation"
  | "new";

type ComposerMode =
  | "create"
  | "reply";

type SupportTopic = {
  value: string;
  label: string;
  description: string;
};

type SupportAttachment = {
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

const MAX_ATTACHMENTS = 4;

const MAX_SOURCE_IMAGE_BYTES =
  8 * 1024 * 1024;

const MAX_DATA_URL_LENGTH =
  1_200_000;

const supportTopics: SupportTopic[] = [
  {
    value: "order_issue",
    label: "Order",
    description:
      "Questions or problems with an existing order.",
  },
  {
    value: "payment_issue",
    label: "Payment",
    description:
      "Payment attempts, status or payment method help.",
  },
  {
    value: "delivery_issue",
    label: "Delivery",
    description:
      "Shipping, tracking or delivery questions.",
  },
  {
    value: "return_issue",
    label: "Return",
    description:
      "Returns, refunds and eligibility questions.",
  },
  {
    value: "product_question",
    label: "Product",
    description:
      "Questions about a product in our catalog.",
  },
  {
    value: "general_question",
    label: "Account & other",
    description:
      "Account help or anything else you need.",
  },
];

function makeAttachmentID() {
  if (
    typeof crypto !== "undefined" &&
    typeof crypto.randomUUID ===
      "function"
  ) {
    return crypto.randomUUID();
  }

  return `${Date.now()}-${Math.random()
    .toString(36)
    .slice(2)}`;
}

function getTopic(
  caseType?: string,
) {
  return (
    supportTopics.find(
      (topic) =>
        topic.value ===
        caseType,
    ) ??
    supportTopics[
      supportTopics.length - 1
    ]
  );
}

function getCaseTopic(
  supportCase: AccountSupportCase,
) {
  return getTopic(
    supportCase.case_type ||
      supportCase.type,
  );
}

function formatDate(
  value?: string,
) {
  if (!value) {
    return "";
  }

  const date =
    new Date(value);

  if (
    Number.isNaN(
      date.getTime(),
    )
  ) {
    return "";
  }

  return new Intl.DateTimeFormat(
    "en",
    {
      month: "short",
      day: "numeric",
    },
  ).format(date);
}

function formatMessageTime(
  value?: string,
) {
  if (!value) {
    return "";
  }

  const date =
    new Date(value);

  if (
    Number.isNaN(
      date.getTime(),
    )
  ) {
    return "";
  }

  return new Intl.DateTimeFormat(
    "en",
    {
      month: "short",
      day: "numeric",
      hour: "numeric",
      minute: "2-digit",
    },
  ).format(date);
}

function customerInitials(
  name?: string,
) {
  const parts =
    (name ?? "")
      .trim()
      .split(/\s+/)
      .filter(Boolean);

  if (
    parts.length === 0
  ) {
    return "U";
  }

  if (
    parts.length === 1
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

function normaliseURL(
  input: string,
) {
  const trimmed =
    input.trim();

  try {
    const parsed =
      new URL(trimmed);

    if (
      parsed.protocol !==
        "https:" &&
      parsed.protocol !==
        "http:"
    ) {
      return null;
    }

    return parsed.toString();
  } catch {
    return null;
  }
}

function asObject(
  value: unknown,
): Record<
  string,
  unknown
> | null {
  if (
    !value ||
    typeof value !==
      "object" ||
    Array.isArray(value)
  ) {
    return null;
  }

  return value as Record<
    string,
    unknown
  >;
}

function parseAttachments(
  input: unknown,
): SupportAttachment[] {
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
    SupportAttachment[] =
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
            "Image attachment",

          url: raw,
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
      asObject(raw);

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
            ? "Image attachment"
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

function serialiseAttachments(
  attachments:
    SupportAttachment[],
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

function attachmentFallback(
  attachments:
    SupportAttachment[],
) {
  const imageCount =
    attachments.filter(
      (attachment) =>
        attachment.kind ===
        "image",
    ).length;

  if (
    imageCount > 0
  ) {
    return imageCount === 1
      ? "Shared an image."
      : `Shared ${imageCount} images.`;
  }

  return "Shared a link.";
}

function buildConversationSubject(
  topic: SupportTopic,
  message: string,
  linkedOrderID: string,
) {
  const preview =
    message
      .trim()
      .replace(
        /\s+/g,
        " ",
      )
      .slice(
        0,
        90,
      );

  const orderPart =
    linkedOrderID
      ? " · Order support"
      : "";

  return `${topic.label}${orderPart}: ${preview}`.slice(
    0,
    180,
  );
}

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
): Promise<SupportAttachment> {
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

  const original =
    await readFileAsDataURL(
      file,
    );

  const image =
    await loadBrowserImage(
      original,
    );

  const maxDimension =
    1280;

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

  let output =
    canvas.toDataURL(
      "image/jpeg",
      0.8,
    );

  if (
    output.length >
    MAX_DATA_URL_LENGTH
  ) {
    output =
      canvas.toDataURL(
        "image/jpeg",
        0.62,
      );
  }

  if (
    output.length >
    MAX_DATA_URL_LENGTH
  ) {
    throw new Error(
      "This image is still too large after compression. Try a smaller screenshot.",
    );
  }

  return {
    id:
      makeAttachmentID(),

    kind:
      "image",

    name:
      file.name ||
      `screenshot-${Date.now()}.jpg`,

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

function MessageAttachments({
  value,
  onPreview,
}: {
  value: unknown;

  onPreview: (
    attachment:
      SupportAttachment,
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

export function SupportCenterClient() {
  const searchParams =
    useSearchParams();

  const {
    customer,
  } =
    useCustomerAuth();

  const linkedOrderID =
    searchParams
      .get("order_id")
      ?.trim() ?? "";

  const [
    cases,
    setCases,
  ] = useState<
    AccountSupportCase[]
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
      AccountSupportCase | null
    >(null);

  const [
    messages,
    setMessages,
  ] =
    useState<
      AccountSupportMessage[]
    >([]);

  const [
    mode,
    setMode,
  ] =
    useState<ConversationMode>(
      linkedOrderID
        ? "new"
        : "conversation",
    );

  const [
    mobileDetailOpen,
    setMobileDetailOpen,
  ] =
    useState(
      Boolean(
        linkedOrderID,
      ),
    );

  const [
    topicValue,
    setTopicValue,
  ] =
    useState(
      linkedOrderID
        ? "order_issue"
        : "order_issue",
    );

  const [
    initialMessage,
    setInitialMessage,
  ] =
    useState("");

  const [
    reply,
    setReply,
  ] =
    useState("");

  const [
    attachments,
    setAttachments,
  ] =
    useState<
      SupportAttachment[]
    >([]);

  const [
    attachmentBusy,
    setAttachmentBusy,
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
    busy,
    setBusy,
  ] =
    useState("");

  const [
    error,
    setError,
  ] =
    useState("");

  const workspaceRef =
    useRef<HTMLDivElement | null>(
      null,
    );

  const fileInputRef =
    useRef<HTMLInputElement | null>(
      null,
    );

  const attachmentPopoverRef =
    useRef<HTMLDivElement | null>(
      null,
    );

  const messagesEndRef =
    useRef<HTMLDivElement | null>(
      null,
    );

  const selectedTopic =
    useMemo(
      () =>
        getTopic(
          topicValue,
        ),
      [
        topicValue,
      ],
    );

  const sortedMessages =
    useMemo(
      () =>
        [...messages].sort(
          (
            a,
            b,
          ) =>
            new Date(
              a.created_at,
            ).getTime() -
            new Date(
              b.created_at,
            ).getTime(),
        ),
      [
        messages,
      ],
    );

  const canReply =
    selected
      ? selected.status !==
        "closed"
      : false;

  const initials =
    customerInitials(
      customer?.full_name,
    );

  /*
   * Make the Support workspace consume exactly
   * the remaining desktop viewport height.
   *
   * On mobile the same values are used for the
   * full-screen conversation and keyboard.
   */
  useEffect(() => {
    const viewport =
      window.visualViewport;

    let frame = 0;

    function syncViewport() {
      window.cancelAnimationFrame(
        frame,
      );

      frame =
        window.requestAnimationFrame(
          () => {
            const element =
              workspaceRef.current;

            if (!element) {
              return;
            }

            const viewportHeight =
              viewport?.height ??
              window.innerHeight;

            const viewportTop =
              viewport?.offsetTop ??
              0;

            const rect =
              element.getBoundingClientRect();

            const relativeTop =
              rect.top -
              viewportTop;

            const availableHeight =
              Math.max(
                360,
                viewportHeight -
                  Math.max(
                    relativeTop,
                    0,
                  ) -
                  12,
              );

            element.style.setProperty(
              "--support-workspace-height",
              `${Math.round(
                availableHeight,
              )}px`,
            );

            element.style.setProperty(
              "--support-viewport-height",
              `${Math.round(
                viewportHeight,
              )}px`,
            );

            element.style.setProperty(
              "--support-viewport-top",
              `${Math.round(
                viewportTop,
              )}px`,
            );
          },
        );
    }

    syncViewport();

    window.addEventListener(
      "resize",
      syncViewport,
    );

    viewport?.addEventListener(
      "resize",
      syncViewport,
    );

    viewport?.addEventListener(
      "scroll",
      syncViewport,
    );

    return () => {
      window.cancelAnimationFrame(
        frame,
      );

      window.removeEventListener(
        "resize",
        syncViewport,
      );

      viewport?.removeEventListener(
        "resize",
        syncViewport,
      );

      viewport?.removeEventListener(
        "scroll",
        syncViewport,
      );
    };
  }, []);

  /*
   * Close attachment popover when clicking outside.
   */
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
        attachmentPopoverRef.current;

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

  /*
   * Image lightbox escape + page scroll lock.
   */
  useEffect(() => {
    if (!previewImage) {
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
        event.key ===
        "Escape"
      ) {
        setPreviewImage(
          null,
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
    previewImage,
  ]);

  const loadCases =
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
              "/api/storefront/account/support?limit=100&offset=0",
              {
                cache:
                  "no-store",
              },
            );

          if (
            !response.ok
          ) {
            throw new Error(
              await readError(
                response,
                "Unable to load your conversations.",
              ),
            );
          }

          const payload =
            (await response.json()) as AccountSupportCasesResponse;

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
            ) => {
              const aTime =
                new Date(
                  a.last_message_at ||
                    a.updated_at ||
                    a.created_at,
                ).getTime();

              const bTime =
                new Date(
                  b.last_message_at ||
                    b.updated_at ||
                    b.created_at,
                ).getTime();

              return (
                bTime -
                aTime
              );
            },
          );

          setCases(
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
                : "Unable to load your conversations.",
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

  const loadDetail =
    useCallback(
      async (
        caseID: string,
        silent = false,
      ) => {
        if (!caseID) {
          setSelected(
            null,
          );

          setMessages(
            [],
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
              caseID,
            );

          const [
            caseResponse,
            messageResponse,
          ] =
            await Promise.all([
              fetch(
                `/api/storefront/account/support/${encoded}`,
                {
                  cache:
                    "no-store",
                },
              ),

              fetch(
                `/api/storefront/account/support/${encoded}/messages?limit=200&offset=0`,
                {
                  cache:
                    "no-store",
                },
              ),
            ]);

          if (
            !caseResponse.ok
          ) {
            throw new Error(
              await readError(
                caseResponse,
                "Unable to open this conversation.",
              ),
            );
          }

          if (
            !messageResponse.ok
          ) {
            throw new Error(
              await readError(
                messageResponse,
                "Unable to load messages.",
              ),
            );
          }

          const casePayload =
            (await caseResponse.json()) as AccountSupportCaseResponse;

          const messagePayload =
            (await messageResponse.json()) as AccountSupportMessagesResponse;

          setSelected(
            casePayload.data,
          );

          setMessages(
            Array.isArray(
              messagePayload.data,
            )
              ? messagePayload.data
              : [],
          );
        } catch (
          caught
        ) {
          if (!silent) {
            setError(
              caught instanceof
                Error
                ? caught.message
                : "Unable to open this conversation.",
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

  useEffect(() => {
    void loadCases();
  }, [
    loadCases,
  ]);

  useEffect(() => {
    if (
      mode ===
        "conversation" &&
      selectedID
    ) {
      void loadDetail(
        selectedID,
      );
    }
  }, [
    loadDetail,
    mode,
    selectedID,
  ]);

  useEffect(() => {
    if (
      !selectedID ||
      mode !==
        "conversation"
    ) {
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

            loadCases(
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
    loadCases,
    loadDetail,
    mode,
    selectedID,
  ]);

  useEffect(() => {
    if (
      mode !==
      "conversation"
    ) {
      return;
    }

    messagesEndRef
      .current
      ?.scrollIntoView({
        block: "end",
      });
  }, [
    messages,
    mode,
  ]);

  function clearAttachments() {
    setAttachments(
      [],
    );

    setAttachmentMenuOpen(
      false,
    );

    setLinkEditorOpen(
      false,
    );

    setLinkInput("");
  }

  function startNewConversation(
    initialTopic?: string,
  ) {
    setMode(
      "new",
    );

    setTopicValue(
      initialTopic ||
        (linkedOrderID
          ? "order_issue"
          : "order_issue"),
    );

    setInitialMessage(
      "",
    );

    setReply("");

    setError("");

    clearAttachments();

    setMobileDetailOpen(
      true,
    );
  }

  function openConversation(
    caseID: string,
  ) {
    setMode(
      "conversation",
    );

    setSelectedID(
      caseID,
    );

    setReply("");

    setError("");

    clearAttachments();

    setMobileDetailOpen(
      true,
    );
  }

  async function addImageFiles(
    files: File[],
  ) {
    if (
      files.length ===
      0
    ) {
      return;
    }

    const remaining =
      MAX_ATTACHMENTS -
      attachments.length;

    if (
      remaining <= 0
    ) {
      setError(
        `You can attach up to ${MAX_ATTACHMENTS} items per message.`,
      );

      return;
    }

    setAttachmentBusy(
      true,
    );

    setError("");

    try {
      const next:
        SupportAttachment[] =
          [];

      for (
        const file of files.slice(
          0,
          remaining,
        )
      ) {
        next.push(
          await imageToAttachment(
            file,
          ),
        );
      }

      setAttachments(
        (
          current,
        ) => [
          ...current,
          ...next,
        ].slice(
          0,
          MAX_ATTACHMENTS,
        ),
      );
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
    );
  }

  function handlePaste(
    event:
      ClipboardEvent<HTMLTextAreaElement>,
  ) {
    const images =
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
      images.length ===
      0
    ) {
      return;
    }

    event.preventDefault();

    void addImageFiles(
      images,
    );
  }

  function handleDrop(
    event:
      DragEvent<HTMLDivElement>,
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
    );
  }

  function addLink() {
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
      attachments.length >=
      MAX_ATTACHMENTS
    ) {
      setError(
        `You can attach up to ${MAX_ATTACHMENTS} items per message.`,
      );

      return;
    }

    setAttachments(
      (
        current,
      ) => [
        ...current,
        {
          id:
            makeAttachmentID(),

          kind:
            "link",

          name:
            url,

          url,
        },
      ],
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

  function removeAttachment(
    attachmentID: string,
  ) {
    setAttachments(
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

  async function createConversation() {
    if (
      busy ===
      "create"
    ) {
      return;
    }

    const typed =
      initialMessage.trim();

    if (
      !typed &&
      attachments.length ===
        0
    ) {
      return;
    }

    const subjectMessage =
      typed ||
      attachmentFallback(
        attachments,
      );

    setBusy(
      "create",
    );

    setError("");

    try {
      const response =
        await fetch(
          "/api/storefront/account/support",
          {
            method:
              "POST",

            headers: {
              "Content-Type":
                "application/json",
            },

            body:
              JSON.stringify({
                type:
                  selectedTopic.value,

                subject:
                  buildConversationSubject(
                    selectedTopic,
                    subjectMessage,
                    linkedOrderID,
                  ),

                message:
                  typed,

                attachments:
                  serialiseAttachments(
                    attachments,
                  ),

                product_id:
                  "",

                variant_id:
                  "",

                order_id:
                  linkedOrderID,

                requested_quantity:
                  0,
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

      const payload =
        (await response.json()) as AccountSupportCaseResponse;

      const newID =
        payload.data.id;

      setInitialMessage(
        "",
      );

      clearAttachments();

      setSelectedID(
        newID,
      );

      setMode(
        "conversation",
      );

      await loadCases(
        newID,
        true,
      );

      await loadDetail(
        newID,
      );
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

  async function sendReply() {
    if (
      !selected ||
      busy ===
        "reply"
    ) {
      return;
    }

    const typed =
      reply.trim();

    if (
      !typed &&
      attachments.length ===
        0
    ) {
      return;
    }

    if (
      selected.status ===
      "closed"
    ) {
      setError(
        "This conversation is closed.",
      );

      return;
    }

    setBusy(
      "reply",
    );

    setError("");

    try {
      const response =
        await fetch(
          `/api/storefront/account/support/${encodeURIComponent(
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
                  typed,

                attachments:
                  serialiseAttachments(
                    attachments,
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

      setReply("");

      clearAttachments();

      await loadDetail(
        selected.id,
      );

      await loadCases(
        selected.id,
        true,
      );
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

  function renderCustomerAvatar(
    size:
      | "message"
      | "large" =
      "message",
  ) {
    const className =
      size === "large"
        ? styles.customerAvatarLarge
        : styles.customerAvatar;

    if (
      customer?.avatar_url
    ) {
      return (
        <span
          className={
            className
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
          className
        }
      >
        {initials}
      </span>
    );
  }

  function renderDraftAttachments() {
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

  function renderComposer(
    composerMode:
      ComposerMode,
  ) {
    const creating =
      composerMode ===
      "create";

    const value =
      creating
        ? initialMessage
        : reply;

    const pending =
      busy ===
      (creating
        ? "create"
        : "reply");

    const send =
      creating
        ? createConversation
        : sendReply;

    const disabled =
      pending ||
      attachmentBusy ||
      (
        !value.trim() &&
        attachments.length ===
          0
      );

    return (
      <div
        className={
          styles.composer
        }
        onDragOver={(
          event,
        ) =>
          event.preventDefault()
        }
        onDrop={
          handleDrop
        }
      >
        {renderDraftAttachments()}

        <input
          ref={
            fileInputRef
          }
          type="file"
          accept="image/*"
          multiple
          className={
            styles.hiddenInput
          }
          onChange={
            handleFileChange
          }
        />

        <div
          ref={
            attachmentPopoverRef
          }
          className={
            styles.composerShell
          }
        >
          {attachmentMenuOpen ? (
            <div
              className={
                styles.attachmentMenu
              }
            >
              <button
                type="button"
                disabled={
                  attachments.length >=
                  MAX_ATTACHMENTS
                }
                onClick={() => {
                  fileInputRef.current?.click();
                }}
              >
                <span
                  className={
                    styles.attachmentMenuIcon
                  }
                  aria-hidden="true"
                >
                  ▧
                </span>

                <span>
                  <strong>
                    Photo or screenshot
                  </strong>

                  <small>
                    Upload an image
                  </small>
                </span>
              </button>

              <button
                type="button"
                disabled={
                  attachments.length >=
                  MAX_ATTACHMENTS
                }
                onClick={() => {
                  setAttachmentMenuOpen(
                    false,
                  );

                  setLinkEditorOpen(
                    true,
                  );
                }}
              >
                <span
                  className={
                    styles.attachmentMenuIcon
                  }
                  aria-hidden="true"
                >
                  ↗
                </span>

                <span>
                  <strong>
                    Add link
                  </strong>

                  <small>
                    Attach a web address
                  </small>
                </span>
              </button>
            </div>
          ) : null}

          {linkEditorOpen ? (
            <div
              className={
                styles.linkPopover
              }
            >
              <span>
                Add link
              </span>

              <div>
                <input
                  type="url"
                  value={
                    linkInput
                  }
                  autoFocus
                  placeholder="https://example.com/..."
                  onChange={(
                    event,
                  ) =>
                    setLinkInput(
                      event
                        .target
                        .value,
                    )
                  }
                  onKeyDown={(
                    event,
                  ) => {
                    if (
                      event.key ===
                      "Enter"
                    ) {
                      event.preventDefault();

                      addLink();
                    }

                    if (
                      event.key ===
                      "Escape"
                    ) {
                      setLinkEditorOpen(
                        false,
                      );
                    }
                  }}
                />

                <button
                  type="button"
                  onClick={
                    addLink
                  }
                >
                  Add
                </button>
              </div>
            </div>
          ) : null}

          <div
            className={
              styles.composerBox
            }
          >
            <button
              type="button"
              className={
                styles.composerAttachmentButton
              }
              aria-label="Add attachment"
              aria-expanded={
                attachmentMenuOpen
              }
              disabled={
                attachmentBusy
              }
              onClick={() => {
                setLinkEditorOpen(
                  false,
                );

                setAttachmentMenuOpen(
                  (
                    current,
                  ) =>
                    !current,
                );
              }}
            >
              <PaperclipIcon />
            </button>

            <textarea
              value={
                value
              }
              rows={1}
              maxLength={
                5000
              }
              placeholder="Write a message…"
              aria-label="Write a message"
              onChange={(
                event,
              ) => {
                if (creating) {
                  setInitialMessage(
                    event
                      .target
                      .value,
                  );
                } else {
                  setReply(
                    event
                      .target
                      .value,
                  );
                }
              }}
              onPaste={
                handlePaste
              }
            />

            <button
              type="button"
              className={
                styles.sendButton
              }
              disabled={
                disabled
              }
              onClick={() => {
                void send();
              }}
            >
              {pending
                ? "Sending…"
                : "Send"}
            </button>
          </div>
        </div>

        {error ? (
          <p
            className={
              styles.composerError
            }
            role="alert"
          >
            {error}
          </p>
        ) : null}
      </div>
    );
  }

  return (
    <main
      className={
        styles.supportPage
      }
    >
      <div
        ref={
          workspaceRef
        }
        className={[
          styles.workspace,

          mobileDetailOpen
            ? styles.mobileDetailOpen
            : "",
        ]
          .filter(Boolean)
          .join(" ")}
      >
        <aside
          className={
            styles.sidebar
          }
        >
          <div
            className={
              styles.sidebarIntro
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

            <span
              className={
                styles.sidebarEyebrow
              }
            >
              Customer support
            </span>

            <h1>
              Support
            </h1>

            <p>
              Your conversations with Ene Dei.
            </p>
          </div>

          <div
            className={
              styles.sidebarConversationHeader
            }
          >
            <strong>
              Conversations
            </strong>

            <button
              type="button"
              className={
                styles.newButton
              }
              onClick={() =>
                startNewConversation()
              }
            >
              <Icon
                name="plus"
                size={15}
              />

              New
            </button>
          </div>

          <div
            className={
              styles.conversations
            }
          >
            {loading ? (
              <div
                className={
                  styles.sidebarState
                }
              >
                Loading…
              </div>
            ) : cases.length ===
              0 ? (
              <div
                className={
                  styles.sidebarEmpty
                }
              >
                <span
                  className={
                    styles.sidebarEmptyIcon
                  }
                >
                  <Icon
                    name="support"
                    size={20}
                  />
                </span>

                <strong>
                  No conversations
                </strong>

                <p>
                  Start a conversation whenever you need help.
                </p>

                <button
                  type="button"
                  onClick={() =>
                    startNewConversation()
                  }
                >
                  Start conversation
                </button>
              </div>
            ) : (
              cases.map(
                (item) => {
                  const topic =
                    getCaseTopic(
                      item,
                    );

                  const active =
                    mode ===
                      "conversation" &&
                    selectedID ===
                      item.id;

                  return (
                    <button
                      key={
                        item.id
                      }
                      type="button"
                      className={[
                        styles.conversationRow,

                        active
                          ? styles.conversationRowActive
                          : "",
                      ]
                        .filter(
                          Boolean,
                        )
                        .join(
                          " ",
                        )}
                      onClick={() =>
                        openConversation(
                          item.id,
                        )
                      }
                    >
                      <span
                        className={
                          styles.rowAvatar
                        }
                      >
                        <Icon
                          name="support"
                          size={16}
                        />
                      </span>

                      <span
                        className={
                          styles.rowContent
                        }
                      >
                        <span
                          className={
                            styles.rowTop
                          }
                        >
                          <strong>
                            {
                              topic.label
                            }
                          </strong>

                          <small>
                            {formatDate(
                              item.last_message_at ||
                                item.created_at,
                            )}
                          </small>
                        </span>

                        <span
                          className={
                            styles.rowSubtitle
                          }
                        >
                          Ene Dei Support
                        </span>
                      </span>
                    </button>
                  );
                },
              )
            )}
          </div>
        </aside>

        <section
          className={
            styles.chatPanel
          }
        >
          <button
            type="button"
            className={
              styles.mobileBack
            }
            onClick={() =>
              setMobileDetailOpen(
                false,
              )
            }
          >
            <span
              aria-hidden="true"
            >
              ←
            </span>

            Conversations
          </button>

          {mode ===
          "new" ? (
            <>
              <header
                className={
                  styles.chatHeader
                }
              >
                <span
                  className={
                    styles.supportAvatar
                  }
                >
                  <Icon
                    name="support"
                    size={18}
                  />
                </span>

                <div
                  className={
                    styles.chatHeaderText
                  }
                >
                  <strong>
                    New conversation
                  </strong>

                  <span>
                    Ene Dei Support
                  </span>
                </div>
              </header>

              <div
                className={
                  styles.newConversation
                }
              >
                <div
                  className={
                    styles.newConversationIntro
                  }
                >
                  <span
                    className={
                      styles.supportAvatarLarge
                    }
                  >
                    <Icon
                      name="support"
                      size={22}
                    />
                  </span>

                  <h2>
                    How can we help?
                  </h2>

                  <p>
                    Choose a topic and send us a message.
                  </p>
                </div>

                {linkedOrderID ? (
                  <Link
                    href={`/account/orders/${encodeURIComponent(
                      linkedOrderID,
                    )}`}
                    className={
                      styles.linkedOrder
                    }
                  >
                    <Icon
                      name="orders"
                      size={15}
                    />

                    Linked to your order
                  </Link>
                ) : null}

                <div
                  className={
                    styles.topicGrid
                  }
                >
                  {supportTopics.map(
                    (topic) => (
                      <button
                        type="button"
                        key={
                          topic.value
                        }
                        className={[
                          styles.topicButton,

                          topic.value ===
                          topicValue
                            ? styles.topicButtonActive
                            : "",
                        ]
                          .filter(
                            Boolean,
                          )
                          .join(
                            " ",
                          )}
                        onClick={() =>
                          setTopicValue(
                            topic.value,
                          )
                        }
                      >
                        <strong>
                          {
                            topic.label
                          }
                        </strong>

                        <span>
                          {
                            topic.description
                          }
                        </span>
                      </button>
                    ),
                  )}
                </div>
              </div>

              {renderComposer(
                "create",
              )}
            </>
          ) : detailLoading ? (
            <div
              className={
                styles.chatState
              }
            >
              Loading conversation…
            </div>
          ) : selected ? (
            <>
              <header
                className={
                  styles.chatHeader
                }
              >
                <span
                  className={
                    styles.supportAvatar
                  }
                >
                  <Icon
                    name="support"
                    size={18}
                  />
                </span>

                <div
                  className={
                    styles.chatHeaderText
                  }
                >
                  <strong>
                    Ene Dei Support
                  </strong>

                  <span>
                    {
                      getCaseTopic(
                        selected,
                      ).label
                    }

                    {selected.order_id
                      ? " · Order"
                      : ""}
                  </span>
                </div>

                {selected.order_id ? (
                  <Link
                    href={`/account/orders/${encodeURIComponent(
                      selected.order_id,
                    )}`}
                    className={
                      styles.headerOrderLink
                    }
                  >
                    View order
                  </Link>
                ) : null}
              </header>

              <div
                className={
                  styles.messageArea
                }
              >
                {sortedMessages.map(
                  (message) => {
                    const isCustomer =
                      message.author_type ===
                      "customer";

                    return (
                      <div
                        key={
                          message.id
                        }
                        className={[
                          styles.messageLine,

                          isCustomer
                            ? styles.messageLineCustomer
                            : styles.messageLineSupport,
                        ]
                          .filter(
                            Boolean,
                          )
                          .join(
                            " ",
                          )}
                      >
                        {!isCustomer ? (
                          <span
                            className={
                              styles.messageSupportAvatar
                            }
                          >
                            <Icon
                              name="support"
                              size={14}
                            />
                          </span>
                        ) : null}

                        <div
                          className={
                            styles.messageGroup
                          }
                        >
                          <div
                            className={[
                              styles.messageBubble,

                              isCustomer
                                ? styles.customerBubble
                                : styles.supportBubble,
                            ]
                              .filter(
                                Boolean,
                              )
                              .join(
                                " ",
                              )}
                          >
                            {message.body ? (
                              <p>
                                <MessageText
                                  text={
                                    message.body
                                  }
                                />
                              </p>
                            ) : null}

                            <MessageAttachments
                              value={
                                message.attachments
                              }
                              onPreview={(
                                attachment,
                              ) =>
                                setPreviewImage({
                                  url:
                                    attachment.url,

                                  name:
                                    attachment.name,
                                })
                              }
                            />
                          </div>

                          <span
                            className={
                              styles.messageTime
                            }
                          >
                            {isCustomer
                              ? "You · "
                              : "Ene Dei Support · "}

                            {formatMessageTime(
                              message.created_at,
                            )}
                          </span>
                        </div>

                        {isCustomer
                          ? renderCustomerAvatar()
                          : null}
                      </div>
                    );
                  },
                )}

                <div
                  ref={
                    messagesEndRef
                  }
                />
              </div>

              {canReply ? (
                renderComposer(
                  "reply",
                )
              ) : (
                <div
                  className={
                    styles.closedConversation
                  }
                >
                  <div>
                    <strong>
                      Conversation closed
                    </strong>

                    <span>
                      Start a new conversation if you need more help.
                    </span>
                  </div>

                  <button
                    type="button"
                    onClick={() =>
                      startNewConversation()
                    }
                  >
                    New conversation
                  </button>
                </div>
              )}
            </>
          ) : (
            <div
              className={
                styles.welcome
              }
            >
              <span
                className={
                  styles.supportAvatarLarge
                }
              >
                <Icon
                  name="support"
                  size={22}
                />
              </span>

              <h2>
                Ene Dei Support
              </h2>

              <p>
                Start a conversation whenever you need help.
              </p>

              <button
                type="button"
                onClick={() =>
                  startNewConversation()
                }
              >
                Start conversation
              </button>
            </div>
          )}
        </section>
      </div>

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