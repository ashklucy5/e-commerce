"use client";

import { useState } from "react";

import {
  useRouter,
} from "next/navigation";

import {
  adminFetch,
  AdminRequestError,
} from "@/lib/admin/api";

import type {
  AdminLoginResponse,
  AdminMfaEnrollment,
  AdminMfaEnrollmentResponse,
  AdminSessionResponse,
} from "@/lib/admin/types";

import AdminAuthFrame from "./AdminAuthFrame";
import AdminEnrollmentForm from "./AdminEnrollmentForm";
import AdminLoginForm from "./AdminLoginForm";
import AdminMfaForm from "./AdminMfaForm";
import AdminRecoveryCodes from "./AdminRecoveryCodes";

type AuthStage =
  | "login"
  | "verify"
  | "enroll"
  | "recovery";

type AdminAuthFlowProps = {
  portal: string;
};

export default function AdminAuthFlow({
  portal,
}: AdminAuthFlowProps) {
  const router = useRouter();

  const [stage, setStage] =
    useState<AuthStage>("login");

  const [
    challengeToken,
    setChallengeToken,
  ] = useState("");

  const [
    enrollment,
    setEnrollment,
  ] =
    useState<AdminMfaEnrollment | null>(
      null,
    );

  const [
    recoveryCodes,
    setRecoveryCodes,
  ] = useState<string[]>([]);

  const [busy, setBusy] =
    useState(false);

  const [error, setError] =
    useState("");

  function openPortal() {
    router.replace(`/${portal}`);
    router.refresh();
  }

  function resetLogin(
    message = "",
  ) {
    setStage("login");
    setChallengeToken("");
    setEnrollment(null);
    setRecoveryCodes([]);
    setError(message);
  }

  function handleRequestError(
    value: unknown,
  ) {
    if (
      value instanceof
      AdminRequestError
    ) {
      if (
        value.code ===
        "INVALID_ADMIN_CHALLENGE"
      ) {
        resetLogin(
          "Your sign-in challenge expired. Please sign in again.",
        );

        return;
      }

      setError(value.message);

      return;
    }

    setError(
      "The operations service could not complete the request.",
    );
  }

  async function handleLogin(
    identifier: string,
    password: string,
  ) {
    setBusy(true);
    setError("");

    try {
      const response =
        await adminFetch<AdminLoginResponse>(
          "/auth/login",
          {
            method: "POST",
            body: JSON.stringify({
              identifier,
              password,
            }),
          },
        );

      const challenge =
        response.data;

      setChallengeToken(
        challenge.challenge_token,
      );

      if (
        !challenge.mfa_enrollment_required
      ) {
        setStage("verify");
        return;
      }

      const enrollmentResponse =
        await adminFetch<AdminMfaEnrollmentResponse>(
          "/auth/mfa/enroll",
          {
            method: "POST",
            body: JSON.stringify({
              challenge_token:
                challenge.challenge_token,
              label:
                "Ene dei Operations",
            }),
          },
        );

      setEnrollment(
        enrollmentResponse.data
          .enrollment,
      );

      setStage("enroll");
    } catch (value) {
      handleRequestError(value);
    } finally {
      setBusy(false);
    }
  }

  async function handleMfaVerify(
    code: string,
  ) {
    setBusy(true);
    setError("");

    try {
      await adminFetch<AdminSessionResponse>(
        "/auth/mfa/verify",
        {
          method: "POST",
          body: JSON.stringify({
            challenge_token:
              challengeToken,
            method: "totp",
            code,
          }),
        },
      );

      openPortal();
    } catch (value) {
      handleRequestError(value);
    } finally {
      setBusy(false);
    }
  }

  async function handleEnrollmentConfirm(
    code: string,
  ) {
    setBusy(true);
    setError("");

    try {
      const response =
        await adminFetch<AdminSessionResponse>(
          "/auth/mfa/enroll/confirm",
          {
            method: "POST",
            body: JSON.stringify({
              challenge_token:
                challengeToken,
              code,
            }),
          },
        );

      const codes =
        response.data.recovery_codes ??
        [];

      if (codes.length > 0) {
        setRecoveryCodes(codes);
        setStage("recovery");
        return;
      }

      openPortal();
    } catch (value) {
      handleRequestError(value);
    } finally {
      setBusy(false);
    }
  }

  if (
    stage === "enroll" &&
    enrollment
  ) {
    return (
      <AdminAuthFrame
        eyebrow="First-time security"
        title="Protect this account."
        description="Set up multi-factor authentication before opening the operations console."
      >
        <AdminEnrollmentForm
          enrollment={enrollment}
          busy={busy}
          error={error}
          onSubmit={
            handleEnrollmentConfirm
          }
          onBack={() =>
            resetLogin()
          }
        />
      </AdminAuthFrame>
    );
  }

  if (stage === "verify") {
    return (
      <AdminAuthFrame
        eyebrow="Identity verification"
        title="One more step."
        description="Enter the current code from the authenticator registered to this Admin account."
      >
        <AdminMfaForm
          busy={busy}
          error={error}
          onSubmit={handleMfaVerify}
          onBack={() =>
            resetLogin()
          }
        />
      </AdminAuthFrame>
    );
  }

  if (stage === "recovery") {
    return (
      <AdminAuthFrame
        eyebrow="Account recovery"
        title="Save your recovery codes."
        description="These codes provide emergency access if the authenticator becomes unavailable."
      >
        <AdminRecoveryCodes
          codes={recoveryCodes}
          onContinue={openPortal}
        />
      </AdminAuthFrame>
    );
  }

  return (
    <AdminAuthFrame
      eyebrow="Private operations"
      title="Welcome back."
      description="Authenticate to manage products, customer orders, fulfillment, inventory and commerce operations."
    >
      <AdminLoginForm
        busy={busy}
        error={error}
        onSubmit={handleLogin}
      />
    </AdminAuthFrame>
  );
}