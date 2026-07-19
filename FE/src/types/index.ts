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
  id: string;
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

// --- Dashboard types ---

export interface DashboardCounts {
  total_value: number;
  active_products: number;
  locations_count: number;
  low_stock: number;
}

export interface DashboardLocation {
  id: string;
  name: string;
  lat: string;
  long: string;
}

export interface LowStockItem {
  id: string;
  product_name: string;
  inventory_name: string;
  quantity: number;
  stats: number; // StockStatus: 0=LOW, 1=NORMAL, 2=OK
}

export interface DashboardData {
  counts: DashboardCounts;
  inventories: DashboardLocation[];
  stocks: LowStockItem[];
}
