export interface Product {
  id: string;
  name: string;
  description: string;
  price: number;
  created_at: string;
  updated_at: string;
  deleted_at?: string;
}

export interface Inventory {
  id: string;
  name: string;
  address: string;
  latitude: string;
  longitude: string;
  created_at: string;
  updated_at: string;
  deleted_at?: string;
}

export interface Stock {
  inventory_id: string;
  product_id: string;
  product_name?: string;
  inventory_name?: string;
  quantity: number;
  created_at: string;
  updated_at: string;
}

export interface PaginatedResponse<T> {
  data: T[];
  total: number;
  page_index: number;
  page_size: number;
}
