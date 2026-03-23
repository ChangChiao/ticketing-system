"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";

export default function Navbar() {
  const pathname = usePathname();

  return (
    <nav className="flex items-center justify-between h-16 px-12 bg-[var(--bg-card)]">
      <Link href="/" className="flex items-center gap-2">
        <span className="font-display text-xl font-semibold text-[var(--text-primary)] tracking-wider">
          TICKETPULSE
        </span>
      </Link>

      <div className="flex items-center gap-6">
        <Link
          href="/"
          className={`font-mono text-[13px] ${
            pathname === "/"
              ? "text-[var(--accent-orange)]"
              : "text-[var(--text-secondary)] hover:text-[var(--text-primary)]"
          } transition-colors`}
        >
          // events
        </Link>
        <Link
          href="/orders"
          className={`font-mono text-[13px] ${
            pathname.startsWith("/orders")
              ? "text-[var(--accent-orange)]"
              : "text-[var(--text-secondary)] hover:text-[var(--text-primary)]"
          } transition-colors`}
        >
          // my_orders
        </Link>
      </div>
    </nav>
  );
}
