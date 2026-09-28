// ENE_INVOICE_RUNTIME_V3_NULL_SAFE
// Location: src/app/api/storefront/order/[orderId]/invoice/route.ts

import { NextResponse } from "next/server";

import { accountCommerceFetch } from "@/lib/account/request";
import { accountRouteErrorResponse } from "@/lib/account/route-error";
import { setAccountAuthCookies } from "@/lib/account/session";
import type { AccountDataResponse } from "@/lib/api/contracts/account";
import { formatMoney } from "@/lib/money/format";

type RouteContext = {
  params: Promise<{
    orderId: string;
  }>;
};

type InvoiceItem = {
  variant_id: string;
  sku: string;
  product_name: string;
  quantity: number;
  unit_price_amount: number;
  line_total_amount: number;
};

type Invoice = {
  id?: string;
  invoice_number: string;
  issued_at: string;
  order_id: string;
  order_number: string;
  order_status: string;
  payment_status: string;
  payment_method: string;
  currency: string;
  subtotal_amount: number;
  discount_amount: number;
  shipping_amount: number;
  total_amount: number;
  customer_name: string;
  customer_phone: string;
  customer_email?: string;
  shipping_address_line1: string;
  shipping_address_line2?: string;
  shipping_city: string;
  shipping_area: string;
  shipping_postal_code?: string;
  delivery_method: string;
  payment_due_at?: string;
  paid_at?: string;
  items: InvoiceItem[];
};

function escapeHTML(value: string | number | undefined | null) {
  return String(value ?? "")
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;")
    .replaceAll("'", "&#039;");
}

function humanizeSafe(value: unknown) {
  if (typeof value !== "string" || !value.trim()) return "â€”";

  return value
    .trim()
    .replaceAll("-", "_")
    .replaceAll(" ", "_")
    .replaceAll("_", " ")
    .replace(/\b\w/g, (character) => character.toUpperCase());
}

function safeInvoiceFilename(value: unknown) {
  const cleaned = (typeof value === "string" ? value : "invoice")
    .trim()
    .replace(/[^a-zA-Z0-9._-]+/g, "-")
    .replace(/^-+|-+$/g, "")
    .slice(0, 120);

  return cleaned || "invoice";
}



type OrderFallbackItem = Partial<InvoiceItem> & {
  id?: string;
};

type OrderFallback = {
  id?: string;
  order_number?: string;
  status?: string;
  payment_status?: string;
  payment_method?: string;
  currency?: string;
  subtotal_amount?: number;
  discount_amount?: number;
  shipping_amount?: number;
  total_amount?: number;
  customer_name?: string;
  customer_phone?: string;
  customer_email?: string;
  shipping_address_line1?: string;
  shipping_address_line2?: string;
  shipping_city?: string;
  shipping_area?: string;
  shipping_postal_code?: string;
  delivery_method?: string;
  payment_due_at?: string;
  paid_at?: string;
  created_at?: string;
  items?: OrderFallbackItem[];
};

function recordValue(value: unknown): Record<string, unknown> {
  return value && typeof value === "object" && !Array.isArray(value)
    ? (value as Record<string, unknown>)
    : {};
}

function firstString(...values: unknown[]) {
  for (const value of values) {
    if (typeof value === "string" && value.trim()) return value;
  }
  return "";
}

function firstNumber(...values: unknown[]) {
  for (const value of values) {
    if (typeof value === "number" && Number.isFinite(value)) return value;
    if (typeof value === "string" && value.trim()) {
      const parsed = Number(value);
      if (Number.isFinite(parsed)) return parsed;
    }
  }
  return 0;
}

function normalizedItems(primary: unknown, fallback: unknown): InvoiceItem[] {
  const source = Array.isArray(primary)
    ? primary
    : Array.isArray(fallback)
      ? fallback
      : [];

  return source.map((entry) => {
    const item = recordValue(entry);
    return {
      variant_id: firstString(item.variant_id),
      sku: firstString(item.sku),
      product_name: firstString(item.product_name, item.name, "Item"),
      quantity: firstNumber(item.quantity),
      unit_price_amount: firstNumber(item.unit_price_amount, item.unit_price),
      line_total_amount: firstNumber(item.line_total_amount, item.total_amount),
    };
  });
}

function normalizeInvoice(
  rawValue: unknown,
  orderValue: unknown,
  requestedOrderId: string,
): Invoice {
  const raw = recordValue(rawValue);
  const order = recordValue(orderValue);

  const orderId = firstString(raw.order_id, order.id, requestedOrderId);
  const orderNumber = firstString(raw.order_number, order.order_number, orderId);

  return {
    id: firstString(raw.id) || undefined,
    invoice_number: firstString(raw.invoice_number, `INV-${orderNumber}`),
    issued_at: firstString(raw.issued_at, raw.created_at, order.created_at),
    order_id: orderId,
    order_number: orderNumber,
    order_status: firstString(raw.order_status, order.status),
    payment_status: firstString(raw.payment_status, order.payment_status),
    payment_method: firstString(raw.payment_method, order.payment_method),
    currency: firstString(raw.currency, order.currency, "BDT"),
    subtotal_amount: firstNumber(raw.subtotal_amount, order.subtotal_amount),
    discount_amount: firstNumber(raw.discount_amount, order.discount_amount),
    shipping_amount: firstNumber(raw.shipping_amount, order.shipping_amount),
    total_amount: firstNumber(raw.total_amount, order.total_amount),
    customer_name: firstString(raw.customer_name, order.customer_name),
    customer_phone: firstString(raw.customer_phone, order.customer_phone),
    customer_email: firstString(raw.customer_email, order.customer_email) || undefined,
    shipping_address_line1: firstString(raw.shipping_address_line1, order.shipping_address_line1),
    shipping_address_line2: firstString(raw.shipping_address_line2, order.shipping_address_line2) || undefined,
    shipping_city: firstString(raw.shipping_city, order.shipping_city),
    shipping_area: firstString(raw.shipping_area, order.shipping_area),
    shipping_postal_code: firstString(raw.shipping_postal_code, order.shipping_postal_code) || undefined,
    delivery_method: firstString(raw.delivery_method, order.delivery_method),
    payment_due_at: firstString(raw.payment_due_at, order.payment_due_at) || undefined,
    paid_at: firstString(raw.paid_at, order.paid_at) || undefined,
    items: normalizedItems(raw.items, order.items),
  };
}

function needsOrderFallback(value: unknown) {
  const invoice = recordValue(value);
  return (
    !firstString(invoice.order_status) ||
    !firstString(invoice.payment_status) ||
    !firstString(invoice.payment_method) ||
    !firstString(invoice.delivery_method) ||
    !firstString(invoice.customer_name) ||
    !firstString(invoice.shipping_address_line1) ||
    !Array.isArray(invoice.items)
  );
}

function dateTime(value?: string) {
  if (!value) return "â€”";
  if (Number.isNaN(Date.parse(value))) return value;

  return new Intl.DateTimeFormat("en-BD", {
    day: "numeric",
    month: "short",
    year: "numeric",
    hour: "numeric",
    minute: "2-digit",
  }).format(new Date(value));
}

function buildInvoiceHTML(invoice: Invoice, autoPrint = false) {
  const itemRows = invoice.items
    .map(
      (item) => `
        <tr>
          <td class="item-cell" data-label="Item">
            <strong>${escapeHTML(item.product_name)}</strong>
            <small>SKU ${escapeHTML(item.sku)}</small>
          </td>
          <td class="number" data-label="Qty">${escapeHTML(item.quantity)}</td>
          <td class="number" data-label="Unit price">${escapeHTML(formatMoney(item.unit_price_amount, invoice.currency))}</td>
          <td class="number" data-label="Line total"><strong>${escapeHTML(formatMoney(item.line_total_amount, invoice.currency))}</strong></td>
        </tr>`,
    )
    .join("");

  const address = [
    invoice.shipping_address_line1,
    invoice.shipping_address_line2,
    invoice.shipping_area,
    invoice.shipping_city,
    invoice.shipping_postal_code,
  ]
    .filter(Boolean)
    .map((line) => escapeHTML(line))
    .join("<br />");

  const paymentDetail = invoice.paid_at
    ? `Paid ${escapeHTML(dateTime(invoice.paid_at))}`
    : invoice.payment_due_at
      ? `Due ${escapeHTML(dateTime(invoice.payment_due_at))}`
      : "Payment timing follows the order status.";

  return `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <meta name="robots" content="noindex,nofollow" />
  <meta name="color-scheme" content="light" />
  <title>${escapeHTML(invoice.invoice_number)} Â· ENE DEI</title>
  <style>
    :root{color-scheme:light;--red:#e60023;--red-dark:#bd001d;--ink:#171719;--muted:#727279;--line:#e8e7ea;--soft:#f7f6f7;--glass:rgba(255,255,255,.73)}
    *{box-sizing:border-box}
    html{background:#f8f6f7}
    body{margin:0;min-height:100vh;background:radial-gradient(circle at 8% 7%,rgba(230,0,35,.09),transparent 29rem),radial-gradient(circle at 92% 28%,rgba(255,94,126,.09),transparent 26rem),linear-gradient(180deg,#fbfafb,#f4f2f3 62%,#fafafa);color:var(--ink);font-family:Inter,-apple-system,BlinkMacSystemFont,"Segoe UI",sans-serif;-webkit-font-smoothing:antialiased}
    body:before,body:after{content:"";position:fixed;z-index:-1;border-radius:999px;pointer-events:none;filter:blur(8px)}
    body:before{width:24rem;height:24rem;left:-12rem;top:12rem;background:radial-gradient(circle,rgba(230,0,35,.1),transparent 68%)}
    body:after{width:30rem;height:30rem;right:-15rem;bottom:1rem;background:radial-gradient(circle,rgba(255,96,129,.1),transparent 68%)}
    a{color:inherit}
    .shell{width:min(calc(100% - 28px),980px);margin:28px auto;padding:12px;border:1px solid rgba(255,255,255,.92);border-radius:32px;background:var(--glass);box-shadow:inset 0 1px 0 rgba(255,255,255,.98),0 28px 80px rgba(31,25,29,.12);backdrop-filter:blur(28px) saturate(155%);-webkit-backdrop-filter:blur(28px) saturate(155%)}
    .toolbar{min-height:52px;display:flex;align-items:center;justify-content:flex-end;gap:12px;padding:2px 4px 12px}
    .back,.print{min-height:44px;display:inline-flex;align-items:center;justify-content:center;gap:7px;border-radius:13px;padding:0 15px;font-size:13px;font-weight:800;text-decoration:none;cursor:pointer;backdrop-filter:blur(20px) saturate(160%);-webkit-backdrop-filter:blur(20px) saturate(160%);box-shadow:inset 0 1px 0 rgba(255,255,255,.72)}
    .back{border:1px solid rgba(230,0,35,.22);background:rgba(255,255,255,.78);color:var(--red)}
    .print{border:1px solid rgba(255,255,255,.34);background:rgba(230,0,35,.86);color:#fff;box-shadow:inset 0 1px 0 rgba(255,255,255,.42),0 8px 20px rgba(230,0,35,.17)}
    .back:hover{background:#fff;border-color:rgba(230,0,35,.4)}.print:hover{background:rgba(189,0,29,.92);border-color:rgba(255,255,255,.38)}
    .back:focus-visible,.print:focus-visible{outline:3px solid rgba(230,0,35,.22);outline-offset:3px}
    .paper{overflow:hidden;border:1px solid rgba(23,23,25,.07);border-radius:24px;background:#fff;box-shadow:0 12px 42px rgba(27,24,26,.07)}
    .top{display:flex;justify-content:space-between;align-items:flex-start;gap:32px;padding:34px 36px 30px;border-bottom:1px solid var(--line)}
    .brand{font-family:Georgia,"Times New Roman",serif;font-size:30px;font-weight:700;letter-spacing:.12em}.brand small{display:block;margin-top:5px;font-family:Inter,-apple-system,BlinkMacSystemFont,"Segoe UI",sans-serif;font-size:8px;font-weight:800;letter-spacing:.28em;color:var(--muted)}
    .invoice-heading{text-align:right}.invoice-heading h1{margin:0;font-size:31px;line-height:1;letter-spacing:-.045em}.invoice-no{margin-top:8px;color:var(--red);font-size:14px;font-weight:850}.muted{color:var(--muted);font-size:12px;line-height:1.55}.invoice-heading .muted{margin-top:5px}
    .meta-grid{display:grid;grid-template-columns:1.05fr .95fr;gap:18px;padding:26px 36px}
    .card{padding:18px;border:1px solid var(--line);border-radius:16px;background:linear-gradient(145deg,#faf9fa,#f6f5f6)}
    .card h2{margin:0 0 10px;color:var(--muted);font-size:10px;font-weight:850;letter-spacing:.13em;text-transform:uppercase}.card p{margin:0;color:#37373b;font-size:13px;line-height:1.62}.card p+p{margin-top:12px}
    .status-row{display:flex;flex-wrap:wrap;gap:6px;margin-top:12px}.status{display:inline-flex;align-items:center;gap:6px;min-height:28px;padding:0 9px;border:1px solid rgba(230,0,35,.13);border-radius:999px;background:#fff;color:#b9001c;font-size:10px;font-weight:800}.status:before{content:"";width:6px;height:6px;border-radius:999px;background:currentColor}
    .items{padding:2px 36px 26px}table{width:100%;border-collapse:collapse}th{text-align:left;padding:11px 9px;border-bottom:1px solid var(--line);color:var(--muted);font-size:10px;font-weight:820;letter-spacing:.08em;text-transform:uppercase}td{padding:15px 9px;border-bottom:1px solid var(--line);color:#3a3a3f;font-size:12px}td small{display:block;margin-top:4px;color:var(--muted);font-size:10px}.item-cell strong{color:var(--ink);font-size:12px}.number{text-align:right;white-space:nowrap}
    .bottom{display:grid;grid-template-columns:1fr 310px;gap:28px;padding:0 36px 34px}.note{align-self:end;color:var(--muted);font-size:11px;line-height:1.6}.note strong{display:block;margin-bottom:4px;color:#45454b;font-size:11px}.totals{border:1px solid var(--line);border-radius:16px;padding:15px;background:#faf9fa}.row{display:flex;justify-content:space-between;gap:16px;padding:6px 0;color:#616168;font-size:12px}.row strong{color:#343438}.total{margin-top:7px;padding-top:13px;border-top:1px solid var(--line);color:var(--ink);font-size:16px;font-weight:850}.total span:last-child{font-size:19px;letter-spacing:-.025em}
    .footer{display:flex;justify-content:space-between;gap:18px;padding:16px 36px 20px;border-top:1px solid var(--line);color:#8a8a90;font-size:10px;line-height:1.5}.footer strong{color:#5b5b61}
    @supports not ((backdrop-filter:blur(1px)) or (-webkit-backdrop-filter:blur(1px))){.shell{background:rgba(255,255,255,.96)}}
    @media(max-width:680px){body{background:#f8f6f7}.shell{width:100%;margin:0;padding:8px;border:0;border-radius:0;box-shadow:none}.toolbar{padding:2px 2px 10px}.back,.print{padding:0 12px;font-size:12px}.paper{border-radius:20px}.top{padding:25px 18px 22px;flex-direction:column;gap:18px}.invoice-heading{text-align:left}.brand{font-size:25px}.invoice-heading h1{font-size:27px}.meta-grid{grid-template-columns:1fr;padding:18px;gap:10px}.items{padding:0 18px 20px}thead{display:none}table,tbody,tr,td{display:block;width:100%}tr{padding:12px 0;border-bottom:1px solid var(--line)}td{padding:5px 0;border:0;display:flex;align-items:flex-start;justify-content:space-between;gap:18px;text-align:right;white-space:normal}td:before{content:attr(data-label);color:var(--muted);font-size:10px;font-weight:800;text-transform:uppercase;letter-spacing:.07em}.item-cell{display:block;text-align:left;padding-bottom:8px}.item-cell:before{display:none}.number{text-align:right}.bottom{grid-template-columns:1fr;padding:0 18px 24px;gap:16px}.totals{width:100%}.footer{padding:15px 18px 20px;flex-direction:column;gap:5px}}
    @media(max-width:390px){.toolbar{align-items:stretch}.back,.print{flex:1;padding:0 9px}.footer{font-size:9px}}
    @media(prefers-reduced-motion:reduce){.back,.print{transition:none}}
    @media print{@page{size:A4;margin:12mm}html,body{background:#fff}.shell{width:100%;margin:0;padding:0;border:0;border-radius:0;background:#fff;box-shadow:none;backdrop-filter:none;-webkit-backdrop-filter:none}.toolbar{display:none}.paper{border:0;border-radius:0;box-shadow:none}.top{padding-top:0}.footer{padding-bottom:0}body:before,body:after{display:none}}
  </style>
</head>
<body>
  <main class="shell">
    <div class="toolbar">
      <button class="print" type="button" onclick="window.print()">Print / Save PDF</button>
    </div>

    <article class="paper">
      <header class="top">
        <div class="brand">ENE DEI<small>PRECISION COMMERCE</small></div>
        <div class="invoice-heading">
          <h1>Invoice</h1>
          <div class="invoice-no">${escapeHTML(invoice.invoice_number)}</div>
          <div class="muted">Issued ${escapeHTML(dateTime(invoice.issued_at))}</div>
        </div>
      </header>

      <section class="meta-grid">
        <div class="card">
          <h2>Bill / ship to</h2>
          <p><strong>${escapeHTML(invoice.customer_name)}</strong><br />${escapeHTML(invoice.customer_phone)}${invoice.customer_email ? `<br />${escapeHTML(invoice.customer_email)}` : ""}<br /><br />${address}</p>
        </div>
        <div class="card">
          <h2>Order & payment</h2>
          <p><strong>Order #${escapeHTML(invoice.order_number)}</strong><br />Delivery: ${escapeHTML(humanizeSafe(invoice.delivery_method))}</p>
          <div class="status-row"><span class="status">${escapeHTML(humanizeSafe(invoice.order_status))}</span><span class="status">${escapeHTML(humanizeSafe(invoice.payment_status))}</span></div>
          <p class="muted">${escapeHTML(humanizeSafe(invoice.payment_method))} Â· ${paymentDetail}</p>
        </div>
      </section>

      <section class="items" aria-label="Invoice items">
        <table>
          <thead><tr><th>Item</th><th class="number">Qty</th><th class="number">Unit price</th><th class="number">Line total</th></tr></thead>
          <tbody>${itemRows}</tbody>
        </table>
      </section>

      <section class="bottom">
        <div class="note"><strong>Invoice record</strong>This document is generated from the confirmed order and its persistent invoice snapshot.</div>
        <div class="totals">
          <div class="row"><span>Subtotal</span><strong>${escapeHTML(formatMoney(invoice.subtotal_amount, invoice.currency))}</strong></div>
          <div class="row"><span>Discount</span><strong>âˆ’ ${escapeHTML(formatMoney(invoice.discount_amount, invoice.currency))}</strong></div>
          <div class="row"><span>Shipping</span><strong>${escapeHTML(formatMoney(invoice.shipping_amount, invoice.currency))}</strong></div>
          <div class="row total"><span>Total</span><span>${escapeHTML(formatMoney(invoice.total_amount, invoice.currency))}</span></div>
        </div>
      </section>

      <footer class="footer">
        <span><strong>Order:</strong> ${escapeHTML(invoice.order_number)}</span>
        <span><strong>Invoice:</strong> ${escapeHTML(invoice.invoice_number)}</span>
      </footer>
    </article>
  </main>
${autoPrint ? `<script>window.addEventListener("load", () => window.print());</script>` : ""}
</body>
</html>`;
}

export async function GET(request: Request, context: RouteContext) {
  const { orderId } = await context.params;
  const url = new URL(request.url);
  const wantsJSON = url.searchParams.get("format") === "json";
  const wantsDownload = url.searchParams.get("download") === "1";
  const wantsPrint = url.searchParams.get("print") === "1";

  try {
    const result = await accountCommerceFetch<AccountDataResponse<unknown>>(
      `/api/v1/orders/${encodeURIComponent(orderId)}/invoice`,
      { cache: "no-store" },
    );

    let fallbackOrder: OrderFallback | undefined;
    let rotatedTokens = result.rotatedTokens;

    if (needsOrderFallback(result.payload.data)) {
      try {
        const orderResult = await accountCommerceFetch<AccountDataResponse<OrderFallback>>(
          `/api/v1/orders/${encodeURIComponent(orderId)}`,
          { cache: "no-store" },
        );
        fallbackOrder = orderResult.payload.data;
        rotatedTokens = orderResult.rotatedTokens ?? rotatedTokens;
      } catch {
        // The invoice itself remains usable even if optional order enrichment fails.
      }
    }

    const invoice = normalizeInvoice(result.payload.data, fallbackOrder, orderId);

    if (wantsJSON) {
      const response = NextResponse.json({ data: invoice }, {
        status: 200,
        headers: {
          "Cache-Control": "private, no-store",
          "X-Robots-Tag": "noindex, nofollow",
          "X-ENE-Invoice-Route-Version": "runtime-v3",
        },
      });

      if (rotatedTokens) {
        setAccountAuthCookies(response, rotatedTokens);
      }

      return response;
    }

    const response = new NextResponse(
      buildInvoiceHTML(invoice, wantsPrint),
      {
        status: 200,
        headers: {
          "Content-Type": "text/html; charset=utf-8",
          "Cache-Control": "private, no-store",
          "Content-Disposition": `${wantsDownload ? "attachment" : "inline"}; filename="${safeInvoiceFilename(invoice.invoice_number)}.html"`,
          "X-Robots-Tag": "noindex, nofollow",
          "X-ENE-Invoice-Route-Version": "runtime-v3",
        },
      },
    );

    if (rotatedTokens) {
      setAccountAuthCookies(response, rotatedTokens);
    }

    return response;
  } catch (error) {
    return accountRouteErrorResponse(error, "Unable to load invoice.");
  }
}

