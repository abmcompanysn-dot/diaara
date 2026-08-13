import type { SVGProps } from 'react';

type IconProps = SVGProps<SVGSVGElement> & { size?: number };

const base = (props: IconProps): IconProps => {
  const { size = 24, ...rest } = props;
  return {
    xmlns: 'http://www.w3.org/2000/svg',
    width: size,
    height: size,
    viewBox: '0 0 24 24',
    fill: 'none',
    stroke: 'currentColor',
    strokeWidth: 2,
    strokeLinecap: 'round',
    strokeLinejoin: 'round',
    ...rest,
  } as IconProps;
};

export const CartIcon = (props: IconProps) => (
  <svg {...base(props)}>
    <circle cx="9" cy="21" r="1.5" />
    <circle cx="19" cy="21" r="1.5" />
    <path d="M2.5 3h2l2.6 12.4a2 2 0 0 0 2 1.6h8.7a2 2 0 0 0 2-1.6L21.5 7H6" />
  </svg>
);

export const PackageIcon = (props: IconProps) => (
  <svg {...base(props)}>
    <path d="M21 8.5v7a2 2 0 0 1-1 1.7l-6.5 3.8a2 2 0 0 1-2 0L5 17.2a2 2 0 0 1-1-1.7v-7a2 2 0 0 1 1-1.7l6.5-3.8a2 2 0 0 1 2 0L20 6.8a2 2 0 0 1 1 1.7z" />
    <path d="M12 22V11" />
    <path d="M3.5 6.5 12 12l8.5-5.5" />
  </svg>
);

export const StoreIcon = (props: IconProps) => (
  <svg {...base(props)}>
    <path d="M4 9v10a1 1 0 0 0 1 1h14a1 1 0 0 0 1-1V9" />
    <path d="M3 5.5 4.6 3h14.8L21 5.5a2 2 0 0 1-2.3 2.4 2 2 0 0 1-1.6-1 2 2 0 0 1-3.3 0 2 2 0 0 1-3.3 0 2 2 0 0 1-3.3 0 2 2 0 0 1-1.6 1A2 2 0 0 1 3 5.5z" />
    <path d="M10 21v-5h4v5" />
  </svg>
);

export const WalletIcon = (props: IconProps) => (
  <svg {...base(props)}>
    <path d="M21 7V5a2 2 0 0 0-2-2H5a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-2" />
    <path d="M21 11h-4a2 2 0 0 0 0 4h4a1 1 0 0 0 1-1v-2a1 1 0 0 0-1-1z" />
  </svg>
);

export const HeadsetIcon = (props: IconProps) => (
  <svg {...base(props)}>
    <path d="M4 13v-1a8 8 0 0 1 16 0v1" />
    <rect x="2.5" y="13" width="4" height="6" rx="1.5" />
    <rect x="17.5" y="13" width="4" height="6" rx="1.5" />
    <path d="M20 19v0a4 4 0 0 1-4 3.5H13" />
  </svg>
);

export const HomeIcon = (props: IconProps) => (
  <svg {...base(props)}>
    <path d="M3 11.5 12 3l9 8.5" />
    <path d="M5 10v10a1 1 0 0 0 1 1h12a1 1 0 0 0 1-1V10" />
  </svg>
);

export const KeyIcon = (props: IconProps) => (
  <svg {...base(props)}>
    <circle cx="8" cy="15" r="4.5" />
    <path d="m11 12 9-9" />
    <path d="M17 6l2 2" />
  </svg>
);

export const UserIcon = (props: IconProps) => (
  <svg {...base(props)}>
    <circle cx="12" cy="8" r="4" />
    <path d="M4 21a8 8 0 0 1 16 0" />
  </svg>
);

export const BookIcon = (props: IconProps) => (
  <svg {...base(props)}>
    <path d="M4 4.5A2.5 2.5 0 0 1 6.5 2H20v17H6.5A2.5 2.5 0 0 0 4 21.5z" />
    <path d="M4 21.5A2.5 2.5 0 0 1 6.5 19H20" />
  </svg>
);

export const FileIcon = (props: IconProps) => (
  <svg {...base(props)}>
    <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z" />
    <path d="M14 2v6h6" />
    <path d="M9 13h6" />
    <path d="M9 17h6" />
  </svg>
);

export const MonitorIcon = (props: IconProps) => (
  <svg {...base(props)}>
    <rect x="2.5" y="4" width="19" height="12" rx="2" />
    <path d="M9 20h6" />
    <path d="M12 16v4" />
  </svg>
);

export const ZapIcon = (props: IconProps) => (
  <svg {...base(props)}>
    <path d="M13 2 4.5 13H11l-1.5 9L19 10h-6.5z" />
  </svg>
);

export const BriefcaseIcon = (props: IconProps) => (
  <svg {...base(props)}>
    <rect x="3" y="7.5" width="18" height="12" rx="2" />
    <path d="M9 7.5V5a2 2 0 0 1 2-2h2a2 2 0 0 1 2 2v2.5" />
    <path d="M3 13h18" />
  </svg>
);

export const CheckIcon = (props: IconProps) => (
  <svg {...base(props)}>
    <path d="m4.5 12.5 5 5 10-11" />
  </svg>
);

export const MegaphoneIcon = (props: IconProps) => (
  <svg {...base(props)}>
    <path d="M3 11v2a2 2 0 0 0 2 2h1l1 5h2l-1-5h1l6 3a1 1 0 0 0 1-1V6a1 1 0 0 0-1-1l-6 3H5a2 2 0 0 0-2 2z" />
  </svg>
);

export const ShieldIcon = (props: IconProps) => (
  <svg {...base(props)}>
    <path d="M12 2.5 4.5 5.5v6c0 4.6 3.2 8.3 7.5 10 4.3-1.7 7.5-5.4 7.5-10v-6z" />
    <path d="m9 11.5 2 2 4-4.5" />
  </svg>
);

export const LockIcon = (props: IconProps) => (
  <svg {...base(props)}>
    <rect x="4" y="10.5" width="16" height="10" rx="2" />
    <path d="M8 10.5V7.5a4 4 0 0 1 8 0v3" />
    <circle cx="12" cy="15.5" r="1" fill="currentColor" stroke="none" />
  </svg>
);

export const ArrowLeftIcon = (props: IconProps) => (
  <svg {...base(props)}>
    <path d="M19 12H5" />
    <path d="m11 18-6-6 6-6" />
  </svg>
);

export const ChevronRightIcon = (props: IconProps) => (
  <svg {...base(props)}>
    <path d="m9 6 6 6-6 6" />
  </svg>
);

export const EditIcon = (props: IconProps) => (
  <svg {...base(props)}>
    <path d="M17 3.5a2.1 2.1 0 0 1 3 3L8 18.5l-4 1 1-4z" />
    <path d="m15 5.5 3 3" />
  </svg>
);

export const LinkIcon = (props: IconProps) => (
  <svg {...base(props)}>
    <path d="M10 13a5 5 0 0 0 7.5.5l2.5-2.5a5 5 0 0 0-7-7l-1.5 1.5" />
    <path d="M14 11a5 5 0 0 0-7.5-.5L4 13a5 5 0 0 0 7 7l1.5-1.5" />
  </svg>
);

export const MenuIcon = (props: IconProps) => (
  <svg {...base(props)}>
    <path d="M4 6h16" />
    <path d="M4 12h16" />
    <path d="M4 18h16" />
  </svg>
);

export const XIcon = (props: IconProps) => (
  <svg {...base(props)}>
    <path d="M18 6 6 18" />
    <path d="m6 6 12 12" />
  </svg>
);

export const DownloadIcon = (props: IconProps) => (
  <svg {...base(props)}>
    <path d="M12 3v12" />
    <path d="m7 10 5 5 5-5" />
    <path d="M5 21h14" />
  </svg>
);

export const CopyIcon = (props: IconProps) => (
  <svg {...base(props)}>
    <rect x="9" y="9" width="12" height="12" rx="2" />
    <path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1" />
  </svg>
);

export const AlertTriangleIcon = (props: IconProps) => (
  <svg {...base(props)}>
    <path d="M12 3 2 20h20z" />
    <path d="M12 9v5" />
    <path d="M12 17.5v.5" />
  </svg>
);

export const TrashIcon = (props: IconProps) => (
  <svg {...base(props)}>
    <path d="M4 7h16" />
    <path d="M10 11v6" />
    <path d="M14 11v6" />
    <path d="M6 7l1 13a1.5 1.5 0 0 0 1.5 1.5h7A1.5 1.5 0 0 0 17 20l1-13" />
    <path d="M9 7V5a1.5 1.5 0 0 1 1.5-1.5h3A1.5 1.5 0 0 1 15 5v2" />
  </svg>
);
