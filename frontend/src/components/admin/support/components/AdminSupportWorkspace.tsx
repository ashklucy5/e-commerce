"use client";

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

import { useAdminSession } from "@/components/admin/layout/context/AdminSessionContext";
import { AdminRequestError, adminFetch } from "@/lib/admin/api";
import type {
  AdminCRMCase,
  AdminCRMCasePriority,
  AdminCRMCaseResponse,
  AdminCRMCasesResponse,
  AdminCRMCaseStatus,
  AdminCRMAttachment,
  AdminCRMEscalationResponse,
  AdminCRMMessage,
  AdminCRMMessageResponse,
  AdminCRMMessagesResponse,
  AdminCRMQueue,
  AdminCRMQueuesResponse,
} from "@/lib/admin/support-types";
import {
  MAX_ADMIN_SUPPORT_ATTACHMENTS,
  createAdminSupportLinkAttachment,
  imageFileToAdminSupportAttachment,
  parseAdminSupportAttachments,
  serialiseAdminSupportAttachments,
} from "@/lib/admin/support-attachments";

import styles from "../css/AdminSupport.module.css";
import attachmentStyles from "../css/AdminSupportAttachments.module.css";

type Props = {
  portal: string;
};

type StatusFilter = "all" | AdminCRMCaseStatus;
type PriorityFilter = "all" | AdminCRMCasePriority;
type Visibility = "customer" | "internal";

const PAGE_SIZE = 50;
const LIVE_REFRESH_MS = 8000;

function errorMessage(value: unknown): string {
  if (value instanceof AdminRequestError) return value.message;
  if (value instanceof Error) return value.message;
  return "Unable to complete the request.";
}

function formatDateTime(value?: string): string {
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

function formatRelative(value?: string): string {
  if (!value) return "—";
  const time = new Date(value).getTime();
  if (Number.isNaN(time)) return "—";

  const difference = Date.now() - time;
  const minute = 60_000;
  const hour = 60 * minute;
  const day = 24 * hour;

  if (difference < minute) return "Just now";
  if (difference < hour) return `${Math.max(1, Math.floor(difference / minute))}m ago`;
  if (difference < day) return `${Math.max(1, Math.floor(difference / hour))}h ago`;
  if (difference < 7 * day) return `${Math.max(1, Math.floor(difference / day))}d ago`;
  return formatDateTime(value);
}

function titleCase(value: string): string {
  return value
    .replace(/[_-]+/g, " ")
    .replace(/\b\w/g, (character) => character.toUpperCase());
}

function statusLabel(status: AdminCRMCaseStatus): string {
  switch (status) {
    case "waiting_support":
      return "Needs response";
    case "waiting_customer":
      return "Waiting customer";
    case "resolved":
      return "Resolved";
    case "closed":
      return "Closed";
  }
}

function priorityLabel(priority: AdminCRMCasePriority): string {
  return titleCase(priority);
}

function statusTone(status: AdminCRMCaseStatus): string {
  switch (status) {
    case "waiting_support":
      return styles.statusWaitingSupport;
    case "waiting_customer":
      return styles.statusWaitingCustomer;
    case "resolved":
      return styles.statusResolved;
    case "closed":
      return styles.statusClosed;
  }
}

function priorityTone(priority: AdminCRMCasePriority): string {
  switch (priority) {
    case "urgent":
      return styles.priorityUrgent;
    case "high":
      return styles.priorityHigh;
    case "normal":
      return styles.priorityNormal;
    case "low":
      return styles.priorityLow;
  }
}

function contextSummary(value: unknown): string | null {
  if (!value || typeof value !== "object") return null;

  const record = value as Record<string, unknown>;
  const candidates = [
    record.message,
    record.description,
    record.reason,
    record.issue,
    record.question,
  ];

  for (const candidate of candidates) {
    if (typeof candidate === "string" && candidate.trim()) {
      return candidate.trim();
    }
  }

  return null;
}

function queueName(caseItem: AdminCRMCase): string {
  return caseItem.queue?.name || caseItem.queue?.code || "Support";
}

function isLikelyMine(caseItem: AdminCRMCase, fullName: string): boolean {
  const actor = caseItem.assignment?.assigned_actor;
  if (!actor) return false;
  return actor.display_name.trim().toLocaleLowerCase() === fullName.trim().toLocaleLowerCase();
}

function PaperclipIcon() {
  return (
    <svg viewBox="0 0 24 24" width="17" height="17" aria-hidden="true">
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

function LinkIcon() {
  return (
    <svg viewBox="0 0 24 24" width="16" height="16" aria-hidden="true">
      <path
        d="M9.8 14.2 14.2 9.8M7.1 16.9l-1.3 1.3a3.8 3.8 0 0 1-5.4-5.4l3.2-3.2A3.8 3.8 0 0 1 9 9.6M16.9 7.1l1.3-1.3a3.8 3.8 0 0 1 5.4 5.4l-3.2 3.2a3.8 3.8 0 0 1-5.4 0"
        fill="none"
        stroke="currentColor"
        strokeWidth="1.8"
        strokeLinecap="round"
      />
    </svg>
  );
}

function MessageAttachments({
  value,
  onPreview,
}: {
  value: unknown;
  onPreview: (attachment: AdminCRMAttachment) => void;
}) {
  const attachments = parseAdminSupportAttachments(value);

  if (attachments.length === 0) return null;

  return (
    <div className={attachmentStyles.messageAttachments}>
      {attachments.map((attachment) =>
        attachment.kind === "image" ? (
          <button
            key={attachment.id}
            type="button"
            className={attachmentStyles.messageImageButton}
            onClick={() => onPreview(attachment)}
          >
            <img
              src={attachment.url}
              alt={attachment.name}
              className={attachmentStyles.messageImage}
            />
          </button>
        ) : (
          <a
            key={attachment.id}
            href={attachment.url}
            target="_blank"
            rel="noopener noreferrer"
            className={attachmentStyles.messageLink}
          >
            <span aria-hidden="true">↗</span>
            <span>{attachment.name}</span>
          </a>
        ),
      )}
    </div>
  );
}

export default function AdminSupportWorkspace({ portal }: Props) {
  const principal = useAdminSession();
  const isSuperAdmin = principal.staff.roles.includes("admin_superuser");
  const permissions = principal.staff.permissions;

  const hasPermission = useCallback(
    (permission: string) => isSuperAdmin || permissions.includes(permission),
    [isSuperAdmin, permissions],
  );

  const canRead = hasPermission("admin.crm.read");
  const canManage = hasPermission("admin.crm.manage");

  const [queues, setQueues] = useState<AdminCRMQueue[]>([]);
  const [cases, setCases] = useState<AdminCRMCase[]>([]);
  const [canLoadMore, setCanLoadMore] = useState(false);

  const [query, setQuery] = useState("");
  const [status, setStatus] = useState<StatusFilter>("all");
  const [priority, setPriority] = useState<PriorityFilter>("all");
  const [queueCode, setQueueCode] = useState("all");

  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [selectedCase, setSelectedCase] = useState<AdminCRMCase | null>(null);
  const [messages, setMessages] = useState<AdminCRMMessage[]>([]);

  const [loading, setLoading] = useState(true);
  const [loadingMore, setLoadingMore] = useState(false);
  const [detailLoading, setDetailLoading] = useState(false);
  const [mutating, setMutating] = useState(false);
  const [error, setError] = useState("");
  const [detailError, setDetailError] = useState("");

  const [message, setMessage] = useState("");
  const [visibility, setVisibility] = useState<Visibility>("customer");
  const [attachments, setAttachments] = useState<AdminCRMAttachment[]>([]);
  const [attachmentBusy, setAttachmentBusy] = useState(false);
  const [attachmentError, setAttachmentError] = useState("");
  const [draggingAttachment, setDraggingAttachment] = useState(false);
  const [linkOpen, setLinkOpen] = useState(false);
  const [linkDraft, setLinkDraft] = useState("");
  const [previewAttachment, setPreviewAttachment] = useState<AdminCRMAttachment | null>(null);
  const [escalationQueue, setEscalationQueue] = useState("");
  const [liveAt, setLiveAt] = useState<Date | null>(null);

  const listAbortRef = useRef<AbortController | null>(null);
  const loadedOffsetRef = useRef(0);
  const drawerRef = useRef<HTMLDivElement | null>(null);
  const attachmentInputRef = useRef<HTMLInputElement | null>(null);

  const loadQueues = useCallback(async () => {
    if (!canRead) return;

    try {
      const response = await adminFetch<AdminCRMQueuesResponse>("/crm/queues");
      setQueues(response.data ?? []);
    } catch {
      // The case list is still useful when queue metadata cannot load.
    }
  }, [canRead]);

  const loadCases = useCallback(
    async (options?: { append?: boolean; silent?: boolean }) => {
      if (!canRead) {
        setLoading(false);
        return;
      }

      const append = options?.append ?? false;
      const silent = options?.silent ?? false;
      const offset = append ? loadedOffsetRef.current : 0;

      if (!append) {
        listAbortRef.current?.abort();
        listAbortRef.current = new AbortController();
      }

      if (append) setLoadingMore(true);
      else if (!silent) setLoading(true);

      if (!silent) setError("");

      try {
        const response = await adminFetch<AdminCRMCasesResponse>(
          `/crm/cases?limit=${PAGE_SIZE}&offset=${offset}`,
          !append && listAbortRef.current
            ? { signal: listAbortRef.current.signal }
            : undefined,
        );

        const next = response.data ?? [];
        setCases((current) => {
          if (!append) return next;

          const known = new Set(current.map((item) => item.id));
          return [...current, ...next.filter((item) => !known.has(item.id))];
        });
        loadedOffsetRef.current = offset + next.length;
        setCanLoadMore(next.length === PAGE_SIZE);
        setLiveAt(new Date());
      } catch (value: unknown) {
        if (value instanceof DOMException && value.name === "AbortError") return;
        if (!silent) setError(errorMessage(value));
      } finally {
        if (append) setLoadingMore(false);
        else if (!silent) setLoading(false);
      }
    },
    [canRead],
  );

  const loadSelected = useCallback(
    async (caseId: string, silent = false) => {
      if (!canRead) return;

      if (!silent) {
        setDetailLoading(true);
        setDetailError("");
      }

      try {
        const [caseResponse, messagesResponse] = await Promise.all([
          adminFetch<AdminCRMCaseResponse>(`/crm/cases/${caseId}`),
          adminFetch<AdminCRMMessagesResponse>(
            `/crm/cases/${caseId}/messages?limit=200&offset=0`,
          ),
        ]);

        setSelectedCase(caseResponse.data);
        setMessages(messagesResponse.data ?? []);
        setCases((current) =>
          current.map((item) =>
            item.id === caseResponse.data.id ? caseResponse.data : item,
          ),
        );
      } catch (value: unknown) {
        if (!silent) setDetailError(errorMessage(value));
      } finally {
        if (!silent) setDetailLoading(false);
      }
    },
    [canRead],
  );

  useEffect(() => {
    void loadQueues();
    void loadCases();

    return () => listAbortRef.current?.abort();
  }, [loadCases, loadQueues]);

  useEffect(() => {
    if (!selectedId) {
      setSelectedCase(null);
      setMessages([]);
      setDetailError("");
      setMessage("");
      setAttachments([]);
      setAttachmentError("");
      setDraggingAttachment(false);
      setLinkOpen(false);
      setLinkDraft("");
      setPreviewAttachment(null);
      setEscalationQueue("");
      return;
    }

    void loadSelected(selectedId);
  }, [loadSelected, selectedId]);

  useEffect(() => {
    const interval = window.setInterval(() => {
      if (document.visibilityState !== "visible" || mutating) return;
      void loadCases({ silent: true });
      if (selectedId) void loadSelected(selectedId, true);
    }, LIVE_REFRESH_MS);

    return () => window.clearInterval(interval);
  }, [loadCases, loadSelected, mutating, selectedId]);

  useEffect(() => {
    if (!selectedCase) return;

    const alternatives = queues.filter(
      (queue) => queue.code !== selectedCase.queue.code,
    );

    if (!alternatives.some((queue) => queue.code === escalationQueue)) {
      setEscalationQueue(alternatives[0]?.code ?? "");
    }
  }, [escalationQueue, queues, selectedCase]);

  const filteredCases = useMemo(() => {
    const normalizedQuery = query.trim().toLocaleLowerCase();

    return cases.filter((item) => {
      if (status !== "all" && item.status !== status) return false;
      if (priority !== "all" && item.priority !== priority) return false;
      if (queueCode !== "all" && item.queue.code !== queueCode) return false;

      if (!normalizedQuery) return true;

      const searchable = [
        item.case_number,
        item.subject,
        item.case_type,
        item.customer_id,
        item.order_id,
        item.product_id,
        item.queue.name,
        item.assignment?.assigned_actor?.display_name,
        contextSummary(item.context_snapshot),
      ]
        .filter(Boolean)
        .join(" ")
        .toLocaleLowerCase();

      return searchable.includes(normalizedQuery);
    });
  }, [cases, priority, query, queueCode, status]);

  const summary = useMemo(() => {
    let needsResponse = 0;
    let waitingCustomer = 0;
    let priorityCases = 0;
    let resolved = 0;

    for (const item of cases) {
      if (item.status === "waiting_support") needsResponse += 1;
      if (item.status === "waiting_customer") waitingCustomer += 1;
      if (item.priority === "urgent" || item.priority === "high") priorityCases += 1;
      if (item.status === "resolved") resolved += 1;
    }

    return { needsResponse, waitingCustomer, priorityCases, resolved };
  }, [cases]);

  const selectedIsMine = useMemo(() => {
    if (!selectedCase) return false;
    return isLikelyMine(selectedCase, principal.staff.full_name);
  }, [principal.staff.full_name, selectedCase]);

  const runMutation = useCallback(
    async (action: () => Promise<unknown>) => {
      if (!selectedId || mutating) return;

      setMutating(true);
      setDetailError("");

      try {
        await action();
        await Promise.all([
          loadSelected(selectedId, true),
          loadCases({ silent: true }),
        ]);
      } catch (value: unknown) {
        setDetailError(errorMessage(value));
      } finally {
        setMutating(false);
      }
    },
    [loadCases, loadSelected, mutating, selectedId],
  );

  const claimCase = useCallback(() => {
    if (!selectedId) return;

    void runMutation(() =>
      adminFetch<AdminCRMCaseResponse>(`/crm/cases/${selectedId}/claim`, {
        method: "POST",
        body: JSON.stringify({}),
      }),
    );
  }, [runMutation, selectedId]);

  const addImageFiles = useCallback(
    async (files: File[]) => {
      const imageFiles = files.filter((file) => file.type.startsWith("image/"));
      if (imageFiles.length === 0) {
        setAttachmentError("Choose an image or screenshot.");
        return;
      }

      const remaining = Math.max(0, MAX_ADMIN_SUPPORT_ATTACHMENTS - attachments.length);
      if (remaining === 0) {
        setAttachmentError(`You can attach up to ${MAX_ADMIN_SUPPORT_ATTACHMENTS} items.`);
        return;
      }

      setAttachmentBusy(true);
      setAttachmentError("");

      try {
        const prepared: AdminCRMAttachment[] = [];

        for (const file of imageFiles.slice(0, remaining)) {
          prepared.push(await imageFileToAdminSupportAttachment(file));
        }

        setAttachments((current) => [
          ...current,
          ...prepared.slice(0, Math.max(0, MAX_ADMIN_SUPPORT_ATTACHMENTS - current.length)),
        ]);

        if (imageFiles.length > remaining) {
          setAttachmentError(`Only ${MAX_ADMIN_SUPPORT_ATTACHMENTS} attachments can be sent at once.`);
        }
      } catch (value: unknown) {
        setAttachmentError(errorMessage(value));
      } finally {
        setAttachmentBusy(false);
      }
    },
    [attachments.length],
  );

  const chooseAttachment = useCallback(
    (event: ChangeEvent<HTMLInputElement>) => {
      const files = Array.from(event.target.files ?? []);
      event.target.value = "";
      void addImageFiles(files);
    },
    [addImageFiles],
  );

  const pasteAttachment = useCallback(
    (event: ClipboardEvent<HTMLTextAreaElement>) => {
      const files = Array.from(event.clipboardData.files).filter((file) =>
        file.type.startsWith("image/"),
      );

      if (files.length === 0) return;

      event.preventDefault();
      void addImageFiles(files);
    },
    [addImageFiles],
  );

  const dragAttachmentOver = useCallback((event: DragEvent<HTMLFormElement>) => {
    if (!Array.from(event.dataTransfer.types).includes("Files")) return;
    event.preventDefault();
    event.dataTransfer.dropEffect = "copy";
    setDraggingAttachment(true);
  }, []);

  const dragAttachmentLeave = useCallback((event: DragEvent<HTMLFormElement>) => {
    const next = event.relatedTarget as Node | null;
    if (next && event.currentTarget.contains(next)) return;
    setDraggingAttachment(false);
  }, []);

  const dropAttachment = useCallback(
    (event: DragEvent<HTMLFormElement>) => {
      event.preventDefault();
      setDraggingAttachment(false);
      void addImageFiles(Array.from(event.dataTransfer.files));
    },
    [addImageFiles],
  );

  const removeAttachment = useCallback((attachmentID: string) => {
    setAttachments((current) => current.filter((item) => item.id !== attachmentID));
    setAttachmentError("");
  }, []);

  const addLinkAttachment = useCallback(() => {
    if (attachments.length >= MAX_ADMIN_SUPPORT_ATTACHMENTS) {
      setAttachmentError(`You can attach up to ${MAX_ADMIN_SUPPORT_ATTACHMENTS} items.`);
      return;
    }

    try {
      const attachment = createAdminSupportLinkAttachment(linkDraft);
      setAttachments((current) => [...current, attachment]);
      setLinkDraft("");
      setLinkOpen(false);
      setAttachmentError("");
    } catch (value: unknown) {
      setAttachmentError(errorMessage(value));
    }
  }, [attachments.length, linkDraft]);

  const submitMessage = useCallback(
    (event: FormEvent<HTMLFormElement>) => {
      event.preventDefault();
      if (!selectedId || (!message.trim() && attachments.length === 0)) return;

      const body = message.trim();
      const payloadAttachments = serialiseAdminSupportAttachments(attachments);

      void runMutation(async () => {
        await adminFetch<AdminCRMMessageResponse>(
          `/crm/cases/${selectedId}/messages`,
          {
            method: "POST",
            body: JSON.stringify({
              message: body,
              visibility,
              attachments: payloadAttachments,
            }),
          },
        );

        setMessage("");
        setAttachments([]);
        setAttachmentError("");
        setLinkOpen(false);
        setLinkDraft("");
      });
    },
    [attachments, message, runMutation, selectedId, visibility],
  );

  const resolveCase = useCallback(() => {
    if (!selectedId) return;

    void runMutation(() =>
      adminFetch<AdminCRMCaseResponse>(`/crm/cases/${selectedId}/resolve`, {
        method: "POST",
        body: JSON.stringify({}),
      }),
    );
  }, [runMutation, selectedId]);

  const escalateCase = useCallback(() => {
    if (!selectedId || !escalationQueue) return;

    void runMutation(() =>
      adminFetch<AdminCRMEscalationResponse>(`/crm/cases/${selectedId}/escalate`, {
        method: "POST",
        body: JSON.stringify({ queue_code: escalationQueue }),
      }),
    );
  }, [escalationQueue, runMutation, selectedId]);

  const openCase = useCallback((caseId: string) => {
    setSelectedId(caseId);
    window.setTimeout(() => {
      drawerRef.current?.focus();
    }, 0);
  }, []);

  const closeDrawer = useCallback(() => {
    if (!mutating) setSelectedId(null);
  }, [mutating]);

  if (!canRead) {
    return (
      <section className={styles.permissionState}>
        <div className={styles.permissionIcon}>◌</div>
        <p className={styles.eyebrow}>Customer support</p>
        <h1>Support access required</h1>
        <p>
          Your staff account does not currently have permission to read CRM support cases.
        </p>
      </section>
    );
  }

  return (
    <section className={styles.workspace}>
      <header className={styles.hero}>
        <div>
          <p className={styles.eyebrow}>Customer operations</p>
          <h1>Support</h1>
          <p className={styles.heroCopy}>
            Triage customer cases, reply, add internal notes, escalate and resolve from one queue.
          </p>
        </div>

        <div className={styles.livePill}>
          <span className={styles.liveDot} aria-hidden="true" />
          <span>
            Live{liveAt ? ` · ${formatDateTime(liveAt.toISOString())}` : ""}
          </span>
        </div>
      </header>

      <div className={styles.summaryGrid}>
        <article className={`${styles.summaryCard} ${styles.summaryBlue}`}>
          <p>Needs response</p>
          <strong>{summary.needsResponse}</strong>
          <span>Customer is waiting for support</span>
        </article>

        <article className={`${styles.summaryCard} ${styles.summaryViolet}`}>
          <p>Waiting customer</p>
          <strong>{summary.waitingCustomer}</strong>
          <span>Support has replied</span>
        </article>

        <article className={`${styles.summaryCard} ${styles.summaryGold}`}>
          <p>Priority</p>
          <strong>{summary.priorityCases}</strong>
          <span>Urgent or high-priority cases</span>
        </article>

        <article className={`${styles.summaryCard} ${styles.summaryGreen}`}>
          <p>Resolved</p>
          <strong>{summary.resolved}</strong>
          <span>Resolved in the loaded queue</span>
        </article>
      </div>

      <section className={styles.queuePanel}>
        <div className={styles.queueHeader}>
          <div>
            <p className={styles.sectionEyebrow}>Support queue</p>
            <h2>{filteredCases.length} visible cases</h2>
          </div>

          <button
            type="button"
            className={styles.refreshButton}
            onClick={() => void loadCases()}
            disabled={loading}
          >
            Refresh
          </button>
        </div>

        <div className={styles.filters}>
          <label className={styles.searchField}>
            <span className={styles.searchIcon} aria-hidden="true">⌕</span>
            <input
              type="search"
              value={query}
              onChange={(event) => setQuery(event.target.value)}
              placeholder="Case number, subject, customer ID, order ID..."
              aria-label="Filter loaded support cases"
            />
          </label>

          <label className={styles.filterField}>
            <span>Status</span>
            <select
              value={status}
              onChange={(event) => setStatus(event.target.value as StatusFilter)}
            >
              <option value="all">All statuses</option>
              <option value="waiting_support">Needs response</option>
              <option value="waiting_customer">Waiting customer</option>
              <option value="resolved">Resolved</option>
              <option value="closed">Closed</option>
            </select>
          </label>

          <label className={styles.filterField}>
            <span>Priority</span>
            <select
              value={priority}
              onChange={(event) => setPriority(event.target.value as PriorityFilter)}
            >
              <option value="all">All priorities</option>
              <option value="urgent">Urgent</option>
              <option value="high">High</option>
              <option value="normal">Normal</option>
              <option value="low">Low</option>
            </select>
          </label>

          <label className={styles.filterField}>
            <span>Queue</span>
            <select
              value={queueCode}
              onChange={(event) => setQueueCode(event.target.value)}
            >
              <option value="all">All queues</option>
              {queues.map((queue) => (
                <option key={queue.id} value={queue.code}>
                  {queue.name}
                </option>
              ))}
            </select>
          </label>
        </div>

        {error ? <div className={styles.errorBanner}>{error}</div> : null}

        {loading ? (
          <div className={styles.loadingState}>Loading support cases…</div>
        ) : filteredCases.length === 0 ? (
          <div className={styles.emptyState}>
            <div className={styles.emptyIcon}>◌</div>
            <h3>No matching cases</h3>
            <p>Adjust the queue filters or wait for a new support case.</p>
          </div>
        ) : (
          <>
            <div className={styles.desktopTableWrap}>
              <table className={styles.caseTable}>
                <thead>
                  <tr>
                    <th>Case</th>
                    <th>Queue</th>
                    <th>Status</th>
                    <th>Priority</th>
                    <th>Assigned</th>
                    <th>Last activity</th>
                    <th aria-label="Open case" />
                  </tr>
                </thead>
                <tbody>
                  {filteredCases.map((item) => (
                    <tr key={item.id} onClick={() => openCase(item.id)}>
                      <td>
                        <strong>{item.case_number}</strong>
                        <span>{item.subject}</span>
                        <small>{titleCase(item.case_type)}</small>
                      </td>
                      <td>
                        <strong>{queueName(item)}</strong>
                        <span>{item.queue.code}</span>
                      </td>
                      <td>
                        <span className={`${styles.statusBadge} ${statusTone(item.status)}`}>
                          {statusLabel(item.status)}
                        </span>
                      </td>
                      <td>
                        <span className={`${styles.priorityBadge} ${priorityTone(item.priority)}`}>
                          {priorityLabel(item.priority)}
                        </span>
                      </td>
                      <td>
                        <strong>{item.assignment?.assigned_actor?.display_name || "Unassigned"}</strong>
                        <span>{item.assignment?.assignment_type || "Queue"}</span>
                      </td>
                      <td>
                        <strong>{formatRelative(item.last_message_at)}</strong>
                        <span>{formatDateTime(item.last_message_at)}</span>
                      </td>
                      <td>
                        <button
                          type="button"
                          className={styles.openButton}
                          onClick={(event) => {
                            event.stopPropagation();
                            openCase(item.id);
                          }}
                        >
                          Open
                        </button>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>

            <div className={styles.mobileCases}>
              {filteredCases.map((item) => (
                <button
                  key={item.id}
                  type="button"
                  className={styles.mobileCaseCard}
                  onClick={() => openCase(item.id)}
                >
                  <div className={styles.mobileCaseTop}>
                    <div>
                      <strong>{item.case_number}</strong>
                      <span>{queueName(item)}</span>
                    </div>
                    <span className={`${styles.statusBadge} ${statusTone(item.status)}`}>
                      {statusLabel(item.status)}
                    </span>
                  </div>

                  <h3>{item.subject}</h3>

                  <div className={styles.mobileMetaRow}>
                    <span className={`${styles.priorityBadge} ${priorityTone(item.priority)}`}>
                      {priorityLabel(item.priority)}
                    </span>
                    <span>{item.assignment?.assigned_actor?.display_name || "Unassigned"}</span>
                    <span>{formatRelative(item.last_message_at)}</span>
                  </div>
                </button>
              ))}
            </div>
          </>
        )}

        {canLoadMore ? (
          <div className={styles.loadMoreRow}>
            <button
              type="button"
              className={styles.loadMoreButton}
              onClick={() => void loadCases({ append: true })}
              disabled={loadingMore}
            >
              {loadingMore ? "Loading…" : "Load more cases"}
            </button>
          </div>
        ) : null}

        <p className={styles.queueFootnote}>
          Search and filters apply to the loaded support queue. More cases can be loaded in batches of {PAGE_SIZE}.
        </p>
      </section>

      {selectedId ? (
        <div className={styles.drawerLayer} role="presentation" onMouseDown={(event) => {
          if (event.target === event.currentTarget) closeDrawer();
        }}>
          <div
            ref={drawerRef}
            className={styles.drawer}
            role="dialog"
            aria-modal="true"
            aria-label="Support case"
            tabIndex={-1}
          >
            <div className={styles.drawerTopbar}>
              <div>
                <p className={styles.sectionEyebrow}>Support case</p>
                <h2>{selectedCase?.case_number || "Loading case…"}</h2>
              </div>

              <button
                type="button"
                className={styles.closeButton}
                onClick={closeDrawer}
                disabled={mutating}
                aria-label="Close support case"
              >
                ×
              </button>
            </div>

            {detailError ? <div className={styles.drawerError}>{detailError}</div> : null}

            {detailLoading && !selectedCase ? (
              <div className={styles.drawerLoading}>Loading case…</div>
            ) : selectedCase ? (
              <div className={styles.drawerBody}>
                <section className={styles.caseOverview}>
                  <div className={styles.caseTitleRow}>
                    <div>
                      <div className={styles.badgeRow}>
                        <span className={`${styles.statusBadge} ${statusTone(selectedCase.status)}`}>
                          {statusLabel(selectedCase.status)}
                        </span>
                        <span className={`${styles.priorityBadge} ${priorityTone(selectedCase.priority)}`}>
                          {priorityLabel(selectedCase.priority)}
                        </span>
                        <span className={styles.queueBadge}>{queueName(selectedCase)}</span>
                      </div>
                      <h3>{selectedCase.subject}</h3>
                      <p>{titleCase(selectedCase.case_type)}</p>
                    </div>
                  </div>

                  <div className={styles.factGrid}>
                    <div>
                      <span>Customer ID</span>
                      <strong>{selectedCase.customer_id}</strong>
                    </div>
                    <div>
                      <span>Assigned to</span>
                      <strong>{selectedCase.assignment?.assigned_actor?.display_name || "Unassigned"}</strong>
                    </div>
                    <div>
                      <span>Created</span>
                      <strong>{formatDateTime(selectedCase.created_at)}</strong>
                    </div>
                    <div>
                      <span>Last activity</span>
                      <strong>{formatRelative(selectedCase.last_message_at)}</strong>
                    </div>
                  </div>

                  {selectedCase.order_id || selectedCase.product_id ? (
                    <div className={styles.linkedContext}>
                      {selectedCase.order_id ? (
                        <div>
                          <span>Linked order</span>
                          <strong>{selectedCase.order_id}</strong>
                        </div>
                      ) : null}
                      {selectedCase.product_id ? (
                        <div>
                          <span>Linked product</span>
                          <strong>{selectedCase.product_id}</strong>
                        </div>
                      ) : null}
                    </div>
                  ) : null}
                </section>

                <section className={styles.ownershipPanel}>
                  <div>
                    <p className={styles.sectionEyebrow}>Ownership</p>
                    <h3>
                      {selectedCase.assignment?.assigned_actor
                        ? selectedIsMine
                          ? "This case is assigned to you"
                          : `Assigned to ${selectedCase.assignment.assigned_actor.display_name}`
                        : "This case is unassigned"}
                    </h3>
                    <p>
                      {selectedCase.assignment?.assigned_actor
                        ? selectedIsMine
                          ? "You can reply, add internal notes, escalate or resolve this case."
                          : "The current owner controls case mutations."
                        : "Claim the case before replying or changing its lifecycle."}
                    </p>
                  </div>

                  {canManage && !selectedCase.assignment?.assigned_actor && selectedCase.status !== "closed" ? (
                    <button
                      type="button"
                      className={styles.primaryButton}
                      onClick={claimCase}
                      disabled={mutating}
                    >
                      {mutating ? "Claiming…" : "Claim case"}
                    </button>
                  ) : null}
                </section>

                <section className={styles.conversationPanel}>
                  <div className={styles.sectionHeader}>
                    <div>
                      <p className={styles.sectionEyebrow}>Conversation</p>
                      <h3>{messages.length} messages</h3>
                    </div>
                  </div>

                  <div className={styles.messageList}>
                    {messages.length === 0 ? (
                      <div className={styles.noMessages}>No messages have been recorded yet.</div>
                    ) : (
                      messages.map((item) => {
                        const internal = item.visibility === "internal";
                        const support = item.author_type === "support";

                        return (
                          <article
                            key={item.id}
                            className={[
                              styles.message,
                              support ? styles.messageSupport : styles.messageCustomer,
                              internal ? styles.messageInternal : "",
                            ].filter(Boolean).join(" ")}
                          >
                            <div className={styles.messageMeta}>
                              <strong>
                                {support
                                  ? item.support_actor?.display_name || "Support"
                                  : titleCase(item.author_type || "Customer")}
                              </strong>
                              <span>{internal ? "Internal note" : "Customer visible"}</span>
                              <time>{formatDateTime(item.created_at)}</time>
                            </div>
                            {
                              item.body &&
                              !(
                                item.body === "Shared an attachment." &&
                                parseAdminSupportAttachments(item.attachments).length > 0
                              ) ? (
                                <p>{item.body}</p>
                              ) : null
                            }

                            <MessageAttachments
                              value={item.attachments}
                              onPreview={setPreviewAttachment}
                            />
                          </article>
                        );
                      })
                    )}
                  </div>

                  {canManage && selectedIsMine && selectedCase.status !== "closed" ? (
                    <form
                      className={[
                        styles.composer,
                        draggingAttachment ? attachmentStyles.composerDragging : "",
                      ]
                        .filter(Boolean)
                        .join(" ")}
                      onSubmit={submitMessage}
                      onDragOver={dragAttachmentOver}
                      onDragLeave={dragAttachmentLeave}
                      onDrop={dropAttachment}
                    >
                      <input
                        ref={attachmentInputRef}
                        type="file"
                        accept="image/jpeg,image/png,image/webp,image/gif"
                        multiple
                        className={attachmentStyles.attachmentInput}
                        onChange={chooseAttachment}
                        tabIndex={-1}
                        aria-hidden="true"
                      />

                      <div className={styles.composerTabs} role="group" aria-label="Message visibility">
                        <button
                          type="button"
                          className={visibility === "customer" ? styles.composerTabActive : styles.composerTab}
                          onClick={() => setVisibility("customer")}
                        >
                          Reply to customer
                        </button>
                        <button
                          type="button"
                          className={visibility === "internal" ? styles.composerTabActive : styles.composerTab}
                          onClick={() => setVisibility("internal")}
                        >
                          Internal note
                        </button>
                      </div>

                      <div className={attachmentStyles.composerSurface}>
                        {draggingAttachment ? (
                          <div className={attachmentStyles.dropOverlay}>
                            <PaperclipIcon />
                            <strong>Drop screenshots here</strong>
                            <span>Images will be prepared before sending.</span>
                          </div>
                        ) : null}

                        <textarea
                          value={message}
                          onChange={(event) => setMessage(event.target.value)}
                          onPaste={pasteAttachment}
                          maxLength={5000}
                          rows={5}
                          placeholder={
                            visibility === "customer"
                              ? "Write a response the customer will receive…"
                              : "Add an internal note for staff only…"
                          }
                        />

                        {attachments.length > 0 ? (
                          <div className={attachmentStyles.attachmentPreviewGrid}>
                            {attachments.map((attachment) => (
                              <div key={attachment.id} className={attachmentStyles.attachmentPreview}>
                                {attachment.kind === "image" ? (
                                  <button
                                    type="button"
                                    className={attachmentStyles.attachmentPreviewImageButton}
                                    onClick={() => setPreviewAttachment(attachment)}
                                    aria-label={`Preview ${attachment.name}`}
                                  >
                                    <img src={attachment.url} alt="" className={attachmentStyles.attachmentPreviewImage} />
                                  </button>
                                ) : (
                                  <span className={attachmentStyles.attachmentPreviewLinkIcon} aria-hidden="true">
                                    <LinkIcon />
                                  </span>
                                )}

                                <div className={attachmentStyles.attachmentPreviewCopy}>
                                  <strong>{attachment.name}</strong>
                                  <span>{attachment.kind === "image" ? "Screenshot" : "Link"}</span>
                                </div>

                                <button
                                  type="button"
                                  className={attachmentStyles.attachmentRemoveButton}
                                  onClick={() => removeAttachment(attachment.id)}
                                  aria-label={`Remove ${attachment.name}`}
                                >
                                  ×
                                </button>
                              </div>
                            ))}
                          </div>
                        ) : null}

                        {linkOpen ? (
                          <div className={attachmentStyles.linkComposer}>
                            <input
                              type="url"
                              value={linkDraft}
                              onChange={(event) => setLinkDraft(event.target.value)}
                              placeholder="https://example.com/reference"
                              maxLength={4096}
                              autoFocus
                            />
                            <button
                              type="button"
                              className={attachmentStyles.linkSaveButton}
                              onClick={addLinkAttachment}
                              disabled={!linkDraft.trim()}
                            >
                              Add link
                            </button>
                            <button
                              type="button"
                              className={attachmentStyles.linkCancelButton}
                              onClick={() => {
                                setLinkOpen(false);
                                setLinkDraft("");
                                setAttachmentError("");
                              }}
                            >
                              Cancel
                            </button>
                          </div>
                        ) : null}

                        {attachmentError ? (
                          <p className={attachmentStyles.attachmentError} role="alert">
                            {attachmentError}
                          </p>
                        ) : null}

                        <div className={attachmentStyles.attachmentToolbar}>
                          <div className={attachmentStyles.attachmentActions}>
                            <button
                              type="button"
                              className={attachmentStyles.attachmentAction}
                              onClick={() => attachmentInputRef.current?.click()}
                              disabled={
                                attachmentBusy ||
                                attachments.length >= MAX_ADMIN_SUPPORT_ATTACHMENTS
                              }
                            >
                              <PaperclipIcon />
                              <span>{attachmentBusy ? "Preparing…" : "Photo or screenshot"}</span>
                            </button>

                            <button
                              type="button"
                              className={attachmentStyles.attachmentAction}
                              onClick={() => {
                                setLinkOpen((current) => !current);
                                setAttachmentError("");
                              }}
                              disabled={attachments.length >= MAX_ADMIN_SUPPORT_ATTACHMENTS}
                            >
                              <LinkIcon />
                              <span>Add link</span>
                            </button>
                          </div>

                          <span className={attachmentStyles.attachmentHint}>
                            Paste or drop screenshots · {attachments.length}/{MAX_ADMIN_SUPPORT_ATTACHMENTS}
                          </span>
                        </div>
                      </div>

                      <div className={styles.composerFooter}>
                        <span>{message.length.toLocaleString()} / 5,000</span>
                        <button
                          type="submit"
                          className={styles.primaryButton}
                          disabled={
                            mutating ||
                            attachmentBusy ||
                            (!message.trim() && attachments.length === 0)
                          }
                        >
                          {mutating ? "Saving…" : visibility === "customer" ? "Send reply" : "Add note"}
                        </button>
                      </div>
                    </form>
                  ) : null}
                </section>

                {canManage && selectedIsMine && selectedCase.status !== "closed" ? (
                  <section className={styles.caseActions}>
                    <div className={styles.actionBlock}>
                      <p className={styles.sectionEyebrow}>Escalation</p>
                      <h3>Move to another support queue</h3>
                      <p>Escalating releases your assignment and returns the case to the destination queue.</p>

                      <div className={styles.inlineAction}>
                        <select
                          value={escalationQueue}
                          onChange={(event) => setEscalationQueue(event.target.value)}
                          disabled={mutating}
                        >
                          {queues
                            .filter((queue) => queue.code !== selectedCase.queue.code)
                            .map((queue) => (
                              <option key={queue.id} value={queue.code}>
                                {queue.name}
                              </option>
                            ))}
                        </select>

                        <button
                          type="button"
                          className={styles.secondaryButton}
                          onClick={escalateCase}
                          disabled={mutating || !escalationQueue}
                        >
                          Escalate
                        </button>
                      </div>
                    </div>

                    <div className={styles.actionBlock}>
                      <p className={styles.sectionEyebrow}>Resolution</p>
                      <h3>Resolve this support case</h3>
                      <p>Resolution records the case as solved while preserving the conversation and audit history.</p>

                      <button
                        type="button"
                        className={styles.resolveButton}
                        onClick={resolveCase}
                        disabled={mutating || selectedCase.status === "resolved"}
                      >
                        {selectedCase.status === "resolved" ? "Already resolved" : "Mark resolved"}
                      </button>
                    </div>
                  </section>
                ) : null}

                {selectedCase.context_snapshot ? (
                  <details className={styles.contextPanel}>
                    <summary>Case context snapshot</summary>
                    <pre>{JSON.stringify(selectedCase.context_snapshot, null, 2)}</pre>
                  </details>
                ) : null}
              </div>
            ) : null}
          </div>
        </div>
      ) : null}

      {previewAttachment?.kind === "image" ? (
        <div
          className={attachmentStyles.imageLightbox}
          role="dialog"
          aria-modal="true"
          aria-label="Attachment preview"
          onMouseDown={(event) => {
            if (event.target === event.currentTarget) setPreviewAttachment(null);
          }}
        >
          <div className={attachmentStyles.imageLightboxPanel}>
            <div className={attachmentStyles.imageLightboxTopbar}>
              <strong>{previewAttachment.name}</strong>
              <button
                type="button"
                className={attachmentStyles.imageLightboxClose}
                onClick={() => setPreviewAttachment(null)}
                aria-label="Close image preview"
              >
                ×
              </button>
            </div>
            <img
              src={previewAttachment.url}
              alt={previewAttachment.name}
              className={attachmentStyles.imageLightboxImage}
            />
          </div>
        </div>
      ) : null}
    </section>
  );
}
