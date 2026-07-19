import { get } from './client';
import type { DashboardData } from '../types';

export const dashboardApi = {
  get: () => get<DashboardData>('/inventories/dashboard'),
};
