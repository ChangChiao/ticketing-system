"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import Navbar from "@/components/Navbar";

interface EventItem {
  id: string;
  title: string;
  event_date: string;
  venue_name: string;
  price_range: string;
  sale_status: string;
  sale_start: string;
  image_url: string;
}

const gradients = [
  "from-[#FF6B35] to-[#1A1A1A]",
  "from-[#00D4AA] to-[#1A1A1A]",
  "from-[#eab308] to-[#1A1A1A]",
  "from-[#ef4444] to-[#1A1A1A]",
  "from-[#7c3aed] to-[#1A1A1A]",
];

function getStatusBadge(status: string) {
  switch (status) {
    case "熱賣中":
      return { label: "ON SALE", bg: "bg-[var(--accent-teal)]" };
    case "即將開賣":
      return { label: "即將開賣", bg: "bg-[var(--accent-orange)]" };
    case "已售完":
      return { label: "SOLD OUT", bg: "bg-[var(--status-grey)]" };
    default:
      return { label: status, bg: "bg-[var(--bg-placeholder)]" };
  }
}

export default function HomePage() {
  const [events, setEvents] = useState<EventItem[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    fetch("/api/events")
      .then((res) => res.json())
      .then((data) => setEvents(data.events || []))
      .catch(() => setEvents([]))
      .finally(() => setLoading(false));
  }, []);

  if (loading) {
    return (
      <div className="flex flex-col h-full">
        <Navbar />
        <div className="flex-1 flex items-center justify-center">
          <span className="font-mono text-[var(--text-secondary)]">// loading...</span>
        </div>
      </div>
    );
  }

  return (
    <div className="flex flex-col h-full">
      <Navbar />
      <main className="flex-1 px-12 py-10">
        <div className="flex items-end justify-between mb-8">
          <div>
            <h1 className="font-display text-4xl font-bold tracking-tight">
              UPCOMING EVENTS
            </h1>
            <p className="font-mono text-[13px] text-[var(--text-secondary)] mt-1">
              // browse_available_concerts
            </p>
          </div>
          <div className="flex items-center gap-2 bg-[var(--bg-card)] rounded-[var(--radius)] px-4 h-11 w-80">
            <svg className="w-4 h-4 text-[var(--text-secondary)]" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <circle cx="11" cy="11" r="8" /><path d="m21 21-4.3-4.3" />
            </svg>
            <span className="font-mono text-[13px] text-[var(--text-secondary)]">
              search_events...
            </span>
          </div>
        </div>

        {events.length === 0 ? (
          <div className="bg-[var(--bg-card)] rounded-[var(--radius)] p-12 text-center">
            <p className="font-mono text-[var(--text-secondary)]">// no_events_found</p>
          </div>
        ) : (
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
            {events.map((event, i) => {
              const badge = getStatusBadge(event.sale_status);
              return (
                <Link
                  key={event.id}
                  href={`/events/${event.id}`}
                  className="bg-[var(--bg-card)] rounded-[var(--radius)] overflow-hidden hover:ring-1 hover:ring-[var(--accent-orange)] transition-all group"
                >
                  <div
                    className={`h-[180px] bg-gradient-to-br ${gradients[i % gradients.length]} flex items-end p-5`}
                  >
                    <span className="font-display text-lg font-semibold text-[var(--text-primary)] opacity-0 group-hover:opacity-100 transition-opacity">
                      {event.title}
                    </span>
                  </div>
                  <div className="p-5 flex flex-col gap-3">
                    <div>
                      <h2 className="font-display text-lg font-semibold leading-tight">
                        {event.title}
                      </h2>
                      <div className="flex gap-4 mt-1.5">
                        <span className="font-mono text-xs text-[var(--text-secondary)]">
                          {new Date(event.event_date).toLocaleDateString("zh-TW")}
                        </span>
                        <span className="font-mono text-xs text-[var(--text-secondary)]">
                          {event.venue_name}
                        </span>
                      </div>
                    </div>
                    <div className="flex items-center justify-between">
                      <span className="font-mono text-[13px] font-semibold text-[var(--accent-orange)]">
                        NT$ {event.price_range}
                      </span>
                      <span
                        className={`${badge.bg} text-[var(--text-on-accent)] font-mono text-[11px] font-semibold px-3 py-1 rounded-lg`}
                      >
                        {badge.label}
                      </span>
                    </div>
                  </div>
                </Link>
              );
            })}
          </div>
        )}
      </main>
    </div>
  );
}
