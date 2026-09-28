"use client";

import Link from "next/link";
import {
  ChangeEvent,
  FormEvent,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";
import { createPortal } from "react-dom";

import type {
  AdminProductRequest,
  AdminProductRequestMessage,
  AdminSourcingConfirmation,
  AdminSourcingOffer,
  AdminSourcingOfferMutationPayload,
} from "@/lib/admin/product-request-types";
import { formatMoney } from "@/lib/money/format";

import {
  MAX_MESSAGE_ATTACHMENT_JSON_BYTES,
  MAX_SOURCING_ATTACHMENTS,
  attachmentFallback,
  attachmentPayloadBytes,
  createLinkAttachment,
  imageToSourcingAttachment,
  parseAttachments,
  serialiseAttachments,
  type SourcingAttachment,
} from "../../../../lib/product-requests/sourcing-attachments";
import { buildSourcingTimeline, offerNumberMap } from "../../../../lib/admin/product-requests/sourcing-timeline";
import SourcingAttachmentView from "./SourcingAttachmentView";
import SourcingFinalizeDialog from "./SourcingFinalizeDialog";
import SourcingOfferSheet from "./SourcingOfferSheet";
import styles from "../css/SourcingNegotiation.module.css";

type DetailState = {
  request: AdminProductRequest;
  messages: AdminProductRequestMessage[];
  offers: AdminSourcingOffer[];
  confirmation: AdminSourcingConfirmation | null;
};

type Props = {
  portal: string;
  detail: DetailState | null;
  loading: boolean;
  error: string;
  busyAction: string;
  staffName: string;
  canReview: boolean;
  canManageOffers: boolean;
  canFinalize: boolean;
  onClose: () => void;
  onRefresh: () => Promise<void>;
  onUpdateStatus: (
    target: "accepted" | "on_hold" | "cancelled",
    reason: string,
  ) => Promise<void>;
  onSendMessage: (
    body: string,
    visibility: "customer" | "internal",
    attachments: unknown[],
  ) => Promise<void>;
  onSaveOffer: (
    offerId: string | null,
    payload: AdminSourcingOfferMutationPayload,
    sendAfterSave: boolean,
  ) => Promise<void>;
  onSendOffer: (offerId: string) => Promise<void>;
  onFinalizeOffer: (offerId: string) => Promise<void>;
};

type ReasonAction = "on_hold" | "cancelled" | null;

function statusLabel(status: string): string {
  const labels: Record<string, string> = {
    pending_review: "Under review",
    on_hold: "On hold",
    accepted: "Accepted for sourcing",
    negotiating: "Negotiating",
    agreed: "Agreement confirmed",
    converted_to_order: "Order created",
    cancelled: "Cancelled",
    draft: "Draft",
    sent: "Awaiting customer",
    customer_accepted: "Customer accepted",
    customer_rejected: "Customer declined",
    superseded: "Superseded",
    expired: "Expired",
    finalized: "Finalized",
  };

  return (
    labels[status] ??
    status.replace(/_/g, " ").replace(/\b\w/g, (letter) => letter.toUpperCase())
  );
}

function statusClass(status: string): string {
  if (["accepted", "agreed", "converted_to_order", "customer_accepted", "finalized"].includes(status)) {
    return styles.statusGood;
  }
  if (["negotiating", "sent"].includes(status)) return styles.statusInfo;
  if (["on_hold", "superseded", "expired"].includes(status)) return styles.statusWarn;
  if (["cancelled", "customer_rejected"].includes(status)) return styles.statusDanger;
  return styles.statusNeutral;
}

function formatDateTime(value?: string): string {
  if (!value) return "—";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "—";

  return new Intl.DateTimeFormat("en", {
    month: "short",
    day: "numeric",
    hour: "numeric",
    minute: "2-digit",
  }).format(date);
}

function formatFullDateTime(value?: string): string {
  if (!value) return "—";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "—";

  return new Intl.DateTimeFormat("en", {
    month: "short",
    day: "numeric",
    year: "numeric",
    hour: "numeric",
    minute: "2-digit",
  }).format(date);
}

function objectEntries(value: unknown): Array<[string, string]> {
  if (!value || typeof value !== "object" || Array.isArray(value)) return [];
  return Object.entries(value as Record<string, unknown>)
    .filter(([, item]) => item != null && String(item).trim())
    .map(([key, item]) => [key, String(item)]);
}

function offerTotal(offer: AdminSourcingOffer): number {
  return offer.unit_price * (offer.quoted_quantity ?? 0) + offer.shipping_price;
}

function initials(name: string): string {
  const parts = name.trim().split(/\s+/).filter(Boolean);
  if (parts.length === 0) return "?";
  return parts.slice(0, 2).map((part) => part[0]?.toUpperCase() ?? "").join("");
}

function RichMessageText({ value }: { value: string }) {
  const parts = value.split(/(https?:\/\/[^\s]+)/gi);
  return (
    <p className={styles.messageText}>
      {parts.map((part, index) =>
        /^https?:\/\//i.test(part) ? (
          <a key={`${part}-${index}`} href={part} target="_blank" rel="noopener noreferrer">
            {part}
          </a>
        ) : (
          <span key={`${part}-${index}`}>{part}</span>
        ),
      )}
    </p>
  );
}

function RequestOverview({
  request,
  onPreview,
}: {
  request: AdminProductRequest;
  onPreview: (attachment: SourcingAttachment) => void;
}) {
  const requirements = objectEntries(request.customer_requirements);

  return (
    <div className={styles.requestOverview}>
      <div className={styles.requestOverviewHeader}>
        <div>
          <span className={styles.overline}>Initial product request</span>
          <strong>{request.requested_product_name}</strong>
        </div>
        <span className={styles.requestQuantity}>Qty {request.requested_quantity}</span>
      </div>
      <p>{request.description}</p>

      {requirements.length > 0 ? (
        <div className={styles.requirementGrid}>
          {requirements.map(([key, value]) => (
            <span key={key}>
              <small>{key}</small>
              <strong>{value}</strong>
            </span>
          ))}
        </div>
      ) : null}

      <SourcingAttachmentView value={request.attachments} onPreview={onPreview} />

      {request.external_url ? (
        <a className={styles.referenceLink} href={request.external_url} target="_blank" rel="noopener noreferrer">
          <span aria-hidden="true">↗</span>
          <span>Customer reference</span>
        </a>
      ) : null}

      <time>{formatFullDateTime(request.created_at)}</time>
    </div>
  );
}

function OfferCard({
  offer,
  offerNumber,
  canManageOffers,
  busy,
  onEdit,
  onSend,
  onPreview,
}: {
  offer: AdminSourcingOffer;
  offerNumber: number;
  canManageOffers: boolean;
  busy: boolean;
  onEdit: (offer: AdminSourcingOffer) => void;
  onSend: (offerId: string) => Promise<void>;
  onPreview: (attachment: SourcingAttachment) => void;
}) {
  const specs = objectEntries(offer.offered_specifications);

  return (
    <article className={`${styles.offerCard} ${offer.status === "draft" ? styles.offerCardDraft : ""}`}>
      <div className={styles.offerCardTopline}>
        <div>
          <span className={styles.overline}>
            {offer.status === "draft" ? "Admin-only draft" : `Commercial offer · Offer ${offerNumber}`}
          </span>
          <h3>{offer.product_name}</h3>
        </div>
        <span className={`${styles.statusPill} ${statusClass(offer.status)}`}>{statusLabel(offer.status)}</span>
      </div>

      <p className={styles.offerDescription}>{offer.description}</p>

      {specs.length > 0 ? (
        <div className={styles.offerSpecs}>
          {specs.map(([key, value]) => (
            <span key={key}>
              <small>{key}</small>
              <strong>{value}</strong>
            </span>
          ))}
        </div>
      ) : null}

      <div className={styles.offerMoneyGrid}>
        <span><small>Quantity</small><strong>{offer.quoted_quantity ?? "—"}</strong></span>
        <span><small>MOQ</small><strong>{offer.minimum_order_quantity ?? "—"}</strong></span>
        <span><small>Unit price</small><strong>{formatMoney(offer.unit_price, offer.currency)}</strong></span>
        <span><small>Shipping</small><strong>{formatMoney(offer.shipping_price, offer.currency)}</strong></span>
      </div>

      <div className={styles.offerTotal}>
        <span>Total</span>
        <strong>{formatMoney(offerTotal(offer), offer.currency)}</strong>
      </div>

      <SourcingAttachmentView value={offer.attachments} onPreview={onPreview} compact />

      <div className={styles.offerMetaRow}>
        <span>{offer.expires_at ? `Expires ${formatFullDateTime(offer.expires_at)}` : "No expiry"}</span>
        {offer.sent_at ? <span>Sent {formatFullDateTime(offer.sent_at)}</span> : null}
      </div>

      {canManageOffers && offer.status === "draft" ? (
        <div className={styles.offerActions}>
          <button type="button" className={styles.secondaryButton} onClick={() => onEdit(offer)} disabled={busy}>Edit draft</button>
          <button type="button" className={styles.primaryButton} onClick={() => void onSend(offer.id).catch(() => undefined)} disabled={busy}>
            {busy ? "Working…" : "Send to customer"}
          </button>
        </div>
      ) : null}
    </article>
  );
}

function AgreementCard({
  confirmation,
  portal,
}: {
  confirmation: AdminSourcingConfirmation;
  portal: string;
}) {
  const specs = objectEntries(confirmation.accepted_specifications);

  return (
    <article className={styles.agreementCard}>
      <div className={styles.agreementIcon} aria-hidden="true">✓</div>
      <div className={styles.agreementBody}>
        <span className={styles.overline}>Sourcing agreement confirmed</span>
        <h3>{confirmation.accepted_product_name}</h3>
        <p>The customer-approved commercial terms are now locked in the sourcing confirmation.</p>

        {specs.length > 0 ? (
          <div className={styles.offerSpecs}>
            {specs.map(([key, value]) => (
              <span key={key}><small>{key}</small><strong>{value}</strong></span>
            ))}
          </div>
        ) : null}

        <div className={styles.agreementGrid}>
          <span><small>Quantity</small><strong>{confirmation.quantity}</strong></span>
          <span><small>MOQ</small><strong>{confirmation.minimum_order_quantity}</strong></span>
          <span><small>Unit price</small><strong>{formatMoney(confirmation.unit_price_snapshot, confirmation.currency)}</strong></span>
          <span><small>Shipping</small><strong>{formatMoney(confirmation.shipping_price_snapshot, confirmation.currency)}</strong></span>
        </div>
        <div className={styles.agreementTotal}>
          <span>Total</span>
          <strong>{formatMoney(confirmation.total_amount, confirmation.currency)}</strong>
        </div>

        {confirmation.created_order_id ? (
          <Link className={styles.orderLink} href={`/${portal}/orders`}>
            View sourcing order <span aria-hidden="true">→</span>
          </Link>
        ) : (
          <span className={styles.agreementHint}>Waiting for the customer to continue to sourcing checkout.</span>
        )}
      </div>
    </article>
  );
}

export default function AdminProductRequestNegotiation({
  portal,
  detail,
  loading,
  error,
  busyAction,
  staffName,
  canReview,
  canManageOffers,
  canFinalize,
  onClose,
  onRefresh,
  onUpdateStatus,
  onSendMessage,
  onSaveOffer,
  onSendOffer,
  onFinalizeOffer,
}: Props) {
  const [mounted, setMounted] = useState(false);
  const [detailsOpen, setDetailsOpen] = useState(false);
  const [reasonAction, setReasonAction] = useState<ReasonAction>(null);
  const [reason, setReason] = useState("");
  const [reasonError, setReasonError] = useState("");
  const [messageBody, setMessageBody] = useState("");
  const [messageVisibility, setMessageVisibility] = useState<"customer" | "internal">("customer");
  const [messageAttachments, setMessageAttachments] = useState<SourcingAttachment[]>([]);
  const [composerMenuOpen, setComposerMenuOpen] = useState(false);
  const [linkMode, setLinkMode] = useState(false);
  const [linkDraft, setLinkDraft] = useState("");
  const [attachmentBusy, setAttachmentBusy] = useState(false);
  const [localError, setLocalError] = useState("");
  const [offerSheetOpen, setOfferSheetOpen] = useState(false);
  const [editingOffer, setEditingOffer] = useState<AdminSourcingOffer | null>(null);
  const [finalizeOffer, setFinalizeOffer] = useState<AdminSourcingOffer | null>(null);
  const [preview, setPreview] = useState<SourcingAttachment | null>(null);

  const fileInputRef = useRef<HTMLInputElement | null>(null);
  const timelineScrollRef = useRef<HTMLDivElement | null>(null);
  const initialScrollDone = useRef(false);

  useEffect(() => {
    setMounted(true);
    const previousOverflow = document.body.style.overflow;
    document.body.style.overflow = "hidden";
    return () => {
      document.body.style.overflow = previousOverflow;
    };
  }, []);

  const request = detail?.request;
  const timeline = useMemo(
    () =>
      detail
        ? buildSourcingTimeline(detail.request, detail.messages, detail.offers, detail.confirmation)
        : [],
    [detail],
  );
  const offerNumbers = useMemo(() => offerNumberMap(detail?.offers ?? []), [detail?.offers]);
  const draftOffer = useMemo(
    () => detail?.offers.find((offer) => offer.status === "draft") ?? null,
    [detail?.offers],
  );
  const acceptedOffer = useMemo(
    () => detail?.offers.find((offer) => offer.status === "customer_accepted") ?? null,
    [detail?.offers],
  );
  const latestCustomerOffer = useMemo(
    () => detail?.offers.find((offer) => offer.status !== "draft") ?? null,
    [detail?.offers],
  );

  const canMessage = Boolean(
    request &&
      canReview &&
      request.crm_status !== "closed" &&
      !["cancelled", "converted_to_order"].includes(request.status),
  );
  const canCreateOffer = Boolean(
    request &&
      canManageOffers &&
      request.crm_status !== "closed" &&
      ["accepted", "negotiating"].includes(request.status) &&
      !draftOffer &&
      !acceptedOffer &&
      !detail?.confirmation,
  );

  useEffect(() => {
    const container = timelineScrollRef.current;
    if (!container || timeline.length === 0) return;

    const distanceFromBottom = container.scrollHeight - container.scrollTop - container.clientHeight;
    if (!initialScrollDone.current || distanceFromBottom < 220) {
      requestAnimationFrame(() => {
        container.scrollTop = container.scrollHeight;
        initialScrollDone.current = true;
      });
    }
  }, [timeline.length]);

  useEffect(() => {
    initialScrollDone.current = false;
    setMessageBody("");
    setMessageAttachments([]);
    setMessageVisibility("customer");
    setComposerMenuOpen(false);
    setLinkMode(false);
    setDetailsOpen(false);
    setOfferSheetOpen(false);
    setEditingOffer(null);
    setFinalizeOffer(null);
  }, [request?.id]);

  if (!mounted) return null;

  async function handleReasonSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!reasonAction || !reason.trim()) return;
    setReasonError("");
    try {
      await onUpdateStatus(reasonAction, reason.trim());
      setReasonAction(null);
      setReason("");
    } catch (value: unknown) {
      setReasonError(value instanceof Error ? value.message : "Unable to update this request.");
    }
  }

  async function handleMessageSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!canMessage || attachmentBusy) return;

    const body = messageBody.trim() || (messageAttachments.length ? attachmentFallback(messageAttachments) : "");
    if (!body) return;

    setLocalError("");
    if (attachmentPayloadBytes(messageAttachments) > MAX_MESSAGE_ATTACHMENT_JSON_BYTES) {
      setLocalError("Message attachments are too large. Remove an image or link and try again.");
      return;
    }

    try {
      await onSendMessage(body, messageVisibility, serialiseAttachments(messageAttachments));
      setMessageBody("");
      setMessageAttachments([]);
      setLinkMode(false);
      setLinkDraft("");
    } catch (value: unknown) {
      setLocalError(value instanceof Error ? value.message : "Unable to send this message.");
    }
  }

  async function handleMessageImage(event: ChangeEvent<HTMLInputElement>) {
    const file = event.target.files?.[0];
    event.target.value = "";
    if (!file) return;

    setLocalError("");
    setAttachmentBusy(true);
    try {
      if (messageAttachments.length >= MAX_SOURCING_ATTACHMENTS) {
        throw new Error(`You can attach up to ${MAX_SOURCING_ATTACHMENTS} references.`);
      }
      const attachment = await imageToSourcingAttachment(file, "message");
      const next = [...messageAttachments, attachment];
      if (attachmentPayloadBytes(next) > MAX_MESSAGE_ATTACHMENT_JSON_BYTES) {
        throw new Error("Message attachments are too large. Remove an image and try again.");
      }
      setMessageAttachments(next);
      setComposerMenuOpen(false);
    } catch (value: unknown) {
      setLocalError(value instanceof Error ? value.message : "Unable to prepare that image.");
    } finally {
      setAttachmentBusy(false);
    }
  }

  function addMessageLink() {
    setLocalError("");
    try {
      if (messageAttachments.length >= MAX_SOURCING_ATTACHMENTS) {
        throw new Error(`You can attach up to ${MAX_SOURCING_ATTACHMENTS} references.`);
      }
      const next = [...messageAttachments, createLinkAttachment(linkDraft)];
      if (attachmentPayloadBytes(next) > MAX_MESSAGE_ATTACHMENT_JSON_BYTES) {
        throw new Error("Message attachments are too large. Remove a reference and try again.");
      }
      setMessageAttachments(next);
      setLinkDraft("");
      setLinkMode(false);
      setComposerMenuOpen(false);
    } catch (value: unknown) {
      setLocalError(value instanceof Error ? value.message : "Unable to add that link.");
    }
  }

  function openNewOffer() {
    setEditingOffer(null);
    setOfferSheetOpen(true);
    setComposerMenuOpen(false);
  }

  function openDraft(offer: AdminSourcingOffer) {
    setEditingOffer(offer);
    setOfferSheetOpen(true);
  }

  async function handleSaveOffer(
    offerId: string | null,
    payload: AdminSourcingOfferMutationPayload,
    sendAfterSave: boolean,
  ) {
    await onSaveOffer(offerId, payload, sendAfterSave);
    setOfferSheetOpen(false);
    setEditingOffer(null);
  }

  async function handleFinalize(offerId: string) {
    await onFinalizeOffer(offerId);
    setFinalizeOffer(null);
  }

  const renderRequestContext = request ? (
    <div className={styles.contextContent}>
      <section className={styles.contextSection}>
        <span className={styles.overline}>Request</span>
        <h3>{request.requested_product_name}</h3>
        <p>{request.description}</p>
        <div className={styles.contextFacts}>
          <span><small>Quantity</small><strong>{request.requested_quantity}</strong></span>
          <span><small>Request</small><strong>{request.request_number}</strong></span>
          <span><small>Case</small><strong>{request.case_number}</strong></span>
          <span><small>Created</small><strong>{formatDateTime(request.created_at)}</strong></span>
        </div>
      </section>

      <section className={styles.contextSection}>
        <span className={styles.overline}>Customer</span>
        <div className={styles.customerIdentity}>
          <div className={styles.avatar}>{initials(request.customer.full_name)}</div>
          <div>
            <strong>{request.customer.full_name}</strong>
            <span>{request.customer.email || "No email on request"}</span>
            <span>{request.customer.phone}</span>
          </div>
        </div>
      </section>

      <section className={styles.contextSection}>
        <div className={styles.contextSectionHeader}>
          <span className={styles.overline}>Workflow</span>
          <span className={`${styles.statusPill} ${statusClass(request.status)}`}>{statusLabel(request.status)}</span>
        </div>

        {request.status_reason ? (
          <div className={styles.statusReason}>
            <small>Status reason</small>
            <p>{request.status_reason}</p>
          </div>
        ) : null}

        {latestCustomerOffer ? (
          <div className={styles.currentOfferMini}>
            <span>Current commercial state</span>
            <strong>Offer {offerNumbers.get(latestCustomerOffer.id) ?? 1} · {statusLabel(latestCustomerOffer.status)}</strong>
            <small>{formatMoney(offerTotal(latestCustomerOffer), latestCustomerOffer.currency)}</small>
          </div>
        ) : null}

        {canReview && ["pending_review", "on_hold"].includes(request.status) ? (
          <div className={styles.reviewActions}>
            <button
              type="button"
              className={styles.primaryButton}
              disabled={Boolean(busyAction)}
              onClick={() => void onUpdateStatus("accepted", "").catch(() => undefined)}
            >
              Accept for sourcing
            </button>
            <div>
              <button type="button" className={styles.secondaryButton} disabled={Boolean(busyAction)} onClick={() => { setReasonError(""); setReasonAction("on_hold"); setReason(request.status === "on_hold" ? request.status_reason ?? "" : ""); }}>
                {request.status === "on_hold" ? "Update hold reason" : "Put on hold"}
              </button>
              <button type="button" className={styles.dangerButton} disabled={Boolean(busyAction)} onClick={() => { setReasonError(""); setReasonAction("cancelled"); setReason(""); }}>
                Cancel request
              </button>
            </div>
          </div>
        ) : null}
      </section>

      <section className={styles.contextSection}>
        <span className={styles.overline}>References</span>
        <SourcingAttachmentView value={request.attachments} onPreview={setPreview} compact />
        {request.external_url ? (
          <a className={styles.referenceLink} href={request.external_url} target="_blank" rel="noopener noreferrer">
            <span aria-hidden="true">↗</span><span>Open customer reference</span>
          </a>
        ) : null}
        {!request.external_url && parseAttachments(request.attachments).length === 0 ? (
          <span className={styles.emptyContext}>No external references attached.</span>
        ) : null}
      </section>
    </div>
  ) : null;

  return createPortal(
    <div className={styles.workspaceBackdrop}>
      <section className={styles.workspace} aria-label="Product request negotiation workspace">
        <header className={styles.workspaceHeader}>
          <button type="button" className={styles.backButton} onClick={onClose} aria-label="Back to product requests">
            <span aria-hidden="true">←</span>
            <span className={styles.backLabel}>Product requests</span>
          </button>

          <div className={styles.workspaceIdentity}>
            <span className={styles.workspaceRequestNumber}>{request?.request_number ?? "Loading request"}</span>
            <strong>{request?.requested_product_name ?? "Opening sourcing workspace…"}</strong>
            {request ? <span>{request.customer.full_name} · Qty {request.requested_quantity} · {statusLabel(request.status)}</span> : null}
          </div>

          <div className={styles.workspaceHeaderActions}>
            {request ? <span className={`${styles.statusPill} ${statusClass(request.status)}`}>{statusLabel(request.status)}</span> : null}
            <button type="button" className={`${styles.secondaryButton} ${styles.mobileOnly}`} onClick={() => setDetailsOpen(true)} disabled={!request}>Details</button>
            <button type="button" className={styles.iconButton} onClick={() => void onRefresh()} aria-label="Refresh request" disabled={loading || Boolean(busyAction)}>↻</button>
            <button type="button" className={styles.iconButton} onClick={onClose} aria-label="Close request">×</button>
          </div>
        </header>

        {error ? <div className={styles.workspaceError}>{error}</div> : null}

        {loading && !detail ? (
          <div className={styles.workspaceLoading}>Loading sourcing conversation…</div>
        ) : detail && request ? (
          <div className={styles.workspaceGrid}>
            <aside className={styles.contextRail}>{renderRequestContext}</aside>

            <section className={styles.conversation}>
              <div className={styles.conversationTopbar}>
                <div>
                  <span className={styles.overline}>Negotiation</span>
                  <strong>Customer conversation</strong>
                </div>
                <div className={styles.conversationMeta}>
                  <span>{detail.messages.length} messages</span>
                  <span>{detail.offers.filter((offer) => offer.status !== "draft").length} offers</span>
                </div>
              </div>

              <div className={styles.timeline} ref={timelineScrollRef}>
                <div className={styles.timelineInner}>
                  {request.status === "on_hold" && request.status_reason ? (
                    <div className={`${styles.stateBanner} ${styles.stateBannerWarn}`}>
                      <strong>Request is on hold</strong>
                      <span>{request.status_reason}</span>
                    </div>
                  ) : null}
                  {request.status === "cancelled" ? (
                    <div className={`${styles.stateBanner} ${styles.stateBannerDanger}`}>
                      <strong>Request cancelled</strong>
                      <span>{request.status_reason || "This sourcing request has been closed."}</span>
                    </div>
                  ) : null}

                  {canReview && ["pending_review", "on_hold"].includes(request.status) ? (
                    <div className={`${styles.mobileReviewCard} ${styles.mobileOnly}`}>
                      <div>
                        <span className={styles.overline}>{request.status === "on_hold" ? "Sourcing paused" : "Request awaiting review"}</span>
                        <strong>{request.status === "on_hold" ? request.status_reason || "Review the hold status." : "Choose how the sourcing team should proceed."}</strong>
                      </div>
                      <button type="button" className={styles.primaryButton} disabled={Boolean(busyAction)} onClick={() => void onUpdateStatus("accepted", "").catch(() => undefined)}>Accept for sourcing</button>
                      <div>
                        <button type="button" className={styles.secondaryButton} disabled={Boolean(busyAction)} onClick={() => { setReasonError(""); setReasonAction("on_hold"); setReason(request.status === "on_hold" ? request.status_reason ?? "" : ""); }}>{request.status === "on_hold" ? "Update hold" : "Hold"}</button>
                        <button type="button" className={styles.dangerButton} disabled={Boolean(busyAction)} onClick={() => { setReasonError(""); setReasonAction("cancelled"); setReason(""); }}>Cancel</button>
                      </div>
                    </div>
                  ) : null}

                  {timeline.map((item) => {
                    if (item.kind === "request") {
                      return (
                        <div className={styles.timelineRequest} key={item.id}>
                          <div className={styles.messageIdentity}>
                            <div className={styles.avatar}>{initials(item.request.customer.full_name)}</div>
                            <span>{item.request.customer.full_name}</span>
                          </div>
                          <RequestOverview request={item.request} onPreview={setPreview} />
                        </div>
                      );
                    }

                    if (item.kind === "message") {
                      const isCustomer = item.message.author_type === "customer";
                      const isInternal = item.message.visibility === "internal";
                      const displayName = isCustomer
                        ? request.customer.full_name
                        : item.message.support_actor?.display_name ?? staffName;

                      return (
                        <article
                          key={item.id}
                          className={`${styles.messageRow} ${isCustomer ? styles.messageRowCustomer : styles.messageRowStaff} ${isInternal ? styles.messageRowInternal : ""}`}
                        >
                          <div className={styles.messageIdentity}>
                            <div className={styles.avatar}>{initials(displayName)}</div>
                            <span>{displayName}</span>
                            {isInternal ? <b>Internal note</b> : null}
                          </div>
                          <div className={styles.messageBubble}>
                            <RichMessageText value={item.message.body} />
                            <SourcingAttachmentView value={item.message.attachments} onPreview={setPreview} />
                          </div>
                          <time>{formatDateTime(item.message.created_at)}</time>
                        </article>
                      );
                    }

                    if (item.kind === "offer_sent") {
                      return (
                        <div className={styles.timelineCommercial} key={item.id}>
                          <span className={styles.commercialIntro}>{item.offer.created_by?.display_name ?? "Sourcing team"} sent Offer {item.offerNumber}</span>
                          <OfferCard
                            offer={item.offer}
                            offerNumber={item.offerNumber}
                            canManageOffers={false}
                            busy={false}
                            onEdit={() => undefined}
                            onSend={async () => undefined}
                            onPreview={setPreview}
                          />
                          <time>{formatDateTime(item.at)}</time>
                        </div>
                      );
                    }

                    if (item.kind === "offer_response") {
                      return (
                        <div className={`${styles.timelineEvent} ${item.response === "accepted" ? styles.timelineEventGood : styles.timelineEventDanger}`} key={item.id}>
                          <span className={styles.timelineEventIcon}>{item.response === "accepted" ? "✓" : "×"}</span>
                          <div>
                            <strong>Customer {item.response === "accepted" ? "accepted" : "declined"} Offer {item.offerNumber}</strong>
                            <span>{item.response === "accepted" ? "Commercial terms are locked for final review." : "Negotiation can continue with a revised offer."}</span>
                          </div>
                          <time>{formatDateTime(item.at)}</time>
                        </div>
                      );
                    }

                    if (item.kind === "agreement") {
                      return (
                        <div className={styles.timelineCommercial} key={item.id}>
                          <AgreementCard confirmation={item.confirmation} portal={portal} />
                          <time>{formatDateTime(item.at)}</time>
                        </div>
                      );
                    }

                    return (
                      <div className={`${styles.timelineEvent} ${styles.timelineEventGood}`} key={item.id}>
                        <span className={styles.timelineEventIcon}>✓</span>
                        <div>
                          <strong>Sourcing order created</strong>
                          <span>{item.confirmation.created_order_id}</span>
                        </div>
                        <Link className={styles.eventLink} href={`/${portal}/orders`}>View order →</Link>
                      </div>
                    );
                  })}

                  {draftOffer ? (
                    <div className={styles.draftZone}>
                      <span className={styles.draftZoneLabel}>Admin-only commercial draft</span>
                      <OfferCard
                        offer={draftOffer}
                        offerNumber={(offerNumbers.get(draftOffer.id) ?? detail.offers.length) + 1}
                        canManageOffers={canManageOffers && request.crm_status !== "closed" && ["accepted", "negotiating"].includes(request.status) && !acceptedOffer}
                        busy={busyAction === `offer-send:${draftOffer.id}`}
                        onEdit={openDraft}
                        onSend={onSendOffer}
                        onPreview={setPreview}
                      />
                    </div>
                  ) : null}

                  {acceptedOffer && canFinalize && request.crm_status !== "closed" && !detail.confirmation ? (
                    <div className={styles.finalizePrompt}>
                      <div>
                        <span className={styles.overline}>Customer accepted the commercial terms</span>
                        <strong>Review the accepted offer before creating the immutable sourcing confirmation.</strong>
                      </div>
                      <button type="button" className={styles.primaryButton} onClick={() => setFinalizeOffer(acceptedOffer)} disabled={Boolean(busyAction)}>
                        Review final agreement
                      </button>
                    </div>
                  ) : null}
                </div>
              </div>

              <div className={styles.composerDock}>
                {localError ? <div className={styles.composerError}>{localError}</div> : null}

                {canMessage ? (
                  <form className={`${styles.composer} ${messageVisibility === "internal" ? styles.composerInternal : ""}`} onSubmit={(event) => void handleMessageSubmit(event)}>
                    {messageVisibility === "internal" ? (
                      <div className={styles.internalModeBanner}>
                        <span>Internal note</span>
                        <button type="button" onClick={() => setMessageVisibility("customer")}>Return to customer reply</button>
                      </div>
                    ) : null}

                    {messageAttachments.length > 0 ? (
                      <div className={styles.composerAttachments}>
                        {messageAttachments.map((attachment) => (
                          <div className={styles.composerAttachment} key={attachment.id}>
                            {attachment.kind === "image" ? (
                              <button type="button" onClick={() => setPreview(attachment)} className={styles.composerAttachmentPreview}>
                                <img src={attachment.url} alt={attachment.name} />
                              </button>
                            ) : (
                              <span className={styles.composerLinkPreview}>↗ {attachment.name}</span>
                            )}
                            <button type="button" className={styles.removeAttachment} onClick={() => setMessageAttachments((current) => current.filter((item) => item.id !== attachment.id))} aria-label={`Remove ${attachment.name}`}>×</button>
                          </div>
                        ))}
                      </div>
                    ) : null}

                    {linkMode ? (
                      <div className={styles.composerLinkAdder}>
                        <input value={linkDraft} onChange={(event) => setLinkDraft(event.target.value)} placeholder="Paste a reference link" autoFocus />
                        <button type="button" className={styles.ghostButton} onClick={addMessageLink} disabled={!linkDraft.trim()}>Add</button>
                        <button type="button" className={styles.iconButtonSmall} onClick={() => { setLinkMode(false); setLinkDraft(""); }} aria-label="Cancel link">×</button>
                      </div>
                    ) : null}

                    <div className={styles.composerMain}>
                      <div className={styles.composerPlusWrap}>
                        <button type="button" className={styles.plusButton} onClick={() => setComposerMenuOpen((current) => !current)} aria-label="Add to message" aria-expanded={composerMenuOpen}>+</button>
                        {composerMenuOpen ? (
                          <div className={styles.composerMenu}>
                            <button type="button" onClick={() => fileInputRef.current?.click()} disabled={attachmentBusy || messageAttachments.length >= MAX_SOURCING_ATTACHMENTS}><span>▧</span> Add image</button>
                            <button type="button" onClick={() => { setLinkMode(true); setComposerMenuOpen(false); }} disabled={messageAttachments.length >= MAX_SOURCING_ATTACHMENTS}><span>↗</span> Add link</button>
                            <button type="button" onClick={() => { setMessageVisibility(messageVisibility === "internal" ? "customer" : "internal"); setComposerMenuOpen(false); }}><span>◌</span> {messageVisibility === "internal" ? "Customer reply" : "Internal note"}</button>
                            {canCreateOffer ? <button type="button" onClick={openNewOffer}><span>¤</span> Create offer</button> : null}
                          </div>
                        ) : null}
                      </div>

                      <input ref={fileInputRef} type="file" accept="image/*" hidden onChange={(event) => void handleMessageImage(event)} />
                      <textarea
                        value={messageBody}
                        onChange={(event) => setMessageBody(event.target.value)}
                        placeholder={messageVisibility === "internal" ? "Add an internal sourcing note…" : "Type a message…"}
                        rows={1}
                        maxLength={5000}
                      />
                      <button type="submit" className={styles.sendButton} disabled={(!messageBody.trim() && messageAttachments.length === 0) || busyAction === "message" || attachmentBusy}>
                        {busyAction === "message" ? "…" : "Send"}
                      </button>
                    </div>

                    <div className={styles.composerFooter}>
                      <span>{messageVisibility === "internal" ? "Only staff can see this note." : "Visible to the customer."}</span>
                      {canCreateOffer ? <button type="button" onClick={openNewOffer}>Create offer</button> : null}
                    </div>
                  </form>
                ) : (
                  <div className={styles.readOnlyComposer}>
                    <strong>{request.status === "converted_to_order" ? "Conversation closed after order creation" : request.status === "cancelled" ? "This request is cancelled" : "Conversation is read-only"}</strong>
                    <span>The sourcing history remains available above.</span>
                  </div>
                )}
              </div>
            </section>
          </div>
        ) : null}
      </section>

      {detailsOpen && request ? (
        <div className={styles.sheetBackdrop} onMouseDown={() => setDetailsOpen(false)}>
          <aside className={`${styles.mobileDetailsSheet} ${styles.mobileOnly}`} onMouseDown={(event) => event.stopPropagation()}>
            <div className={styles.sheetHandle} aria-hidden="true" />
            <header className={styles.mobileSheetHeader}>
              <div><span className={styles.overline}>Product request</span><h2>Details</h2></div>
              <button type="button" className={styles.iconButton} onClick={() => setDetailsOpen(false)} aria-label="Close details">×</button>
            </header>
            {renderRequestContext}
          </aside>
        </div>
      ) : null}

      {reasonAction ? (
        <div className={styles.modalBackdrop} onMouseDown={() => setReasonAction(null)}>
          <form className={`${styles.modalCard} ${styles.reasonDialog}`} onSubmit={(event) => void handleReasonSubmit(event)} onMouseDown={(event) => event.stopPropagation()}>
            <div className={styles.sheetHandle} aria-hidden="true" />
            <header className={styles.modalHeader}>
              <div>
                <span className={styles.overline}>{reasonAction === "on_hold" ? "Workflow" : "Close sourcing request"}</span>
                <h2>{reasonAction === "on_hold" ? (request?.status === "on_hold" ? "Update hold reason" : "Put request on hold") : "Cancel request"}</h2>
                <p>{reasonAction === "on_hold" ? "The customer can see the hold reason, so keep it clear and useful." : "Cancellation ends this sourcing request. Add the reason that should be shown with the status."}</p>
              </div>
              <button type="button" className={styles.iconButton} onClick={() => setReasonAction(null)} aria-label="Close workflow dialog">×</button>
            </header>
            {reasonError ? <div className={styles.inlineError}>{reasonError}</div> : null}
            <label className={styles.reasonField}>
              <span>Reason</span>
              <textarea value={reason} onChange={(event) => setReason(event.target.value)} rows={4} autoFocus placeholder={reasonAction === "on_hold" ? "Why is this request on hold?" : "Why is this request being cancelled?"} />
            </label>
            <footer className={styles.modalFooter}>
              <button type="button" className={styles.secondaryButton} onClick={() => setReasonAction(null)} disabled={Boolean(busyAction)}>Back</button>
              <button type="submit" className={reasonAction === "cancelled" ? styles.dangerButton : styles.primaryButton} disabled={!reason.trim() || Boolean(busyAction)}>
                {reasonAction === "on_hold" ? "Save hold status" : "Cancel request"}
              </button>
            </footer>
          </form>
        </div>
      ) : null}

      {request && offerSheetOpen ? (
        <SourcingOfferSheet
          open
          request={request}
          offer={editingOffer}
          busy={busyAction === "offer-save"}
          onClose={() => { setOfferSheetOpen(false); setEditingOffer(null); }}
          onPreview={setPreview}
          onSave={handleSaveOffer}
        />
      ) : null}

      <SourcingFinalizeDialog
        offer={finalizeOffer}
        busy={Boolean(finalizeOffer && busyAction === `offer-finalize:${finalizeOffer.id}`)}
        onClose={() => setFinalizeOffer(null)}
        onFinalize={handleFinalize}
      />

      {preview ? (
        <div className={styles.lightboxBackdrop} onMouseDown={() => setPreview(null)}>
          <figure className={styles.lightboxFigure} onMouseDown={(event) => event.stopPropagation()}>
            <button type="button" className={styles.lightboxClose} onClick={() => setPreview(null)} aria-label="Close image preview">×</button>
            <img src={preview.url} alt={preview.name} />
            <figcaption>{preview.name}</figcaption>
          </figure>
        </div>
      ) : null}
    </div>,
    document.body,
  );
}
