// Location: src/components/account/components/AddressesClient.tsx
"use client";

import Link from "next/link";
import { type FormEvent, useCallback, useEffect, useState } from "react";

import { Icon } from "@/components/ui/Icon";
import type { AccountAddress, AccountDataResponse } from "@/lib/api/contracts/account";

import { AccountSectionHeader } from "./AccountSectionHeader";
import styles from "../css/AccountManagement.module.css";

type FormState = {
  label: string;
  recipient_name: string;
  phone: string;
  address_line1: string;
  address_line2: string;
  city: string;
  area: string;
  postal_code: string;
  is_default: boolean;
};

const emptyForm: FormState = {
  label: "",
  recipient_name: "",
  phone: "",
  address_line1: "",
  address_line2: "",
  city: "",
  area: "",
  postal_code: "",
  is_default: false,
};

async function readError(response: Response, fallback: string) {
  try {
    const payload = (await response.json()) as { error?: { message?: string } };
    return payload.error?.message?.trim() || fallback;
  } catch {
    return fallback;
  }
}

export function AddressesClient() {
  const [addresses, setAddresses] = useState<AccountAddress[]>([]);
  const [form, setForm] = useState<FormState>(emptyForm);
  const [loading, setLoading] = useState(true);
  const [busy, setBusy] = useState("");
  const [editingID, setEditingID] = useState("");
  const [error, setError] = useState("");
  const [signedOut, setSignedOut] = useState(false);

  const load = useCallback(async () => {
    setLoading(true);
    setError("");

    try {
      const response = await fetch("/api/storefront/account/addresses", { cache: "no-store" });
      if (response.status === 401) {
        setSignedOut(true);
        return;
      }
      if (!response.ok) throw new Error(await readError(response, "Unable to load addresses."));
      const payload = (await response.json()) as AccountDataResponse<AccountAddress[]>;
      setAddresses(payload.data ?? []);
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : "Unable to load addresses.");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    let cancelled = false;

    void Promise.resolve().then(() => {
      if (!cancelled) {
        void load();
      }
    });

    return () => {
      cancelled = true;
    };
  }, [load]);

  function updateField<K extends keyof FormState>(key: K, value: FormState[K]) {
    setForm((current) => ({ ...current, [key]: value }));
    setError("");
  }

  async function saveAddress(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (busy) return;

    if (!form.label.trim() || !form.recipient_name.trim() || !form.phone.trim() || !form.address_line1.trim() || !form.city.trim() || !form.area.trim()) {
      setError("Complete the required address fields.");
      return;
    }

    setBusy("create");
    setError("");

    try {
      const endpoint = editingID
        ? `/api/storefront/account/addresses/${encodeURIComponent(editingID)}`
        : "/api/storefront/account/addresses";

      const response = await fetch(endpoint, {
        method: editingID ? "PATCH" : "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(form),
      });
      if (!response.ok) throw new Error(await readError(response, "Unable to save address."));
      setForm(emptyForm);
      setEditingID("");
      await load();
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : "Unable to save address.");
    } finally {
      setBusy("");
    }
  }


  function startEdit(address: AccountAddress) {
    if (busy) return;

    setEditingID(address.id);
    setForm({
      label: address.label ?? "",
      recipient_name: address.recipient_name ?? "",
      phone: address.phone ?? "",
      address_line1: address.address_line1 ?? "",
      address_line2: address.address_line2 ?? "",
      city: address.city ?? "",
      area: address.area ?? "",
      postal_code: address.postal_code ?? "",
      is_default: Boolean(address.is_default),
    });
    setError("");
  }

  function cancelEdit() {
    if (busy) return;
    setEditingID("");
    setForm(emptyForm);
    setError("");
  }

  async function removeAddress(id: string) {
    if (busy) return;
    setBusy(id);
    setError("");

    try {
      const response = await fetch(`/api/storefront/account/addresses/${encodeURIComponent(id)}`, {
        method: "DELETE",
      });
      if (!response.ok) throw new Error(await readError(response, "Unable to remove address."));
      await load();
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : "Unable to remove address.");
    } finally {
      setBusy("");
    }
  }

  if (signedOut) {
    return (
      <main className={styles.page}>
        <section className={styles.state}>
          <Icon name="address" size={24} />
          <h1>Sign in to manage addresses.</h1>
          <Link className={styles.primary} href="/account/sign-in?next=%2Faccount%2Faddresses">Sign in</Link>
        </section>
      </main>
    );
  }

  return (
    <main className={styles.page}>
      <div className={styles.shell}>
        <AccountSectionHeader
          eyebrow="Saved addresses"
          title="Delivery details, ready when you checkout."
          description="Keep your regular delivery destinations here. Checkout remains authoritative and will validate the selected address with the order."
        />

        <div className={styles.grid}>
          <section className={`${styles.card} ${styles.span7}`}>
            <div className={styles.cardHeader}>
              <div><h2>Your addresses</h2><p>{addresses.length.toLocaleString()} saved</p></div>
            </div>
            {loading ? <p className={styles.message}>Loading addresses…</p> : null}
            {!loading && addresses.length === 0 ? <div className={styles.empty}>No saved address yet.</div> : null}
            <div className={styles.list}>
              {addresses.map((address) => (
                <article className={styles.row} key={address.id}>
                  <div className={styles.rowMain}>
                    <strong>{address.label || "Address"} {address.is_default ? <span className={styles.defaultBadge}>Default</span> : null}</strong>
                    <span>{address.recipient_name} · {address.phone}</span>
                    <p>{address.address_line1}{address.address_line2 ? `, ${address.address_line2}` : ""}, {address.area}, {address.city}{address.postal_code ? ` ${address.postal_code}` : ""}</p>
                  </div>
                  <div className={styles.rowActions}>
                    <button
                      className={styles.secondary}
                      type="button"
                      disabled={Boolean(busy)}
                      onClick={() => startEdit(address)}
                    >
                      Edit
                    </button>
                    <button className={styles.danger} type="button" disabled={Boolean(busy)} onClick={() => void removeAddress(address.id)}>
                      {busy === address.id ? "Removing…" : "Remove"}
                    </button>
                  </div>
                </article>
              ))}
            </div>
          </section>

          <section className={`${styles.card} ${styles.span5}`}>
            <div className={styles.cardHeader}>
              <div><h2>{editingID ? "Edit address" : "Add address"}</h2><p>{editingID ? "Update this saved delivery destination." : "Create another delivery destination."}</p></div>
            </div>
            <form className={styles.form} onSubmit={saveAddress}>
              <label className={styles.field}><span>Label</span><input value={form.label} placeholder="Home, Office, Warehouse" onChange={(e) => updateField("label", e.target.value)} /></label>
              <label className={styles.field}><span>Recipient</span><input value={form.recipient_name} autoComplete="name" onChange={(e) => updateField("recipient_name", e.target.value)} /></label>
              <label className={styles.field}><span>Phone</span><input value={form.phone} type="tel" autoComplete="tel" onChange={(e) => updateField("phone", e.target.value)} /></label>
              <label className={styles.field}><span>City / district</span><input value={form.city} onChange={(e) => updateField("city", e.target.value)} /></label>
              <label className={`${styles.field} ${styles.fieldWide}`}><span>Address line 1</span><input value={form.address_line1} autoComplete="address-line1" onChange={(e) => updateField("address_line1", e.target.value)} /></label>
              <label className={`${styles.field} ${styles.fieldWide}`}><span>Address line 2</span><input value={form.address_line2} autoComplete="address-line2" onChange={(e) => updateField("address_line2", e.target.value)} /></label>
              <label className={styles.field}><span>Area / thana</span><input value={form.area} onChange={(e) => updateField("area", e.target.value)} /></label>
              <label className={styles.field}><span>Postal code</span><input value={form.postal_code} autoComplete="postal-code" onChange={(e) => updateField("postal_code", e.target.value)} /></label>
              <label className={styles.checkbox}><input type="checkbox" checked={form.is_default} onChange={(e) => updateField("is_default", e.target.checked)} /> Make this my default address</label>
              {error ? <p className={`${styles.message} ${styles.error}`} role="alert">{error}</p> : null}
              <div className={styles.actions}>
                <button className={styles.primary} type="submit" disabled={Boolean(busy)}>
                  {busy === "create" ? "Saving…" : editingID ? "Update address" : "Save address"}
                </button>
                {editingID ? (
                  <button className={styles.secondary} type="button" disabled={Boolean(busy)} onClick={cancelEdit}>
                    Cancel
                  </button>
                ) : null}
              </div>
            </form>
          </section>
        </div>
      </div>
    </main>
  );
}
