import type {
  AdminCRMAttachment,
} from "@/lib/admin/support-types";

export const MAX_ADMIN_SUPPORT_ATTACHMENTS =
  4;

const MAX_SOURCE_IMAGE_BYTES =
  8 * 1024 * 1024;

const MAX_DATA_URL_LENGTH =
  1_200_000;

const MAX_LINK_LENGTH =
  4096;

function makeAttachmentID(): string {
  if (
    typeof crypto !==
      "undefined" &&
    typeof crypto.randomUUID ===
      "function"
  ) {
    return crypto.randomUUID();
  }

  return `support-${Date.now()}-${Math.random()
    .toString(36)
    .slice(2, 10)}`;
}

function asObject(
  value: unknown,
): Record<string, unknown> | null {
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

export function normaliseSupportAttachmentURL(
  value: string,
): string | null {
  const trimmed =
    value.trim();

  if (
    !trimmed ||
    trimmed.length >
      MAX_LINK_LENGTH
  ) {
    return null;
  }

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

function validInlineImage(
  value: string,
): boolean {
  const lower =
    value.toLowerCase();

  return [
    "data:image/jpeg;base64,",
    "data:image/jpg;base64,",
    "data:image/png;base64,",
    "data:image/webp;base64,",
    "data:image/gif;base64,",
  ].some(
    (prefix) =>
      lower.startsWith(
        prefix,
      ),
  );
}

export function parseAdminSupportAttachments(
  input: unknown,
): AdminCRMAttachment[] {
  if (!input) {
    return [];
  }

  let value =
    input;

  if (
    typeof value ===
    "string"
  ) {
    const trimmed =
      value.trim();

    if (!trimmed) {
      return [];
    }

    if (
      trimmed.startsWith(
        "[",
      )
    ) {
      try {
        value =
          JSON.parse(
            trimmed,
          );
      } catch {
        return [];
      }
    } else if (
      validInlineImage(
        trimmed,
      )
    ) {
      return [
        {
          id:
            makeAttachmentID(),

          kind:
            "image",

          type:
            "image",

          name:
            "Image attachment",

          url:
            trimmed,
        },
      ];
    } else {
      const url =
        normaliseSupportAttachmentURL(
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

          type:
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
    AdminCRMAttachment[] =
      [];

  for (
    const raw of value
  ) {
    if (
      typeof raw ===
      "string"
    ) {
      if (
        validInlineImage(
          raw,
        )
      ) {
        result.push({
          id:
            makeAttachmentID(),

          kind:
            "image",

          type:
            "image",

          name:
            "Image attachment",

          url:
            raw,
        });

        continue;
      }

      const url =
        normaliseSupportAttachmentURL(
          raw,
        );

      if (url) {
        result.push({
          id:
            makeAttachmentID(),

          kind:
            "link",

          type:
            "link",

          name:
            url,

          url,
        });
      }

      continue;
    }

    const object =
      asObject(
        raw,
      );

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

    const image =
      validInlineImage(
        rawURL,
      ) ||
      mime?.startsWith(
        "image/",
      ) === true ||
      rawKind ===
        "image";

    const url =
      image
        ? validInlineImage(
            rawURL,
          ) ||
          rawURL.startsWith(
            "https://",
          ) ||
          rawURL.startsWith(
            "http://",
          )
          ? rawURL
          : null
        : normaliseSupportAttachmentURL(
            rawURL,
          );

    if (!url) {
      continue;
    }

    const rawName =
      object.name ??
      object.filename ??
      object.title;

    result.push({
      id:
        typeof object.id ===
          "string" &&
        object.id
          ? object.id
          : makeAttachmentID(),

      kind:
        image
          ? "image"
          : "link",

      type:
        image
          ? "image"
          : "link",

      name:
        typeof rawName ===
          "string" &&
        rawName.trim()
          ? rawName.trim()
          : image
            ? "Image attachment"
            : url,

      url,

      mime_type:
        mime,

      size:
        typeof object.size ===
        "number"
          ? object.size
          : undefined,
    });
  }

  return result.slice(
    0,
    MAX_ADMIN_SUPPORT_ATTACHMENTS,
  );
}

export function serialiseAdminSupportAttachments(
  attachments:
    AdminCRMAttachment[],
): Array<
  Record<
    string,
    unknown
  >
> {
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

function readFileAsDataURL(
  file: File,
): Promise<string> {
  return new Promise(
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
        () =>
          reject(
            new Error(
              "Unable to read image.",
            ),
          );

      reader.readAsDataURL(
        file,
      );
    },
  );
}

function loadBrowserImage(
  src: string,
): Promise<HTMLImageElement> {
  return new Promise(
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

export async function imageFileToAdminSupportAttachment(
  file: File,
): Promise<AdminCRMAttachment> {
  const supportedImageTypes =
    new Set([
      "image/jpeg",
      "image/png",
      "image/webp",
      "image/gif",
    ]);

  if (
    !supportedImageTypes.has(
      file.type.toLowerCase(),
    )
  ) {
    throw new Error(
      "Use a JPEG, PNG, WebP or GIF image.",
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

  const source =
    await readFileAsDataURL(
      file,
    );

  const image =
    await loadBrowserImage(
      source,
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
    output =
      canvas.toDataURL(
        "image/jpeg",
        0.46,
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

    type:
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

export function createAdminSupportLinkAttachment(
  rawURL: string,
): AdminCRMAttachment {
  const url =
    normaliseSupportAttachmentURL(
      rawURL,
    );

  if (!url) {
    throw new Error(
      "Enter a valid http or https link.",
    );
  }

  return {
    id:
      makeAttachmentID(),

    kind:
      "link",

    type:
      "link",

    name:
      url,

    url,
  };
}