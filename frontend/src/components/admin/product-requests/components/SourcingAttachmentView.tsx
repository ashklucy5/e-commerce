"use client";

import type { SourcingAttachment } from "@/lib/product-requests/sourcing-attachments";
import { parseAttachments } from "@/lib/product-requests/sourcing-attachments";

import styles from "../css/SourcingNegotiation.module.css";

type Props = {
  value: unknown;
  onPreview: (attachment: SourcingAttachment) => void;
  compact?: boolean;
};

export default function SourcingAttachmentView({
  value,
  onPreview,
  compact = false,
}: Props) {
  const attachments = parseAttachments(value);

  if (attachments.length === 0) {
    return null;
  }

  return (
    <div
      className={`${styles.attachments} ${
        compact ? styles.attachmentsCompact : ""
      }`}
    >
      {attachments.map((attachment) =>
        attachment.kind === "image" ? (
          <button
            key={attachment.id}
            type="button"
            className={styles.attachmentImageButton}
            onClick={() => onPreview(attachment)}
          >
            <img
              className={styles.attachmentImage}
              src={attachment.url}
              alt={attachment.name}
              loading="lazy"
              decoding="async"
              fetchPriority="low"
              draggable={false}
            />
          </button>
        ) : (
          <a
            key={attachment.id}
            className={styles.attachmentLink}
            href={attachment.url}
            target="_blank"
            rel="noopener noreferrer"
          >
            <span className={styles.attachmentLinkIcon} aria-hidden="true">
              ↗
            </span>

            <span className={styles.attachmentLinkText}>
              {attachment.name}
            </span>
          </a>
        ),
      )}
    </div>
  );
}