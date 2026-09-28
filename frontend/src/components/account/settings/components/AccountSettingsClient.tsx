"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import {
  type ChangeEvent,
  type FormEvent,
  useCallback,
  useEffect,
  useRef,
  useState,
} from "react";

import { Icon } from "@/components/ui/Icon";
import type {
  AccountCustomer,
  AccountDataResponse,
} from "@/lib/api/contracts/account";
import type {
  CustomerNotificationPreferences,
  CustomerPreferences,
  CustomerPrivacySettings,
} from "@/lib/account/settings-types";

import styles from "../css/AccountSettings.module.css";

type SettingsData = {
  customer: AccountCustomer;
  preferences: CustomerPreferences;
  notifications: CustomerNotificationPreferences;
  privacy: CustomerPrivacySettings;
};

type PageState =
  | { kind: "loading" }
  | { kind: "error"; message: string }
  | { kind: "ready"; data: SettingsData };

type Feedback = {
  kind: "success" | "error";
  message: string;
} | null;

type UploadTarget = {
  key: string;
  method: string;
  url: string;
  headers: Record<string, string>;
};

type ProfileForm = {
  full_name: string;
  email: string;
  phone: string;
};

const sections = [
  { href: "#profile", label: "Profile" },
  { href: "#display", label: "Language & currency" },
  { href: "#notifications", label: "Notifications" },
  { href: "#privacy", label: "Privacy" },
  { href: "#access", label: "Account access" },
] as const;

function initials(name: string) {
  return (
    name
      .trim()
      .split(/\s+/)
      .slice(0, 2)
      .map((part) => part[0]?.toUpperCase() ?? "")
      .join("") || "ED"
  );
}

function objectValue(value: unknown): Record<string, unknown> | null {
  return value && typeof value === "object"
    ? (value as Record<string, unknown>)
    : null;
}

function resolveUploadTarget(payload: unknown): UploadTarget | null {
  const root = objectValue(payload);
  const data = objectValue(root?.data) ?? root;
  const candidate = objectValue(data?.target) ?? objectValue(data?.upload) ?? data;

  if (!candidate) return null;

  const key = typeof candidate.key === "string" ? candidate.key : "";
  const url = typeof candidate.url === "string" ? candidate.url : "";
  const method = typeof candidate.method === "string" ? candidate.method : "PUT";
  const rawHeaders = objectValue(candidate.headers);
  const headers: Record<string, string> = {};

  if (rawHeaders) {
    for (const [name, value] of Object.entries(rawHeaders)) {
      if (typeof value === "string") headers[name] = value;
    }
  }

  return key && url
    ? {
        key,
        url,
        method,
        headers,
      }
    : null;
}

async function readError(response: Response, fallback: string) {
  try {
    const payload = (await response.json()) as {
      error?: { message?: string } | string;
      message?: string;
    };

    if (typeof payload.error === "string" && payload.error.trim()) {
      return payload.error;
    }

    if (
      typeof payload.error === "object" &&
      payload.error?.message?.trim()
    ) {
      return payload.error.message.trim();
    }

    return payload.message?.trim() || fallback;
  } catch {
    return fallback;
  }
}

async function getData<T>(url: string, fallback: string) {
  const response = await fetch(url, {
    cache: "no-store",
  });

  if (response.status === 401) {
    const error = new Error("Authentication required.") as Error & {
      status?: number;
    };
    error.status = 401;
    throw error;
  }

  if (!response.ok) {
    throw new Error(await readError(response, fallback));
  }

  return (await response.json()) as AccountDataResponse<T>;
}

async function patchData<T>(url: string, body: unknown) {
  const response = await fetch(url, {
    method: "PATCH",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify(body),
  });

  if (!response.ok) {
    throw new Error(await readError(response, "Unable to save changes."));
  }

  return (await response.json()) as AccountDataResponse<T>;
}

export function AccountSettingsClient() {
  const router = useRouter();
  const avatarInputRef = useRef<HTMLInputElement>(null);

  const [state, setState] = useState<PageState>({ kind: "loading" });
  const [busy, setBusy] = useState("");
  const [feedback, setFeedback] = useState<Feedback>(null);
  const [profileForm, setProfileForm] = useState<ProfileForm>({
    full_name: "",
    email: "",
    phone: "",
  });

  const load = useCallback(async (showLoading = true) => {
    if (showLoading) setState({ kind: "loading" });
    setFeedback(null);

    try {
      const [customer, preferences, notifications, privacy] = await Promise.all([
        getData<AccountCustomer>(
          "/api/storefront/account/settings/profile",
          "Unable to load your profile.",
        ),
        getData<CustomerPreferences>(
          "/api/storefront/account/settings/preferences",
          "Unable to load display preferences.",
        ),
        getData<CustomerNotificationPreferences>(
          "/api/storefront/account/settings/notifications",
          "Unable to load notification preferences.",
        ),
        getData<CustomerPrivacySettings>(
          "/api/storefront/account/settings/privacy",
          "Unable to load privacy settings.",
        ),
      ]);

      const data: SettingsData = {
        customer: customer.data,
        preferences: preferences.data,
        notifications: notifications.data,
        privacy: privacy.data,
      };

      setProfileForm({
        full_name: data.customer.full_name ?? "",
        email: data.customer.email ?? "",
        phone: data.customer.phone ?? "",
      });
      setState({ kind: "ready", data });
    } catch (caught) {
      const error = caught as Error & { status?: number };

      if (error.status === 401) {
        router.replace("/account/sign-in?next=%2Faccount%2Fsettings");
        return;
      }

      setState({
        kind: "error",
        message:
          caught instanceof Error
            ? caught.message
            : "Unable to load account settings.",
      });
    }
  }, [router]);

  useEffect(() => {
    let cancelled = false;

    void Promise.resolve().then(() => {
      if (!cancelled) void load();
    });

    return () => {
      cancelled = true;
    };
  }, [load]);

  function showSuccess(message: string) {
    setFeedback({ kind: "success", message });
  }

  function showError(caught: unknown, fallback: string) {
    setFeedback({
      kind: "error",
      message: caught instanceof Error ? caught.message : fallback,
    });
  }

  function updateSettingData<K extends keyof SettingsData>(
    key: K,
    value: SettingsData[K],
  ) {
    setState((current) =>
      current.kind === "ready"
        ? {
            kind: "ready",
            data: {
              ...current.data,
              [key]: value,
            },
          }
        : current,
    );
  }

  async function saveProfile(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (busy) return;

    if (!profileForm.full_name.trim() || !profileForm.phone.trim()) {
      setFeedback({
        kind: "error",
        message: "Name and phone are required.",
      });
      return;
    }

    setBusy("profile");
    setFeedback(null);

    try {
      const result = await patchData<AccountCustomer>(
        "/api/storefront/account/settings/profile",
        {
          full_name: profileForm.full_name.trim(),
          email: profileForm.email.trim(),
          phone: profileForm.phone.trim(),
        },
      );

      updateSettingData("customer", result.data);
      setProfileForm({
        full_name: result.data.full_name ?? "",
        email: result.data.email ?? "",
        phone: result.data.phone ?? "",
      });

      window.dispatchEvent(new Event("customer-auth-changed"));
      showSuccess("Profile updated.");
    } catch (caught) {
      showError(caught, "Unable to update your profile.");
    } finally {
      setBusy("");
    }
  }

  async function uploadAvatar(file: File) {
    if (busy) return;

    const supported = ["image/jpeg", "image/png", "image/webp"];

    if (!supported.includes(file.type)) {
      setFeedback({
        kind: "error",
        message: "Choose a JPEG, PNG or WebP image.",
      });
      return;
    }

    if (file.size <= 0 || file.size > 5 * 1024 * 1024) {
      setFeedback({
        kind: "error",
        message: "Profile photos must be 5 MB or smaller.",
      });
      return;
    }

    setBusy("avatar");
    setFeedback(null);

    try {
      const targetResponse = await fetch(
        "/api/storefront/account/avatar/upload-target",
        {
          method: "POST",
          headers: {
            "Content-Type": "application/json",
          },
          body: JSON.stringify({
            content_type: file.type,
            content_length: file.size,
          }),
        },
      );

      if (!targetResponse.ok) {
        throw new Error(
          await readError(targetResponse, "Unable to prepare the photo upload."),
        );
      }

      const target = resolveUploadTarget(await targetResponse.json());

      if (!target) {
        throw new Error("The photo upload target was incomplete.");
      }

      const uploadResponse = await fetch(target.url, {
        method: target.method || "PUT",
        headers: target.headers,
        body: file,
      });

      if (!uploadResponse.ok) {
        throw new Error("The photo could not be uploaded to storage.");
      }

      const completeResponse = await fetch(
        "/api/storefront/account/avatar/complete",
        {
          method: "POST",
          headers: {
            "Content-Type": "application/json",
          },
          body: JSON.stringify({ key: target.key }),
        },
      );

      if (!completeResponse.ok) {
        throw new Error(
          await readError(completeResponse, "Unable to save your profile photo."),
        );
      }

      const completed =
        (await completeResponse.json()) as AccountDataResponse<AccountCustomer>;

      updateSettingData("customer", completed.data);
      window.dispatchEvent(new Event("customer-auth-changed"));
      showSuccess("Profile photo updated.");
    } catch (caught) {
      showError(caught, "Unable to update your profile photo.");
    } finally {
      setBusy("");
      if (avatarInputRef.current) avatarInputRef.current.value = "";
    }
  }

  async function removeAvatar() {
    if (busy || state.kind !== "ready" || !state.data.customer.avatar_url) {
      return;
    }

    setBusy("avatar");
    setFeedback(null);

    try {
      const response = await fetch("/api/storefront/account/avatar", {
        method: "DELETE",
      });

      if (!response.ok) {
        throw new Error(
          await readError(response, "Unable to remove the profile photo."),
        );
      }

      const refreshed = await getData<AccountCustomer>(
        "/api/storefront/account/settings/profile",
        "Unable to refresh your profile.",
      );

      updateSettingData("customer", refreshed.data);
      window.dispatchEvent(new Event("customer-auth-changed"));
      showSuccess("Profile photo removed.");
    } catch (caught) {
      showError(caught, "Unable to remove your profile photo.");
    } finally {
      setBusy("");
    }
  }

  async function saveDisplayPreferences(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (busy || state.kind !== "ready") return;

    const locale =
      new FormData(event.currentTarget).get("locale")?.toString() || "en";

    setBusy("preferences");
    setFeedback(null);

    try {
      const result = await patchData<CustomerPreferences>(
        "/api/storefront/account/settings/preferences",
        { locale },
      );

      updateSettingData("preferences", result.data);
      showSuccess("Display preferences updated.");
    } catch (caught) {
      showError(caught, "Unable to update display preferences.");
    } finally {
      setBusy("");
    }
  }

  async function updateNotification(
    field: keyof CustomerNotificationPreferences,
    value: boolean,
  ) {
    if (busy || state.kind !== "ready") return;

    setBusy(`notification:${field}`);
    setFeedback(null);

    try {
      const result = await patchData<CustomerNotificationPreferences>(
        "/api/storefront/account/settings/notifications",
        { [field]: value },
      );

      updateSettingData("notifications", result.data);
      showSuccess("Notification preference updated.");
    } catch (caught) {
      showError(caught, "Unable to update notification preferences.");
    } finally {
      setBusy("");
    }
  }

  async function updatePrivacy(
    field: keyof CustomerPrivacySettings,
    value: boolean,
  ) {
    if (busy || state.kind !== "ready") return;

    setBusy(`privacy:${field}`);
    setFeedback(null);

    try {
      const result = await patchData<CustomerPrivacySettings>(
        "/api/storefront/account/settings/privacy",
        { [field]: value },
      );

      updateSettingData("privacy", result.data);
      showSuccess("Privacy preference updated.");
    } catch (caught) {
      showError(caught, "Unable to update privacy settings.");
    } finally {
      setBusy("");
    }
  }

  async function signOut() {
    if (busy) return;

    setBusy("logout");
    setFeedback(null);

    try {
      await fetch("/api/storefront/auth/logout", {
        method: "POST",
      });
    } finally {
      window.dispatchEvent(new Event("customer-auth-changed"));
      router.replace("/account/sign-in");
      router.refresh();
    }
  }

  if (state.kind === "loading") {
    return (
      <main className={styles.page} aria-busy="true">
        <div className={styles.shell}>
          <div className={styles.loadingHeader} />
          <div className={styles.loadingLayout}>
            <div className={styles.loadingRail} />
            <div className={styles.loadingSections}>
              <div />
              <div />
              <div />
            </div>
          </div>
        </div>
      </main>
    );
  }

  if (state.kind === "error") {
    return (
      <main className={styles.page}>
        <section className={styles.errorState} role="alert">
          <Icon name="account" size={26} />
          <h1>Settings are temporarily unavailable.</h1>
          <p>{state.message}</p>
          <button type="button" onClick={() => void load()}>
            Try again
          </button>
        </section>
      </main>
    );
  }

  const { customer, preferences, notifications, privacy } = state.data;

  return (
    <main className={styles.page}>
      <div className={styles.lightField} aria-hidden="true" />

      <div className={styles.shell}>
        <Link href="/account" className={styles.backLink}>
          <Icon name="chevronLeft" size={14} />
          Account
        </Link>

        <header className={styles.pageHeader}>
          <div>
            <h1>Profile & settings</h1>
            <p>
              Manage your account details, display preferences, notifications,
              privacy and security.
            </p>
          </div>
          <span className={styles.accountState}>Account active</span>
        </header>

        <div className={styles.layout}>
          <aside className={styles.sectionRail} aria-label="Settings sections">
            <div className={styles.identityMini}>
              <Avatar customer={customer} compact />
              <div>
                <strong>{customer.full_name}</strong>
                <span>{customer.phone}</span>
              </div>
            </div>

            <nav>
              {sections.map((section) => (
                <a key={section.href} href={section.href}>
                  {section.label}
                </a>
              ))}
            </nav>

            <Link href="/account/security" className={styles.securityLink}>
              <Icon name="secureCheckout" size={16} />
              Security & sessions
              <Icon name="chevronRight" size={13} />
            </Link>
          </aside>

          <div className={styles.content}>
            {feedback ? (
              <div
                className={`${styles.feedback} ${
                  feedback.kind === "error"
                    ? styles.feedbackError
                    : styles.feedbackSuccess
                }`}
                role={feedback.kind === "error" ? "alert" : "status"}
              >
                {feedback.message}
              </div>
            ) : null}

            <section className={styles.section} id="profile">
              <div className={styles.sectionIntro}>
                <div>
                  <h2>Your profile</h2>
                  <p>Information used for your account, orders and support.</p>
                </div>
              </div>

              <div className={styles.profileGrid}>
                <div className={styles.photoPanel}>
                  <Avatar customer={customer} />

                  <div className={styles.photoCopy}>
                    <strong>Profile photo</strong>
                    <span>JPEG, PNG or WebP. Maximum 5 MB.</span>
                    <div className={styles.inlineActions}>
                      <button
                        type="button"
                        className={styles.secondaryButton}
                        disabled={Boolean(busy)}
                        onClick={() => avatarInputRef.current?.click()}
                      >
                        {busy === "avatar" ? "Updating…" : "Change photo"}
                      </button>

                      {customer.avatar_url ? (
                        <button
                          type="button"
                          className={styles.textButton}
                          disabled={Boolean(busy)}
                          onClick={() => void removeAvatar()}
                        >
                          Remove
                        </button>
                      ) : null}
                    </div>
                  </div>

                  <input
                    ref={avatarInputRef}
                    className={styles.fileInput}
                    type="file"
                    accept="image/jpeg,image/png,image/webp"
                    onChange={(event: ChangeEvent<HTMLInputElement>) => {
                      const file = event.target.files?.[0];
                      if (file) void uploadAvatar(file);
                    }}
                  />
                </div>

                <form className={styles.form} onSubmit={saveProfile}>
                  <label className={styles.field}>
                    <span>Full name</span>
                    <input
                      value={profileForm.full_name}
                      autoComplete="name"
                      onChange={(event) =>
                        setProfileForm((current) => ({
                          ...current,
                          full_name: event.target.value,
                        }))
                      }
                    />
                  </label>

                  <label className={styles.field}>
                    <span>Email</span>
                    <input
                      value={profileForm.email}
                      type="email"
                      autoComplete="email"
                      placeholder="name@example.com"
                      onChange={(event) =>
                        setProfileForm((current) => ({
                          ...current,
                          email: event.target.value,
                        }))
                      }
                    />
                  </label>

                  <label className={styles.field}>
                    <span>Phone</span>
                    <input
                      value={profileForm.phone}
                      type="tel"
                      autoComplete="tel"
                      onChange={(event) =>
                        setProfileForm((current) => ({
                          ...current,
                          phone: event.target.value,
                        }))
                      }
                    />
                  </label>

                  <div className={styles.formActions}>
                    <button
                      type="submit"
                      className={styles.primaryButton}
                      disabled={Boolean(busy)}
                    >
                      {busy === "profile" ? "Saving…" : "Save profile"}
                    </button>
                  </div>
                </form>
              </div>
            </section>

            <section className={styles.section} id="display">
              <div className={styles.sectionIntro}>
                <div>
                  <h2>Language & currency</h2>
                  <p>Choose how account information is displayed.</p>
                </div>
              </div>

              <form
                className={styles.preferenceBar}
                onSubmit={saveDisplayPreferences}
              >
                <label className={styles.field}>
                  <span>Site language</span>
                  <select name="locale" defaultValue={preferences.locale || "en"}>
                    <option value="en">English</option>
                    <option value="bn">বাংলা</option>
                  </select>
                </label>

                <label className={styles.field}>
                  <span>Currency</span>
                  <input
                    value={`${preferences.currency || "BDT"} — Bangladeshi taka`}
                    disabled
                    readOnly
                  />
                  <small>Currency conversion is not applied automatically.</small>
                </label>

                <button
                  type="submit"
                  className={styles.secondaryButton}
                  disabled={Boolean(busy)}
                >
                  {busy === "preferences" ? "Saving…" : "Save changes"}
                </button>
              </form>
            </section>

            <section className={styles.section} id="notifications">
              <div className={styles.sectionIntro}>
                <div>
                  <h2>Notifications</h2>
                  <p>Choose which account updates should also arrive by email.</p>
                </div>
              </div>

              <div className={styles.toggleList}>
                <ToggleRow
                  title="Order updates"
                  description="Confirmation and important changes to your orders."
                  checked={notifications.order_updates_email}
                  disabled={Boolean(busy)}
                  onChange={(value) =>
                    void updateNotification("order_updates_email", value)
                  }
                />
                <ToggleRow
                  title="Delivery updates"
                  description="Shipping and delivery progress when an order is on the way."
                  checked={notifications.delivery_updates_email}
                  disabled={Boolean(busy)}
                  onChange={(value) =>
                    void updateNotification("delivery_updates_email", value)
                  }
                />
                <ToggleRow
                  title="Support updates"
                  description="Replies and important changes to your support cases."
                  checked={notifications.support_updates_email}
                  disabled={Boolean(busy)}
                  onChange={(value) =>
                    void updateNotification("support_updates_email", value)
                  }
                />
                <ToggleRow
                  title="Offers and promotions"
                  description="Occasional commercial offers from ene dei."
                  checked={notifications.promotions_email}
                  disabled={Boolean(busy)}
                  onChange={(value) =>
                    void updateNotification("promotions_email", value)
                  }
                />
                <ToggleRow
                  title="Product recommendations"
                  description="Product suggestions based on activity when personalization is enabled."
                  checked={notifications.recommendations_email}
                  disabled={Boolean(busy)}
                  onChange={(value) =>
                    void updateNotification("recommendations_email", value)
                  }
                />
              </div>

              <div className={styles.availabilityNote}>
                <Icon name="support" size={16} />
                <span>
                  SMS controls are not shown because SMS delivery is not enabled.
                </span>
              </div>
            </section>

            <section className={styles.section} id="privacy">
              <div className={styles.sectionIntro}>
                <div>
                  <h2>Privacy</h2>
                  <p>Control which browsing signals can be kept for your account.</p>
                </div>
              </div>

              <div className={styles.toggleList}>
                <ToggleRow
                  title="Personalization"
                  description="Allow browsing activity to improve product discovery and recommendations."
                  checked={privacy.personalization_enabled}
                  disabled={Boolean(busy)}
                  onChange={(value) =>
                    void updatePrivacy("personalization_enabled", value)
                  }
                />
                <ToggleRow
                  title="Search history"
                  description="Keep recent searches so they are easier to return to."
                  checked={privacy.search_history_enabled}
                  disabled={Boolean(busy)}
                  onChange={(value) =>
                    void updatePrivacy("search_history_enabled", value)
                  }
                />
                <ToggleRow
                  title="Recently viewed products"
                  description="Keep recently viewed products available across your account."
                  checked={privacy.recently_viewed_enabled}
                  disabled={Boolean(busy)}
                  onChange={(value) =>
                    void updatePrivacy("recently_viewed_enabled", value)
                  }
                />
              </div>
            </section>

            <section className={styles.section} id="access">
              <div className={styles.sectionIntro}>
                <div>
                  <h2>Account access</h2>
                  <p>Manage account security or sign out of this device.</p>
                </div>
              </div>

              <div className={styles.accessRows}>
                <Link href="/account/security" className={styles.accessLink}>
                  <span className={styles.accessIcon}>
                    <Icon name="secureCheckout" size={18} />
                  </span>
                  <span>
                    <strong>Security & sessions</strong>
                    <small>Review your password, sessions and security controls.</small>
                  </span>
                  <Icon name="chevronRight" size={14} />
                </Link>

                <div className={styles.signOutRow}>
                  <div>
                    <strong>Sign out</strong>
                    <span>End your current ene dei session on this device.</span>
                  </div>
                  <button
                    type="button"
                    className={styles.signOutButton}
                    disabled={Boolean(busy)}
                    onClick={() => void signOut()}
                  >
                    <Icon name="arrowLeft" size={15} />
                    {busy === "logout" ? "Signing out…" : "Sign out"}
                  </button>
                </div>
              </div>
            </section>
          </div>
        </div>
      </div>
    </main>
  );
}

function Avatar({
  customer,
  compact = false,
}: {
  customer: AccountCustomer;
  compact?: boolean;
}) {
  return (
    <span
      className={`${styles.avatar} ${compact ? styles.avatarCompact : ""}`}
      aria-hidden="true"
    >
      {customer.avatar_url ? (
        // Signed private avatar URLs are supplied by the commerce backend.
        // eslint-disable-next-line @next/next/no-img-element
        <img src={customer.avatar_url} alt="" />
      ) : (
        <span>{initials(customer.full_name)}</span>
      )}
    </span>
  );
}

function ToggleRow({
  title,
  description,
  checked,
  disabled,
  onChange,
}: {
  title: string;
  description: string;
  checked: boolean;
  disabled: boolean;
  onChange: (value: boolean) => void;
}) {
  return (
    <label className={styles.toggleRow}>
      <span className={styles.toggleCopy}>
        <strong>{title}</strong>
        <span>{description}</span>
      </span>

      <span className={styles.switchControl}>
        <input
          type="checkbox"
          checked={checked}
          disabled={disabled}
          onChange={(event) => onChange(event.target.checked)}
        />
        <span aria-hidden="true" />
      </span>
    </label>
  );
}
