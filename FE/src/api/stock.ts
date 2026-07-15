import { get, post } from './client';
import type { Stock, PaginatedResponse } from '../types';

export interface ListStockParams {
  inventory_id?: string;
  product_id?: string;
  search?: string;
  created_from?: string;
  created_to?: string;
  order_by?: string;
  sort?: string;
  page_index?: number;
  page_size?: number;
}

export const stockApi = {
  list: (params?: ListStockParams) => {
    const query = new URLSearchParams();
    if (params) {
      Object.entries(params).forEach(([key, value]) => {
        if (value !== undefined && value !== '') query.set(key, String(value));
      });
    }
    return get<PaginatedResponse<Stock>>(`/stock?${query.toString()}`);
  },

  add: (inventoryId: string, productId: string, quantity: number) =>
    post<Stock>('/stock/add', { inventory_id: inventoryId, product_id: productId, quantity }),

  deduct: (inventoryId: string, productId: string, quantity: number) =>
    post<Stock>('/stock/deduct', { inventory_id: inventoryId, product_id: productId, quantity }),

  set: (inventoryId: string, productId: string, quantity: number) =>
    post<Stock>('/stock/set', { inventory_id: inventoryId, product_id: productId, quantity }),

  remove: (inventoryId: string, productId: string) =>
    post<void>('/stock/remove', { inventory_id: inventoryId, product_id: productId }),

  transfer: (fromInventoryId: string, toInventoryId: string, productId: string, quantity: number) =>
    post<{ message: string }>('/stock/transfer', {
      from_inventory_id: fromInventoryId,
      to_inventory_id: toInventoryId,
      product_id: productId,
      quantity,
    }),

  listByProduct: (productId: string) =>
    get<Stock[]>(`/products/${productId}/inventories`),
};
