import type { Metadata } from 'next';
import { Lora, DM_Sans } from 'next/font/google';
import { ThemeProvider } from '@/lib/theme';
import Navbar from '@/components/Navbar';
import './globals.css';

const lora = Lora({
  variable: '--font-lora',
  subsets: ['latin'],
  display: 'swap',
});

const dmSans = DM_Sans({
  variable: '--font-dm-sans',
  subsets: ['latin'],
  display: 'swap',
});

export const metadata: Metadata = {
  title: 'Verso — Social Reading',
  description: 'Track, discover, and discuss books you love.',
};

const ANTI_FOUC_SCRIPT = `
(function() {
  var t = localStorage.getItem('verso_theme');
  var dark = t === 'dark' || (!t && window.matchMedia('(prefers-color-scheme: dark)').matches);
  if (dark) document.documentElement.classList.add('dark');
})()
`;

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html
      lang="en"
      suppressHydrationWarning
      className={`${lora.variable} ${dmSans.variable} h-full antialiased`}
    >
      <head>
        <script dangerouslySetInnerHTML={{ __html: ANTI_FOUC_SCRIPT }} />
      </head>
        <body className="flex min-h-full flex-col font-[var(--font-dm-sans)] bg-[var(--color-cream)] text-[var(--color-ink)] dark:bg-[#0C0A09] dark:text-[#F5F5F4]">
        <ThemeProvider>
          <Navbar />
          <main className="flex-1 pt-14">{children}</main>
        </ThemeProvider>
      </body>
    </html>
  );
}
