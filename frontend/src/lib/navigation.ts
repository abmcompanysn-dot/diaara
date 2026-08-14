import type { ComponentType, SVGProps } from 'react';
import {
  HomeIcon,
  CartIcon,
  PackageIcon,
  StoreIcon,
  WalletIcon,
  MegaphoneIcon,
  ShieldIcon,
  HeadsetIcon,
  UserIcon,
} from '@/components/icons';

export type IconType = ComponentType<SVGProps<SVGSVGElement> & { size?: number }>;

export type NavSection = 'principal' | 'vendeur' | 'closer' | 'admin';

export interface NavItem {
  href: string;
  label: string;
  Icon: IconType;
  section: NavSection;
  /** Rôles requis (client = implicite). */
  roles?: string[];
  /** Accès réservé aux administrateurs. */
  admin?: boolean;
  /** Visible pour tout utilisateur connecté. */
  always?: boolean;
}

export interface UserLike {
  is_admin?: boolean;
  roles?: string[];
}

/** Ordre et libellés des groupes de la sidebar. */
export const NAV_SECTIONS: { id: NavSection; label: string | null }[] = [
  { id: 'principal', label: null },
  { id: 'vendeur', label: 'Espace vendeur' },
  { id: 'closer', label: 'Affiliation' },
  { id: 'admin', label: 'Administration' },
];

/** Source de vérité unique : liens de l'espace connecté, filtrés par rôle. */
export const NAV_ITEMS: NavItem[] = [
  { href: '/dashboard', label: 'Mon espace', Icon: HomeIcon, section: 'principal', always: true },
  { href: '/catalog', label: 'Catalogue', Icon: CartIcon, section: 'principal', always: true },
  { href: '/orders', label: 'Mes commandes', Icon: PackageIcon, section: 'principal', always: true },
  { href: '/vendor', label: 'Espace vendeur', Icon: StoreIcon, section: 'vendeur', roles: ['vendeur'] },
  { href: '/vendor/earnings', label: 'Mes revenus', Icon: WalletIcon, section: 'vendeur', roles: ['vendeur'] },
  { href: '/closer/dashboard', label: 'Affiliation', Icon: MegaphoneIcon, section: 'closer', roles: ['closer'] },
  { href: '/admin', label: 'Administration', Icon: ShieldIcon, section: 'admin', admin: true },
  { href: '/support', label: 'Support', Icon: HeadsetIcon, section: 'principal', always: true },
  { href: '/account', label: 'Mon compte', Icon: UserIcon, section: 'principal', always: true },
];

/** Filtre les liens visibles selon le rôle de l'utilisateur connecté. */
export function visibleNavItems(user: UserLike | null): NavItem[] {
  if (!user) return [];
  return NAV_ITEMS.filter((item) => {
    if (item.always) return true;
    if (item.admin) return !!user.is_admin;
    if (item.roles) return item.roles.some((r) => user.roles?.includes(r));
    return true;
  });
}

/** Libellés lisibles des rôles d'un utilisateur (pour le badge de statut). */
export function userRoleLabels(user: UserLike | null): string[] {
  if (!user) return [];
  const labels: string[] = [];
  if (user.is_admin) labels.push('Admin');
  if (user.roles?.includes('vendeur')) labels.push('Vendeur');
  if (user.roles?.includes('closer')) labels.push('Affilié');
  if (labels.length === 0) labels.push('Client');
  return labels;
}
