import Link from 'next/link';
import { LockIcon, CheckIcon } from '@/components/icons';

const OPERATORS = ['Wave', 'Orange Money', 'MTN MoMo', 'Free'];

const TERMINAL_LINES = [
  { label: '> connexion sécurisée', value: 'ok' },
  { label: '> chiffrement TLS', value: 'actif' },
  { label: '> paiement mobile money', value: 'vérifié' },
  { label: '> livraison', value: 'instantanée' },
];

export function AuthShell({ children }: { children: React.ReactNode }) {
  return (
    <main className="min-h-[calc(100vh-4rem)] grid lg:grid-cols-2">
      {/* Panneau coffre */}
      <div className="relative gradient-green text-white overflow-hidden hidden lg:flex flex-col justify-between p-10">
        <div className="grid-pattern absolute inset-0" aria-hidden />
        <div className="wax-pattern absolute inset-0" aria-hidden />
        <div className="absolute bottom-0 right-0 w-96 h-96 bg-lime/15 rounded-full blur-3xl" aria-hidden />

        <div className="relative flex items-center gap-2">
          <span className="font-display text-2xl font-bold tracking-tight">DIARRA</span>
          <span className="w-2 h-2 rounded-full bg-green-400 inline-block" aria-hidden />
        </div>

        <div className="relative max-w-md">
          <p className="font-mono text-sm text-green-300 uppercase tracking-widest mb-4">
            // coffre-fort numérique
          </p>
          <div className="flex items-center gap-4">
            <span className="w-14 h-14 rounded-xl bg-lime/15 border border-lime/40 flex items-center justify-center shrink-0" aria-hidden>
              <LockIcon size={28} className="text-lime" />
            </span>
            <h2 className="font-display text-3xl font-bold leading-tight tracking-tight">
              Votre espace est
              <br />
              <span className="underline-accent text-lime">verrouillé</span>.
            </h2>
          </div>

          <div className="mt-8 bg-white/5 backdrop-blur rounded-xl p-5 border border-white/10 font-mono text-sm">
            {TERMINAL_LINES.map((line, i) => (
              <p
                key={line.label}
                className={`flex justify-between py-2 ${
                  i === 0 ? 'border-b border-white/10' : ''
                }`}
              >
                <span className="text-white/60">{line.label}</span>
                <span className="text-lime flex items-center gap-1.5">
                  <CheckIcon size={14} />
                  {line.value}
                </span>
              </p>
            ))}
            <p className="receipt-cursor py-2 text-green-300">{'> vérification en cours'}</p>
          </div>
        </div>

        <div className="relative">
          <p className="text-xs text-white/50 mb-2.5">Paiements acceptés :</p>
          <div className="flex flex-wrap gap-2">
            {OPERATORS.map((p) => (
              <span
                key={p}
                className="px-2.5 py-1 rounded bg-white/10 border border-white/15 text-xs font-bold text-white/90"
              >
                {p}
              </span>
            ))}
          </div>
          <p className="mt-5 font-mono text-xs text-white/40">
            Wave ✦ Orange Money ✦ MTN MoMo ✦ Free — via PawaPay
          </p>
        </div>
      </div>

      {/* Formulaire */}
      <div className="flex items-center justify-center bg-paper px-4 py-16 relative">
        <div className="grid-pattern-dark absolute inset-0" aria-hidden />
        <div className="relative w-full max-w-md">{children}</div>
      </div>
    </main>
  );
}
