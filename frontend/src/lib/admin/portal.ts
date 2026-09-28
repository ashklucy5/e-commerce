import { randomBytes } from "node:crypto";
import {
  readFile,
  writeFile,
} from "node:fs/promises";
import { join } from "node:path";

const PORTAL_FILE_NAME =
  ".admin-portal-slug";

const PORTAL_PATTERN =
  /^[A-Za-z0-9_-]{32,128}$/;

let cachedPortalSlug:
  | string
  | null = null;

function validatePortalSlug(
  value: string,
): string {
  const slug = value.trim();

  if (!PORTAL_PATTERN.test(slug)) {
    throw new Error(
      "Admin portal slug must be a URL-safe random value between 32 and 128 characters.",
    );
  }

  return slug;
}

function createPortalSlug(): string {
  return randomBytes(24).toString(
    "base64url",
  );
}

function getPortalFilePath(): string {
  return join(
    process.cwd(),
    PORTAL_FILE_NAME,
  );
}

function errorCode(
  error: unknown,
): string | undefined {
  if (
    typeof error === "object" &&
    error !== null &&
    "code" in error
  ) {
    return String(
      (
        error as {
          code?: unknown;
        }
      ).code,
    );
  }

  return undefined;
}

async function readPersistedPortalSlug(
  filePath: string,
): Promise<string> {
  const value = await readFile(
    filePath,
    "utf8",
  );

  return validatePortalSlug(value);
}

export async function getAdminPortalSlug(): Promise<string> {
  if (cachedPortalSlug) {
    return cachedPortalSlug;
  }

  const configuredSlug =
    process.env.ADMIN_PORTAL_SLUG?.trim();

  if (configuredSlug) {
    cachedPortalSlug =
      validatePortalSlug(
        configuredSlug,
      );

    return cachedPortalSlug;
  }

  if (
    process.env.NODE_ENV ===
    "production"
  ) {
    throw new Error(
      "ADMIN_PORTAL_SLUG must be configured in production.",
    );
  }

  const filePath =
    getPortalFilePath();

  try {
    cachedPortalSlug =
      await readPersistedPortalSlug(
        filePath,
      );

    return cachedPortalSlug;
  } catch (error) {
    if (
      errorCode(error) !== "ENOENT"
    ) {
      throw error;
    }
  }

  const generatedSlug =
    createPortalSlug();

  try {
    await writeFile(
      filePath,
      `${generatedSlug}\n`,
      {
        encoding: "utf8",
        flag: "wx",
        mode: 0o600,
      },
    );

    cachedPortalSlug =
      generatedSlug;

    console.info(
      `[admin] Private development portal: /${generatedSlug}/login`,
    );

    return generatedSlug;
  } catch (error) {
    if (
      errorCode(error) === "EEXIST"
    ) {
      cachedPortalSlug =
        await readPersistedPortalSlug(
          filePath,
        );

      return cachedPortalSlug;
    }

    throw error;
  }
}