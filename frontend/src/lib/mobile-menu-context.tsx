'use client';

import { createContext, useContext, useState } from 'react';

interface MobileMenuContextValue {
  open: boolean;
  setOpen: (open: boolean) => void;
}

const MobileMenuContext = createContext<MobileMenuContextValue>({
  open: false,
  setOpen: () => {},
});

/** Partage l'état ouvert/fermé du tiroir de navigation entre le header et la barre du bas. */
export function MobileMenuProvider({ children }: { children: React.ReactNode }) {
  const [open, setOpen] = useState(false);
  return (
    <MobileMenuContext.Provider value={{ open, setOpen }}>{children}</MobileMenuContext.Provider>
  );
}

export function useMobileMenu() {
  return useContext(MobileMenuContext);
}