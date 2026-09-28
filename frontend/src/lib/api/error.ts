export type CommerceApiErrorPayload = {
  error?: {
    code?: string;
    message?: string;
  };
};

export class CommerceApiError extends Error {
  readonly status: number;
  readonly code: string | null;

  constructor(
    message: string,
    status: number,
    code: string | null = null,
  ) {
    super(message);

    this.name = "CommerceApiError";
    this.status = status;
    this.code = code;
  }
}