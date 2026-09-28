// Location: src/components/account/components/AccountAuthClient.tsx
"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { type FormEvent, useState } from "react";

import { Icon } from "@/components/ui/Icon";

import styles from "../css/Auth.module.css";

type Mode = "sign-in" | "sign-up";

type Props = {
  mode: Mode;
  returnTo?: string;
};

type ErrorPayload = {
  error?: {
    code?: string;
    message?: string;
  };
};

function normalizeBangladeshPhone(value: string) {
  let digits = value.replace(/\D/g, "");

  if (digits.startsWith("880")) digits = digits.slice(3);
  if (digits.startsWith("0")) digits = digits.slice(1);

  if (!/^1[3-9]\d{8}$/.test(digits)) {
    throw new Error("Enter a valid Bangladesh mobile number.");
  }

  return `+880${digits}`;
}

async function readError(response: Response, fallback = "Unable to continue.") {
  try {
    const payload = (await response.json()) as ErrorPayload;
    return payload.error?.message?.trim() || fallback;
  } catch {
    return fallback;
  }
}

export function AccountAuthClient({ mode, returnTo = "/account" }: Props) {
  const router = useRouter();
  const isSignUp = mode === "sign-up";

  const [fullName, setFullName] = useState("");
  const [phone, setPhone] = useState("");
  const [password, setPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [showPassword, setShowPassword] = useState(false);
  const [acceptedTerms, setAcceptedTerms] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  const guestHref = returnTo === "/checkout" ? "/cart" : "/";

  async function finishAuthentication() {
    if (returnTo === "/checkout") {
      const checkoutResponse = await fetch("/api/storefront/checkout", {
        method: "POST",
      });

      if (checkoutResponse.ok) {
        router.replace("/checkout");
        router.refresh();
        return;
      }

      if (checkoutResponse.status === 404) {
        router.replace("/cart");
        router.refresh();
        return;
      }

      throw new Error(
        await readError(
          checkoutResponse,
          "You are signed in, but checkout could not be prepared.",
        ),
      );
    }

    router.replace(returnTo);
    router.refresh();
  }

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (busy) return;

    setError("");

    if (isSignUp && fullName.trim().length < 2) {
      setError("Enter your full name.");
      return;
    }

    let normalizedPhone = "";

    try {
      normalizedPhone = normalizeBangladeshPhone(phone);
    } catch (caught) {
      setError(
        caught instanceof Error
          ? caught.message
          : "Enter a valid Bangladesh mobile number.",
      );
      return;
    }

    if (password.length < 8) {
      setError("Password must contain at least 8 characters.");
      return;
    }

    if (isSignUp && password !== confirmPassword) {
      setError("Passwords do not match.");
      return;
    }

    if (isSignUp && !acceptedTerms) {
      setError("Please accept the Terms and Privacy Policy.");
      return;
    }

    setBusy(true);

    try {
      const response = await fetch(
        isSignUp ? "/api/storefront/auth/register" : "/api/storefront/auth/login",
        {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify(
            isSignUp
              ? {
                  full_name: fullName.trim(),
                  phone: normalizedPhone,
                  password,
                }
              : { phone: normalizedPhone, password },
          ),
        },
      );

      if (!response.ok) {
        throw new Error(await readError(response));
      }

      await finishAuthentication();
    } catch (caught) {
      setBusy(false);
      setError(caught instanceof Error ? caught.message : "Unable to continue.");
    }
  }

  return (
    <main className={styles.page} data-mode={mode}>
      <div className={styles.ambient} aria-hidden="true" />

      <section className={styles.shell} aria-labelledby="account-auth-title">
        <form className={styles.glassCard} onSubmit={submit} noValidate>
          <div className={styles.heroIcon} aria-hidden="true">
            <Icon name={isSignUp ? "account" : "secureCheckout"} size={23} />
          </div>

          <div className={styles.heading}>
            <h1 id="account-auth-title">
              {isSignUp ? "Create your account" : "Welcome back"}
            </h1>
            <p>
              {isSignUp
                ? "Set up your Ene Dei account to order, save addresses and manage your purchases."
                : "Sign in with your mobile number to access your account and continue checkout."}
            </p>
          </div>

          <div className={styles.formBody}>
            {isSignUp ? (
              <label className={styles.field}>
                <span>
                  Full name <em>*</em>
                </span>
                <input
                  type="text"
                  autoComplete="name"
                  value={fullName}
                  disabled={busy}
                  placeholder="Your full name"
                  onChange={(event) => setFullName(event.target.value)}
                />
              </label>
            ) : null}

            <label className={styles.field}>
              <span>
                Mobile number <em>*</em>
              </span>
              <div className={styles.phoneInput}>
                <span className={styles.countryCode} aria-label="Bangladesh country code">
                  <span className={styles.countryDot} aria-hidden="true" />
                  +880
                </span>
                <input
                  type="tel"
                  inputMode="tel"
                  autoComplete="tel"
                  value={phone}
                  disabled={busy}
                  placeholder="1712 345 678"
                  aria-describedby="phone-help"
                  onChange={(event) => setPhone(event.target.value)}
                />
              </div>
              <small id="phone-help">We use this number for secure authentication.</small>
            </label>

            <label className={styles.field}>
              <span>
                Password <em>*</em>
              </span>
              <div className={styles.passwordInput}>
                <input
                  type={showPassword ? "text" : "password"}
                  autoComplete={isSignUp ? "new-password" : "current-password"}
                  value={password}
                  disabled={busy}
                  placeholder={isSignUp ? "At least 8 characters" : "Your password"}
                  onChange={(event) => setPassword(event.target.value)}
                />
                <button
                  type="button"
                  className={styles.showPassword}
                  aria-label={showPassword ? "Hide password" : "Show password"}
                  onClick={() => setShowPassword((current) => !current)}
                >
                  {showPassword ? "Hide" : "Show"}
                </button>
              </div>
            </label>

            {isSignUp ? (
              <label className={styles.field}>
                <span>
                  Confirm password <em>*</em>
                </span>
                <div className={styles.passwordInput}>
                  <input
                    type={showPassword ? "text" : "password"}
                    autoComplete="new-password"
                    value={confirmPassword}
                    disabled={busy}
                    placeholder="Re-enter your password"
                    onChange={(event) => setConfirmPassword(event.target.value)}
                  />
                  <button
                    type="button"
                    className={styles.showPassword}
                    aria-label={showPassword ? "Hide password" : "Show password"}
                    onClick={() => setShowPassword((current) => !current)}
                  >
                    {showPassword ? "Hide" : "Show"}
                  </button>
                </div>
                <small>Use at least 8 characters.</small>
              </label>
            ) : null}

            {isSignUp ? (
              <label className={styles.terms}>
                <input
                  type="checkbox"
                  checked={acceptedTerms}
                  disabled={busy}
                  onChange={(event) => setAcceptedTerms(event.target.checked)}
                />
                <span>
                  I agree to the <Link href="/terms">Terms</Link> and{" "}
                  <Link href="/privacy">Privacy Policy</Link>.
                </span>
              </label>
            ) : null}

            {error ? (
              <div className={styles.error} role="alert">
                {error}
              </div>
            ) : null}

            <button className={styles.primaryButton} type="submit" disabled={busy}>
              <span>{busy ? "Please wait…" : isSignUp ? "Create account" : "Sign in"}</span>
              <span aria-hidden="true">→</span>
            </button>

            {!isSignUp ? (
              <>
                <div className={styles.divider} aria-hidden="true">
                  <span />
                  <small>or</small>
                  <span />
                </div>

                <Link className={styles.guestButton} href={guestHref}>
                  <Icon name="cart" size={17} />
                  <span>Browse as guest</span>
                  <span aria-hidden="true">›</span>
                </Link>
              </>
            ) : null}

            <div className={styles.switchAccount}>
              {isSignUp ? (
                <>
                  <span>Already have an account?</span>
                  <Link href={`/account/sign-in?next=${encodeURIComponent(returnTo)}`}>
                    Sign in
                  </Link>
                </>
              ) : (
                <>
                  <span>New to Ene Dei?</span>
                  <Link href={`/account/sign-up?next=${encodeURIComponent(returnTo)}`}>
                    Create account
                  </Link>
                </>
              )}
            </div>
          </div>

          <div className={styles.accountNote}>
            <span className={styles.noteIcon} aria-hidden="true">
              <Icon name={isSignUp ? "account" : "secureCheckout"} size={18} />
            </span>
            <span>
              <strong>
                {isSignUp
                  ? "One account for your entire buying journey."
                  : "Secure phone-based access."}
              </strong>
              {isSignUp
                ? "Manage orders, addresses, wishlists and support from one place."
                : "Browsing and cart stay open without an account; checkout requires sign in."}
            </span>
          </div>
        </form>
      </section>
    </main>
  );
}
