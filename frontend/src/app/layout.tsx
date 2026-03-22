import type { Metadata } from "next";
import "./globals.css";

export const metadata: Metadata = {
  title: "演唱會售票系統",
  description: "線上演唱會售票平台",
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="zh-TW">
      <body className="min-h-screen bg-gray-50">{children}</body>
    </html>
  );
}
