'use client';

import { RequireRole } from '@/lib/guards';

export default function VendorLayout({ children }: { children: React.ReactNode }) {
  return <RequireRole roles={['vendeur']}>{children}</RequireRole>;
}
