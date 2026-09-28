import type {
  ProductRequest,
  ProductRequestMessage,
  SourcingConfirmation,
  SourcingOffer,
} from "@/lib/api/contracts/product-request";

export type CustomerSourcingTimelineItem =
  | {
      id: string;
      kind: "request";
      at: string;
      request: ProductRequest;
    }
  | {
      id: string;
      kind: "message";
      at: string;
      message: ProductRequestMessage;
    }
  | {
      id: string;
      kind: "offer_sent";
      at: string;
      offer: SourcingOffer;
      offerNumber: number;
    }
  | {
      id: string;
      kind: "offer_response";
      at: string;
      offer: SourcingOffer;
      offerNumber: number;
      response: "accepted" | "rejected";
    }
  | {
      id: string;
      kind: "agreement";
      at: string;
      confirmation: SourcingConfirmation;
    }
  | {
      id: string;
      kind: "order_created";
      at: string;
      confirmation: SourcingConfirmation;
    };

function safeTime(value?: string): number {
  if (!value) return 0;
  const parsed = new Date(value).getTime();
  return Number.isFinite(parsed) ? parsed : 0;
}

export function customerOfferNumberMap(
  offers: SourcingOffer[],
): Map<string, number> {
  const ordered = [...offers]
    .filter((offer) => offer.status !== "draft")
    .sort((a, b) => safeTime(a.created_at) - safeTime(b.created_at));

  return new Map(ordered.map((offer, index) => [offer.id, index + 1]));
}

export function buildCustomerSourcingTimeline(
  request: ProductRequest,
  messages: ProductRequestMessage[],
  offers: SourcingOffer[],
  confirmation: SourcingConfirmation | null,
): CustomerSourcingTimelineItem[] {
  const items: CustomerSourcingTimelineItem[] = [
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

  const numbers = customerOfferNumberMap(offers);

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
        response:
          offer.status === "customer_rejected" ? "rejected" : "accepted",
      });
    }
  }

  if (confirmation) {
    const finalizedAt =
      offers.find((offer) => offer.id === confirmation.offer_id)?.finalized_at ??
      confirmation.created_at;

    items.push({
      id: `agreement:${confirmation.id}`,
      kind: "agreement",
      at: finalizedAt,
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
