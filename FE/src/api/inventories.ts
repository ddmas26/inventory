import { get, post, put, del } from './client';
import type { Inventory, PaginatedResponse } from '../types';

export const inventoriesApi = {
  list: (pageIndex = 1, pageSize = 20) =>
    get<PaginatedResponse<Inventory>>(`/inventories?page_index=${pageIndex}&page_size=${pageSize}`),

  getById: (id: string) =>
    get<Inventory>(`/inventories/${id}`),

  create: (data: { name: string; address?: string; latitude?: string; longitude?: string }) =>
    post<Inventory>('/inventories', data),

  update: (id: string, data: { name?: string; address?: string; latitude?: string; longitude?: string }) =>
    put<Inventory>(`/inventories/${id}`, data),

  delete: (id: string) =>
    del(`/inventories/${id}`),
};
