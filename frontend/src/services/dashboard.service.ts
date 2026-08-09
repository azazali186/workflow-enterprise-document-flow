import { post } from './http';
import type { DashboardData, WorkflowData } from '@/types/entities';

export interface WindowInput {
  days?: number;
}

export const dashboardService = {
  report(days = 14): Promise<DashboardData> {
    return post<DashboardData>('/api/v1/reports/dashboard', { days });
  },

  workflow(days = 14): Promise<WorkflowData> {
    return post<WorkflowData>('/api/v1/analytics/workflow', { days });
  },
};
