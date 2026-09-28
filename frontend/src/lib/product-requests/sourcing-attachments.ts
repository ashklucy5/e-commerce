export type SourcingAttachment = {
  id: string;
  kind: "image" | "link";
  name: string;
  url: string;
  mime_type?: string;
  size?: number;
};

export const MAX_SOURCING_ATTACHMENTS = 4;
export const MAX_MESSAGE_ATTACHMENT_JSON_BYTES = 118 * 1024;
export const MAX_OFFER_ATTACHMENT_JSON_BYTES = 60 * 1024;

const MAX_SOURCE_IMAGE_BYTES = 8 * 1024 * 1024;
const MESSAGE_IMAGE_DATA_URL_LENGTH = 28_000;
const OFFER_IMAGE_DATA_URL_LENGTH = 12_000;

type UnknownRecord = Record<string, unknown>;

function asRecord(value: unknown): UnknownRecord | null {
  if (!value || typeof value !== "object" || Array.isArray(value)) return null;
  return value as UnknownRecord;
}

export function makeAttachmentID(): string {
  if (typeof crypto !== "undefined" && typeof crypto.randomUUID === "function") {
    return crypto.randomUUID();
  }

  return `${Date.now()}-${Math.random().toString(36).slice(2)}`;
}

export function normaliseURL(value: string): string {
  const trimmed = value.trim();
  if (!trimmed) return "";

  const candidate = /^https?:\/\//i.test(trimmed) ? trimmed : `https://${trimmed}`;

  try {
    const parsed = new URL(candidate);
    if (parsed.protocol !== "http:" && parsed.protocol !== "https:") return "";
    return parsed.toString();
  } catch {
    return "";
  }
}

export function parseAttachments(input: unknown): SourcingAttachment[] {
  if (!input) return [];

  let value = input;

  if (typeof value === "string") {
    const trimmed = value.trim();
    if (!trimmed) return [];

    if (trimmed.startsWith("[")) {
      try {
        value = JSON.parse(trimmed);
      } catch {
        return [];
      }
    } else if (trimmed.startsWith("data:image/")) {
      return [
        {
          id: makeAttachmentID(),
          kind: "image",
          name: "Reference image",
          url: trimmed,
        },
      ];
    } else {
      const url = normaliseURL(trimmed);
      return url
        ? [
            {
              id: makeAttachmentID(),
              kind: "link",
              name: url,
              url,
            },
          ]
        : [];
    }
  }

  if (!Array.isArray(value)) return [];

  const result: SourcingAttachment[] = [];

  for (const raw of value) {
    if (typeof raw === "string") {
      if (raw.startsWith("data:image/")) {
        result.push({
          id: makeAttachmentID(),
          kind: "image",
          name: "Reference image",
          url: raw,
        });
        continue;
      }

      const url = normaliseURL(raw);
      if (url) {
        result.push({ id: makeAttachmentID(), kind: "link", name: url, url });
      }
      continue;
    }

    const object = asRecord(raw);
    if (!object) continue;

    const rawURL =
      object.url ?? object.href ?? object.src ?? object.public_url ?? object.download_url;
    if (typeof rawURL !== "string") continue;

    const rawKind = object.kind ?? object.type;
    const mime =
      typeof object.mime_type === "string"
        ? object.mime_type
        : typeof object.mime === "string"
          ? object.mime
          : undefined;

    const isImage =
      rawURL.startsWith("data:image/") || mime?.startsWith("image/") || rawKind === "image";
    const validURL = isImage ? rawURL : normaliseURL(rawURL);
    if (!validURL) continue;

    const nameValue = object.name ?? object.filename ?? object.title;

    result.push({
      id: typeof object.id === "string" ? object.id : makeAttachmentID(),
      kind: isImage ? "image" : "link",
      name:
        typeof nameValue === "string"
          ? nameValue
          : isImage
            ? "Reference image"
            : validURL,
      url: validURL,
      mime_type: mime,
      size: typeof object.size === "number" ? object.size : undefined,
    });
  }

  return result;
}

export function serialiseAttachments(attachments: SourcingAttachment[]) {
  return attachments.map((attachment) => ({
    id: attachment.id,
    type: attachment.kind,
    kind: attachment.kind,
    name: attachment.name,
    url: attachment.url,
    mime_type: attachment.mime_type,
    size: attachment.size,
  }));
}

export function attachmentPayloadBytes(attachments: SourcingAttachment[]): number {
  return new TextEncoder().encode(JSON.stringify(serialiseAttachments(attachments))).byteLength;
}

export function attachmentFallback(attachments: SourcingAttachment[]): string {
  const imageCount = attachments.filter((attachment) => attachment.kind === "image").length;
  const linkCount = attachments.filter((attachment) => attachment.kind === "link").length;

  if (imageCount > 0 && linkCount > 0) return "Shared references.";
  if (imageCount === 1) return "Shared an image.";
  if (imageCount > 1) return `Shared ${imageCount} images.`;
  if (linkCount === 1) return "Shared a link.";
  return `Shared ${linkCount} links.`;
}

export function createLinkAttachment(value: string): SourcingAttachment {
  const url = normaliseURL(value);
  if (!url) throw new Error("Enter a valid http or https link.");

  return {
    id: makeAttachmentID(),
    kind: "link",
    name: url,
    url,
  };
}

function readFileAsDataURL(file: File): Promise<string> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader();
    reader.onload = () => {
      if (typeof reader.result === "string") resolve(reader.result);
      else reject(new Error("Unable to read image."));
    };
    reader.onerror = () => reject(new Error("Unable to read image."));
    reader.readAsDataURL(file);
  });
}

function loadBrowserImage(src: string): Promise<HTMLImageElement> {
  return new Promise((resolve, reject) => {
    const image = new Image();
    image.onload = () => resolve(image);
    image.onerror = () => reject(new Error("Unable to process image."));
    image.src = src;
  });
}

export async function imageToSourcingAttachment(
  file: File,
  target: "message" | "offer",
): Promise<SourcingAttachment> {
  if (!file.type.startsWith("image/")) {
    throw new Error("Only image attachments are supported.");
  }

  if (file.size > MAX_SOURCE_IMAGE_BYTES) {
    throw new Error("Images must be 8 MB or smaller.");
  }

  const source = await readFileAsDataURL(file);
  const image = await loadBrowserImage(source);
  const maxLength =
    target === "offer" ? OFFER_IMAGE_DATA_URL_LENGTH : MESSAGE_IMAGE_DATA_URL_LENGTH;

  let maxDimension = target === "offer" ? 720 : 900;
  const qualities = target === "offer" ? [0.62, 0.5, 0.4, 0.32, 0.25] : [0.72, 0.58, 0.46, 0.36, 0.28];
  let output = "";

  for (let attempt = 0; attempt < 6; attempt += 1) {
    const scale = Math.min(1, maxDimension / Math.max(image.naturalWidth, image.naturalHeight));
    const width = Math.max(1, Math.round(image.naturalWidth * scale));
    const height = Math.max(1, Math.round(image.naturalHeight * scale));

    const canvas = document.createElement("canvas");
    canvas.width = width;
    canvas.height = height;

    const context = canvas.getContext("2d");
    if (!context) throw new Error("Unable to prepare image.");

    context.fillStyle = "#ffffff";
    context.fillRect(0, 0, width, height);
    context.drawImage(image, 0, 0, width, height);

    for (const quality of qualities) {
      const candidate = canvas.toDataURL("image/jpeg", quality);
      if (candidate.length <= maxLength) {
        output = candidate;
        break;
      }
    }

    if (output) break;
    maxDimension = Math.max(320, Math.round(maxDimension * 0.76));
  }

  if (!output) {
    throw new Error("This image could not be compressed enough. Try cropping it first.");
  }

  return {
    id: makeAttachmentID(),
    kind: "image",
    name: file.name || `reference-${Date.now()}.jpg`,
    url: output,
    mime_type: "image/jpeg",
    size: Math.round(output.length * 0.75),
  };
}
