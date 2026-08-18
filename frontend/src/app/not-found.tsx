import Link from 'next/link';

export const metadata = {
  title: 'Page introuvable',
};

export default function NotFound() {
  return (
    <section className="gradient-green text-white relative overflow-hidden min-h-[70vh] flex items-center">
      <div className="wax-pattern absolute inset-0" aria-hidden />
      <div className="relative max-w-2xl mx-auto px-4 py-20 text-center">
        <p className="font-mono text-sm text-green-300 uppercase tracking-widest mb-4">
          // erreur 404
        </p>
        <h1 className="font-display text-5xl sm:text-6xl font-bold tracking-tight">
          Cette page s&apos;est vendue.
        </h1>
        <p className="mt-5 text-white/75 max-w-md mx-auto">
          Le lien est cassé ou la page a été déplacée. Le catalogue, lui, est toujours là.
        </p>
        <div className="mt-8 flex flex-wrap justify-center gap-4">
          <Link
            href="/catalog"
            className="px-6 py-3 rounded-full bg-lime text-green-950 font-semibold hover:bg-green-300 transition-colors"
          >
            Voir le catalogue
          </Link>
          <Link
            href="/"
            className="px-6 py-3 rounded-full border border-white/30 text-white hover:border-lime hover:text-lime transition-colors"
          >
            Retour à l&apos;accueil
          </Link>
        </div>
      </div>
    </section>
  );
}
