import type {
  AdminProductRequest,
  AdminProductRequestMessage,
  AdminSourcingConfirmation,
  AdminSourcingOffer,
} from "@/lib/admin/product-request-types";

export type SourcingTimelineItem =
  | {
      id: string;
      kind: "request";
      at: string;
      request: AdminProductRequest;
    }
  | {
      id: string;
      kind: "message";
      at: string;
      message: AdminProductRequestMessage;
    }
  | {
      id: string;
      kind: "offer_sent";
      at: string;
      offer: AdminSourcingOffer;
      offerNumber: number;
    }
  | {
      id: string;
      kind: "offer_response";
      at: string;
      offer: AdminSourcingOffer;
      offerNumber: number;
      response: "accepted" | "rejected";
    }
  | {
      id: string;
      kind: "agreement";
      at: string;
      confirmation: AdminSourcingConfirmation;
    }
  | {
      id: string;
      kind: "order_created";
      at: string;
      confirmation: AdminSourcingConfirmation;
    };

function safeTime(value?: string): number {
  if (!value) return 0;
  const parsed = new Date(value).getTime();
  return Number.isFinite(parsed) ? parsed : 0;
}

export function offerNumberMap(offers: AdminSourcingOffer[]): Map<string, number> {
  const ordered = [...offers]
    .filter((offer) => offer.status !== "draft")
    .sort((a, b) => safeTime(a.created_at) - safeTime(b.created_at));

  return new Map(ordered.map((offer, index) => [offer.id, index + 1]));
}

export function buildSourcingTimeline(
  request: AdminProductRequest,
  messages: AdminProductRequestMessage[],
  offers: AdminSourcingOffer[],
  confirmation: AdminSourcingConfirmation | null,
): SourcingTimelineItem[] {
  const items: SourcingTimelineItem[] = [
    {
      id: `request:${request.id}`,
      kind: "request",
      at: request.created_at,
      request,
    },
  ];

  for (const message of messages) {
    items.push({
      id: `message:${message.id}`,
      kind: "message",
      at: message.created_at,
      message,
    });
  }

  const numbers = offerNumberMap(offers);

  for (const offer of offers) {
    if (offer.status === "draft") continue;

    const offerNumber = numbers.get(offer.id) ?? 1;
    items.push({
      id: `offer:${offer.id}`,
      kind: "offer_sent",
      at: offer.sent_at ?? offer.created_at,
      offer,
      offerNumber,
    });

    if (
      (offer.status === "customer_accepted" ||
        offer.status === "customer_rejected" ||
        offer.status === "finalized") &&
      offer.customer_responded_at
    ) {
      items.push({
        id: `offer-response:${offer.id}`,
        kind: "offer_response",
        at: offer.customer_responded_at,
        offer,
        offerNumber,
        response: offer.status === "customer_rejected" ? "rejected" : "accepted",
      });
    }
  }

  if (confirmation) {
    items.push({
      id: `agreement:${confirmation.id}`,
      kind: "agreement",
      at:
        offers.find((offer) => offer.id === confirmation.offer_id)?.finalized_at ??
        confirmation.created_at,
      confirmation,
    });

    if (confirmation.created_order_id) {
      items.push({
        id: `order:${confirmation.id}`,
        kind: "order_created",
        at: confirmation.updated_at,
        confirmation,
      });
    }
  }

  return items.sort((a, b) => safeTime(a.at) - safeTime(b.at));
}
