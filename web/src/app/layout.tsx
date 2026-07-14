import type { Metadata } from 'next';
import { Lora, DM_Sans } from 'next/font/google';
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

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="en" className={`${lora.variable} ${dmSans.variable} h-full antialiased`}>
      <body className="flex min-h-full flex-col font-[var(--font-dm-sans)]">
        <Navbar />
        <main className="flex-1 pt-14">{children}</main>
      </body>
    </html>
  );
}
