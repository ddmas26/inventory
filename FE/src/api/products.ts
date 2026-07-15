import { get, post, put, del } from './client';
import type { Product, PaginatedResponse } from '../types';

export const productsApi = {
  list: (pageIndex = 1, pageSize = 20) =>
    get<PaginatedResponse<Product>>(`/products?page_index=${pageIndex}&page_size=${pageSize}`),

  getById: (id: string) =>
    get<Product>(`/products/${id}`),

  create: (data: { name: string; description?: string; price?: number }) =>
    post<Product>('/products', data),

  update: (id: string, data: { name?: string; description?: string; price?: number }) =>
    put<Product>(`/products/${id}`, data),

  delete: (id: string) =>
    del(`/products/${id}`),
};
