import {
  unzip,
} from "fflate";

import {
  adminFetch,
} from "./api";

export const MAX_CATALOG_WORKBOOK_BYTES =
  20 * 1024 * 1024;

export const MAX_CATALOG_ZIP_BYTES =
  512 * 1024 * 1024;

const MAX_PRODUCT_IMAGE_BYTES =
  15 * 1024 * 1024;

const MAX_PACKAGE_IMAGES =
  1000;

const MAX_EXTRACTED_IMAGE_BYTES =
  1024 * 1024 * 1024;

const XLSX_CONTENT_TYPE =
  "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet";

const IMAGE_CONTENT_TYPES:
  Record<string, string> = {
    ".jpg": "image/jpeg",
    ".jpeg": "image/jpeg",
    ".png": "image/png",
    ".webp": "image/webp",
    ".avif": "image/avif",
  };

export type CatalogPackageImage = {
  reference: string;
  filename: string;
  file: File;
};

export type PreparedCatalogImport = {
  workbook: File;

  images: CatalogPackageImage[];

  sourceFilename: string;

  isPackage: boolean;
};

type AdminUploadTarget = {
  provider: string;

  key: string;

  method: string;

  url: string;

  headers?: Record<
    string,
    string
  >;

  expires_at: string;
};

type AdminCreateUploadResponse = {
  filename: string;

  purpose: string;

  upload: AdminUploadTarget;
};

export type CatalogImageUploadProgress = {
  completed: number;

  total: number;

  filename: string;
};

type ArchiveEntry = {
  path: string;

  bytes: Uint8Array;
};

function fileExtension(
  filename: string,
): string {
  const normalized =
    filename
      .trim()
      .toLowerCase();

  const index =
    normalized.lastIndexOf(
      ".",
    );

  if (index < 0) {
    return "";
  }

  return normalized.slice(
    index,
  );
}

function basename(
  value: string,
): string {
  const normalized =
    value.replaceAll(
      "\\",
      "/",
    );

  const parts =
    normalized.split("/");

  return (
    parts[
      parts.length - 1
    ] ?? ""
  );
}

function dirname(
  value: string,
): string {
  const normalized =
    value.replaceAll(
      "\\",
      "/",
    );

  const index =
    normalized.lastIndexOf(
      "/",
    );

  if (index < 0) {
    return "";
  }

  return normalized.slice(
    0,
    index,
  );
}

function normalizeArchivePath(
  value: string,
): string {
  value =
    value
      .trim()
      .replaceAll(
        "\\",
        "/",
      );

  while (
    value.startsWith(
      "./",
    )
  ) {
    value =
      value.slice(2);
  }

  value =
    value.replace(
      /^\/+/,
      "",
    );

  if (!value) {
    return "";
  }

  const segments =
    value.split("/");

  const clean: string[] =
    [];

  for (
    const segment
    of segments
  ) {
    const trimmed =
      segment.trim();

    if (
      !trimmed ||
      trimmed === "."
    ) {
      continue;
    }

    if (
      trimmed === ".."
    ) {
      throw new Error(
        "The catalog ZIP contains an unsafe file path.",
      );
    }

    clean.push(
      trimmed,
    );
  }

  return clean.join("/");
}

function copyArrayBuffer(
  bytes: Uint8Array,
): ArrayBuffer {
  const copy =
    new Uint8Array(
      bytes.byteLength,
    );

  copy.set(
    bytes,
  );

  return copy.buffer;
}

function unzipAsync(
  data: Uint8Array,
): Promise<
  Record<
    string,
    Uint8Array
  >
> {
  return new Promise(
    (
      resolve,
      reject,
    ) => {
      unzip(
        data,
        (
          error,
          files,
        ) => {
          if (error) {
            reject(
              error,
            );

            return;
          }

          resolve(
            files,
          );
        },
      );
    },
  );
}

function archiveEntries(
  files: Record<
    string,
    Uint8Array
  >,
): ArchiveEntry[] {
  const entries:
    ArchiveEntry[] = [];

  const seen =
    new Set<string>();

  for (
    const [
      rawPath,
      bytes,
    ]
    of Object.entries(
      files,
    )
  ) {
    if (
      rawPath.endsWith(
        "/",
      )
    ) {
      continue;
    }

    const path =
      normalizeArchivePath(
        rawPath,
      );

    if (!path) {
      continue;
    }

    if (
      path.startsWith(
        "__MACOSX/",
      )
    ) {
      continue;
    }

    const key =
      path.toLowerCase();

    if (
      seen.has(
        key,
      )
    ) {
      throw new Error(
        `The catalog ZIP contains duplicate path "${path}".`,
      );
    }

    seen.add(
      key,
    );

    entries.push({
      path,
      bytes,
    });
  }

  return entries;
}

function validateWorkbook(
  name: string,
  size: number,
): void {
  if (
    size <= 0
  ) {
    throw new Error(
      "The Excel workbook is empty.",
    );
  }

  if (
    !name
      .toLowerCase()
      .endsWith(
        ".xlsx",
      )
  ) {
    throw new Error(
      "The catalog workbook must be an XLSX file.",
    );
  }

  if (
    size >
    MAX_CATALOG_WORKBOOK_BYTES
  ) {
    throw new Error(
      "The Excel workbook cannot exceed 20 MiB.",
    );
  }
}

function imageContentType(
  filename: string,
): string {
  return (
    IMAGE_CONTENT_TYPES[
      fileExtension(
        filename,
      )
    ] ?? ""
  );
}

export function validateCatalogImportFile(
  file: File,
): string {
  if (
    file.size <= 0
  ) {
    return "Choose a non-empty XLSX workbook or catalog ZIP.";
  }

  const extension =
    fileExtension(
      file.name,
    );

  if (
    extension ===
    ".xlsx"
  ) {
    if (
      file.size >
      MAX_CATALOG_WORKBOOK_BYTES
    ) {
      return "The Excel workbook cannot exceed 20 MiB.";
    }

    return "";
  }

  if (
    extension ===
    ".zip"
  ) {
    if (
      file.size >
      MAX_CATALOG_ZIP_BYTES
    ) {
      return "The catalog ZIP cannot exceed 512 MiB.";
    }

    return "";
  }

  return "Choose either an XLSX workbook or a catalog ZIP.";
}

export async function prepareCatalogImport(
  file: File,
): Promise<PreparedCatalogImport> {
  const validationError =
    validateCatalogImportFile(
      file,
    );

  if (
    validationError
  ) {
    throw new Error(
      validationError,
    );
  }

  const extension =
    fileExtension(
      file.name,
    );

  /*
   * Legacy XLSX workflow.
   *
   * Existing URL-backed imports continue
   * exactly as before.
   */
  if (
    extension ===
    ".xlsx"
  ) {
    validateWorkbook(
      file.name,
      file.size,
    );

    return {
      workbook: file,

      images: [],

      sourceFilename:
        file.name,

      isPackage: false,
    };
  }

  /*
   * Catalog ZIP workflow.
   *
   * ZIP extraction happens entirely in
   * the Admin browser. The large ZIP is
   * never sent through the Vercel API.
   */
  const archiveBuffer =
    await file.arrayBuffer();

  const archive =
    await unzipAsync(
      new Uint8Array(
        archiveBuffer,
      ),
    );

  const entries =
    archiveEntries(
      archive,
    );

  const workbooks =
    entries.filter(
      entry =>
        entry.path
          .toLowerCase()
          .endsWith(
            ".xlsx",
          ),
    );

  if (
    workbooks.length ===
    0
  ) {
    throw new Error(
      "The catalog ZIP does not contain an XLSX workbook.",
    );
  }

  if (
    workbooks.length >
    1
  ) {
    throw new Error(
      "The catalog ZIP must contain exactly one XLSX workbook.",
    );
  }

  const workbookEntry =
    workbooks[0];

  const workbookFilename =
    basename(
      workbookEntry.path,
    );

  validateWorkbook(
    workbookFilename,
    workbookEntry.bytes
      .byteLength,
  );

  /*
   * Support both:
   *
   * products.xlsx
   * images/...
   *
   * and:
   *
   * batch-01/products.xlsx
   * batch-01/images/...
   */
  const workbookDirectory =
    dirname(
      workbookEntry.path,
    );

  const imagePrefix =
    workbookDirectory
      ? `${workbookDirectory}/images/`
      : "images/";

  const imageEntries =
    entries.filter(
      entry =>
        entry.path.startsWith(
          imagePrefix,
        ),
    );

  const supportedImages =
    imageEntries.filter(
      entry =>
        Boolean(
          imageContentType(
            entry.path,
          ),
        ),
    );

  if (
    supportedImages.length ===
    0
  ) {
    throw new Error(
      'The catalog ZIP must contain product images inside an "images/" folder.',
    );
  }

  if (
    supportedImages.length >
    MAX_PACKAGE_IMAGES
  ) {
    throw new Error(
      `The catalog ZIP contains too many product images. Maximum: ${MAX_PACKAGE_IMAGES}.`,
    );
  }

  const seenFilenames =
    new Set<string>();

  let extractedImageBytes =
    0;

  const images:
    CatalogPackageImage[] =
    [];

  for (
    const entry
    of supportedImages
  ) {
    const relativeReference =
      entry.path.slice(
        imagePrefix.length,
      );

    if (
      !relativeReference
    ) {
      continue;
    }

    const filename =
      basename(
        relativeReference,
      );

    const contentType =
      imageContentType(
        filename,
      );

    if (
      !contentType
    ) {
      continue;
    }

    if (
      entry.bytes
        .byteLength <= 0
    ) {
      throw new Error(
        `Product image "${relativeReference}" is empty.`,
      );
    }

    if (
      entry.bytes
        .byteLength >
      MAX_PRODUCT_IMAGE_BYTES
    ) {
      throw new Error(
        `Product image "${relativeReference}" exceeds the 15 MiB limit.`,
      );
    }

    extractedImageBytes +=
      entry.bytes
        .byteLength;

    if (
      extractedImageBytes >
      MAX_EXTRACTED_IMAGE_BYTES
    ) {
      throw new Error(
        "The extracted catalog images exceed the 1 GiB safety limit.",
      );
    }

    /*
     * Simple workbook image cells normally
     * contain only the filename, so duplicate
     * basenames would be ambiguous even when
     * stored in different ZIP subdirectories.
     */
    const filenameKey =
      filename.toLowerCase();

    if (
      seenFilenames.has(
        filenameKey,
      )
    ) {
      throw new Error(
        `The catalog ZIP contains duplicate image filename "${filename}". Image filenames must be unique.`,
      );
    }

    seenFilenames.add(
      filenameKey,
    );

    const imageFile =
      new File(
        [
          copyArrayBuffer(
            entry.bytes,
          ),
        ],
        filename,
        {
          type:
            contentType,
        },
      );

    images.push({
      reference:
        relativeReference,

      filename,

      file:
        imageFile,
    });
  }

  const workbook =
    new File(
      [
        copyArrayBuffer(
          workbookEntry.bytes,
        ),
      ],
      workbookFilename,
      {
        type:
          XLSX_CONTENT_TYPE,
      },
    );

  return {
    workbook,

    images,

    sourceFilename:
      file.name,

    isPackage: true,
  };
}

async function uploadOneImage(
  image:
    CatalogPackageImage,
): Promise<string> {
  const signed =
    await adminFetch<AdminCreateUploadResponse>(
      "/uploads",
      {
        method:
          "POST",

        body:
          JSON.stringify({
            purpose:
              "product_image",

            filename:
              image.filename,

            content_type:
              image.file.type,

            content_length:
              image.file.size,
          }),
      },
    );

  const upload =
    signed.upload;

  if (
    !upload?.url ||
    !upload?.key
  ) {
    throw new Error(
      `The upload service returned an invalid target for "${image.filename}".`,
    );
  }

  const response =
    await fetch(
      upload.url,
      {
        method:
          upload.method ||
          "PUT",

        headers: {
          ...(
            upload.headers ??
            {}
          ),
        },

        body:
          image.file,
      },
    );

  if (
    !response.ok
  ) {
    throw new Error(
      `Storage upload failed for "${image.filename}" (${response.status}).`,
    );
  }

  return upload.key;
}

export async function uploadCatalogPackageImages(
  prepared:
    PreparedCatalogImport,

  onProgress?: (
    progress:
      CatalogImageUploadProgress,
  ) => void,
): Promise<
  Record<
    string,
    string
  >
> {
  if (
    prepared.images.length ===
    0
  ) {
    return {};
  }

  const assetMap:
    Record<
      string,
      string
    > = {};

  let nextIndex = 0;
  let completed = 0;

  /*
   * A small bounded pool is much faster
   * than 300 sequential uploads without
   * flooding the browser, Admin API or
   * object-storage endpoint.
   */
  const concurrency =
    Math.min(
      4,
      prepared.images.length,
    );

  async function worker() {
    for (;;) {
      const index =
        nextIndex;

      nextIndex += 1;

      if (
        index >=
        prepared.images.length
      ) {
        return;
      }

      const image =
        prepared.images[
          index
        ];

      const storageKey =
        await uploadOneImage(
          image,
        );

      /*
       * Store both forms.

       * This allows Excel to use:
       *
       * product-01.webp
       *
       * or:
       *
       * some-folder/product-01.webp
       */
      assetMap[
        image.reference
      ] =
        storageKey;

      assetMap[
        image.filename
      ] =
        storageKey;

      completed += 1;

      onProgress?.({
        completed,

        total:
          prepared.images
            .length,

        filename:
          image.filename,
      });
    }
  }

  await Promise.all(
    Array.from(
      {
        length:
          concurrency,
      },
      () =>
        worker(),
    ),
  );

  return assetMap;
}