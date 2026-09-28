export type AdminInventoryStockItem = {
  variant_id: string;
  sku: string;

  product_id: string;
  product_code: string;
  product_name: string;

  quantity_on_hand: number;
  quantity_reserved: number;
  available_quantity: number;

  reorder_level: number;
  low_stock: boolean;

  updated_at: string;
};

export type AdminInventoryStockList = {
  items: AdminInventoryStockItem[];
  limit: number;
  offset: number;
};

export type AdminInventoryListResponse = {
  data: AdminInventoryStockList;
};

export type AdminInventoryItemResponse = {
  data: AdminInventoryStockItem;
};

export type AdminInventoryMovement = {
  id: string;

  variant_id: string;

  reservation_id?: string;

  movement_type: string;

  quantity_on_hand_delta: number;
  quantity_reserved_delta: number;

  quantity_on_hand_after: number;
  quantity_reserved_after: number;

  reference_type?: string;
  reference_id?: string;

  reason?: string;
  note?: string;

  actor_type?: string;
  actor_id?: string;

  created_at: string;
};

export type AdminInventoryMovementList = {
  items: AdminInventoryMovement[];
  limit: number;
  offset: number;
};

export type AdminInventoryMovementListResponse = {
  data: AdminInventoryMovementList;
};

export type AdminInventoryAdjustmentRequest = {
  quantity_delta: number;
  reason: string;
  note?: string;
};

export type AdminInventoryAdjustmentResponse = {
  data: AdminInventoryStockItem;
};