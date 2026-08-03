/** @type {import('tailwindcss').Config} */
module.exports = {
  content: [
    './src/pages/**/*.{js,ts,jsx,tsx,mdx}',
    './src/components/**/*.{js,ts,jsx,tsx,mdx}',
    './src/app/**/*.{js,ts,jsx,tsx,mdx}',
  ],
  theme: {
    extend: {
      colors: {
        // Vert = couleur de l'argent, de la croissance et du mobile money en Afrique
        green: {
          950: '#052018',
          900: '#0A3225',
          800: '#0E4431',
          700: '#115E40',
          600: '#0F7A50',
          500: '#10A05F',
          400: '#17C26F',
          300: '#46E092',
          200: '#8AF0BB',
          100: '#C6F8E0',
          50: '#EBFBF2',
        },
        lime: {
          DEFAULT: '#C9F22E',
          dark: '#A8CF18',
        },
        paper: {
          DEFAULT: '#F2F7F4',
          cream: '#FAFDFB',
        },
        money: {
          orange: '#FF7A3D',
          wave: '#17C26F',
          om: '#FF6A00',
          mtn: '#FFC602',
        },
      },
      fontFamily: {
        display: ['Bricolage Grotesque', 'sans-serif'],
        body: ['Work Sans', 'sans-serif'],
        mono: ['Space Mono', 'monospace'],
      },
      boxShadow: {
        card: '0 1px 2px rgba(5,32,24,0.06), 0 8px 24px rgba(5,32,24,0.08)',
        lift: '0 2px 4px rgba(5,32,24,0.08), 0 16px 40px rgba(5,32,24,0.16)',
        glow: '0 0 0 3px rgba(23,194,111,0.25)',
      },
    },
  },
  plugins: [],
};
