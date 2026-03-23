import type { Metadata } from "next";
import "./globals.css";

export const metadata: Metadata = {
  title: "TICKETPULSE — 演唱會售票系統",
  description: "線上演唱會售票平台",
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="zh-TW" className="h-full">
      <head>
        <link
          href="https://fonts.googleapis.com/css2?family=Oswald:wght@400;600;700&family=JetBrains+Mono:wght@400;600&display=swap"
          rel="stylesheet"
        />
      </head>
      <body className="h-full bg-[var(--bg-page)] text-[var(--text-primary)]">
        {children}
      </body>
    </html>
  );
}
