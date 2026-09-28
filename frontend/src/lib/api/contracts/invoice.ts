// Location: src/lib/api/contracts/invoice.ts

export type CustomerInvoiceItem = {
  variant_id: string;
  sku: string;
  product_name: string;
  quantity: number;
  unit_price_amount: number;
  line_total_amount: number;
};

export type CustomerInvoice = {
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
  items: CustomerInvoiceItem[];
};

export type CustomerInvoiceResponse = {
  data: CustomerInvoice;
};
