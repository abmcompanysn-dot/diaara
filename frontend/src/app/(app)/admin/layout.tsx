'use client';

import { RequireAdmin } from '@/lib/guards';

export default function AdminLayout({ children }: { children: React.ReactNode }) {
  return <RequireAdmin>{children}</RequireAdmin>;
}
