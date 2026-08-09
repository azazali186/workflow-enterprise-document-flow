import { useQuery } from '@tanstack/react-query';
import {
  CheckCircle2,
  Clock3,
  FileText,
  HardDrive,
  Hourglass,
  Users,
} from 'lucide-react';
import { Card, PageHeader } from '@/components/ui/Card';
import { StatCard } from './StatCard';
import { TrendChart } from './TrendChart';
import { StatSkeleton } from '@/components/ui/Skeleton';
import { ErrorState, EmptyState } from '@/components/ui/States';
import { Badge, statusTone } from '@/components/ui/Badge';
import { dashboardService } from '@/services/dashboard.service';
import { formatBytes, formatDateTime, humanize, timeAgo } from '@/lib/format';

const STATUS_ORDER = ['draft', 'pending_verification', 'verified', 'rejected', 'approved', 'archived'];

export function DashboardPage() {
  const report = useQuery({ queryKey: ['dashboard'], queryFn: () => dashboardService.report(14) });
  const workflow = useQuery({ queryKey: ['workflow'], queryFn: () => dashboardService.workflow(14) });

  if (report.isLoading || workflow.isLoading) {
    return (
      <div className="space-y-6">
        <PageHeader title="Dashboard" description="A live view of your document workflow." />
        <StatSkeleton />
        <div className="grid gap-4 lg:grid-cols-2">
          <div className="h-72 animate-pulse rounded-xl bg-ink-100" />
          <div className="h-72 animate-pulse rounded-xl bg-ink-100" />
        </div>
      </div>
    );
  }

  if (report.isError || workflow.isError) {
    return (
      <div className="space-y-6">
        <PageHeader title="Dashboard" />
        <Card>
          <ErrorState onRetry={() => { void report.refetch(); void workflow.refetch(); }} />
        </Card>
      </div>
    );
  }

  const data = report.data!;
  const wf = workflow.data!;

  const totalDocs = Object.values(data.documents ?? {}).reduce((a, b) => a + b, 0);
  const totalUsers = Object.values(data.users ?? {}).reduce((a, b) => a + b, 0);
  const totalStored = Object.values(data.storages ?? {}).reduce((a, b) => a + b, 0);

  return (
    <div className="space-y-6 animate-fade-up">
      <PageHeader title="Dashboard" description="A live view of your document workflow." />

      {/* Stat row */}
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-4">
        <StatCard label="Total documents" value={totalDocs.toLocaleString()} icon={FileText} tone="primary" sub="across all statuses" />
        <StatCard label="Pending approvals" value={data.pending_approvals?.toLocaleString() ?? '0'} icon={Hourglass} tone="warning" sub="awaiting a decision" />
        <StatCard label="Verified docs" value={(data.documents?.verified ?? 0).toLocaleString()} icon={CheckCircle2} tone="success" sub="authenticated" />
        <StatCard label="Storage used" value={formatBytes(data.total_storage_bytes)} icon={HardDrive} tone="neutral" sub={`${totalStored.toLocaleString()} stored objects`} />
      </div>

      <div className="grid gap-4 lg:grid-cols-5">
        {/* Trend */}
        <Card className="p-6 lg:col-span-3">
          <div className="mb-5 flex items-center justify-between">
            <div>
              <h3 className="font-display text-base font-semibold text-ink-950">Documents created</h3>
              <p className="text-xs text-ink-400">Last 14 days</p>
            </div>
            <Badge tone="primary">{totalUsers.toLocaleString()} users</Badge>
          </div>
          <TrendChart data={data.documents_per_day ?? {}} />
        </Card>

        {/* Status funnel */}
        <Card className="p-6 lg:col-span-2">
          <h3 className="font-display text-base font-semibold text-ink-950">Documents by status</h3>
          <p className="text-xs text-ink-400">Current distribution</p>
          <ul className="mt-5 space-y-3">
            {STATUS_ORDER.map((status) => {
              const count = data.documents?.[status] ?? 0;
              const pct = totalDocs > 0 ? Math.round((count / totalDocs) * 100) : 0;
              return (
                <li key={status}>
                  <div className="mb-1 flex items-center justify-between text-[13px]">
                    <span className="flex items-center gap-2 text-ink-600">
                      <Badge tone={statusTone(status)}>{humanize(status)}</Badge>
                    </span>
                    <span className="tabular text-ink-800">{count}</span>
                  </div>
                  <div className="h-1.5 overflow-hidden rounded-full bg-ink-100">
                    <div
                      className="h-full rounded-full bg-primary-500 transition-all duration-500"
                      style={{ width: `${pct}%` }}
                    />
                  </div>
                </li>
              );
            })}
          </ul>
          <div className="mt-6 grid grid-cols-2 gap-3 border-t border-ink-100 pt-5">
            <div>
              <p className="flex items-center gap-1.5 text-xs text-ink-400">
                <Clock3 className="size-3.5" /> Pending verification
              </p>
              <p className="mt-1 font-display text-2xl font-semibold tabular text-ink-950">
                {wf.pending_verifications ?? 0}
              </p>
            </div>
            <div>
              <p className="flex items-center gap-1.5 text-xs text-ink-400">
                <Hourglass className="size-3.5" /> Pending approvals
              </p>
              <p className="mt-1 font-display text-2xl font-semibold tabular text-ink-950">
                {wf.pending_approvals ?? 0}
              </p>
            </div>
          </div>
        </Card>
      </div>

      {/* Recent activity */}
      <Card>
        <div className="border-b border-ink-100 px-6 py-4">
          <h3 className="font-display text-base font-semibold text-ink-950">Recent activity</h3>
        </div>
        {!data.recent_activity?.length ? (
          <EmptyState title="No activity yet" description="Actions across documents, users, and roles will appear here." />
        ) : (
          <ul className="divide-y divide-ink-100/80">
            {data.recent_activity.map((log) => (
              <li key={log.id} className="flex items-center gap-3 px-6 py-3.5">
                <div className="flex size-8 shrink-0 items-center justify-center rounded-full bg-paper-100 text-ink-400">
                  <Users className="size-4" aria-hidden />
                </div>
                <div className="min-w-0 flex-1">
                  <p className="truncate text-sm text-ink-800">
                    <span className="font-medium">{log.actor_email || 'System'}</span>{' '}
                    <span className="text-ink-400">{log.action}</span> {log.entity}
                  </p>
                  <p className="text-xs text-ink-400">{log.entity_id ? `#${log.entity_id.slice(0, 8)}` : ''}</p>
                </div>
                <div className="shrink-0 text-right">
                  <p className="text-xs tabular text-ink-400">{timeAgo(log.created_at)}</p>
                  <p className="text-[11px] text-ink-300">{formatDateTime(log.created_at)}</p>
                </div>
              </li>
            ))}
          </ul>
        )}
      </Card>
    </div>
  );
}
