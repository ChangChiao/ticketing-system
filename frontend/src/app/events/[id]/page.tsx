"use client";

import { useEffect, useState } from "react";
import { useParams, useRouter } from "next/navigation";
import { api, type EventDetail } from "@/lib/api";
import Navbar from "@/components/Navbar";

export default function EventDetailPage() {
  const params = useParams();
  const router = useRouter();
  const [event, setEvent] = useState<EventDetail | null>(null);
  const [loading, setLoading] = useState(true);
  const [days, setDays] = useState(0);
  const [hours, setHours] = useState(0);
  const [minutes, setMinutes] = useState(0);
  const [showCountdown, setShowCountdown] = useState(false);

  const eventId = params.id as string;

  useEffect(() => {
    api
      .getEvent(eventId)
      .then(setEvent)
      .catch(() => setEvent(null))
      .finally(() => setLoading(false));
  }, [eventId]);

  useEffect(() => {
    if (!event) return;
    const saleStart = new Date(event.sale_start).getTime();

    const timer = setInterval(() => {
      const diff = saleStart - Date.now();
      if (diff <= 0) {
        setShowCountdown(false);
        clearInterval(timer);
        if (event.sale_status !== "已售完" && event.sale_status !== "已結束" && event.status !== "ended") {
          router.push(`/events/${eventId}/queue`);
        }
        return;
      }
      setShowCountdown(true);
      setDays(Math.floor(diff / (1000 * 60 * 60 * 24)));
      setHours(Math.floor((diff % (1000 * 60 * 60 * 24)) / (1000 * 60 * 60)));
      setMinutes(Math.floor((diff % (1000 * 60 * 60)) / (1000 * 60)));
    }, 1000);

    return () => clearInterval(timer);
  }, [event, eventId, router]);

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

  if (!event) {
    return (
      <div className="flex flex-col h-full">
        <Navbar />
        <div className="flex-1 flex items-center justify-center">
          <span className="font-mono text-[var(--text-secondary)]">// event_not_found</span>
        </div>
      </div>
    );
  }

  const isSaleStarted = new Date(event.sale_start).getTime() <= Date.now();
  const isSoldOut = event.sale_status === "已售完";
  const isEnded = event.sale_status === "已結束" || event.status === "ended";
  const canBuy = isSaleStarted && !isSoldOut && !isEnded;

  function getSectionDotColor(section: { remaining: number; quota: number }) {
    if (section.remaining === 0) return "bg-[var(--status-grey)]";
    const ratio = section.remaining / section.quota;
    if (ratio > 0.5) return "bg-[var(--status-green)]";
    if (ratio > 0.1) return "bg-[var(--status-yellow)]";
    return "bg-[var(--status-red)]";
  }

  return (
    <div className="flex flex-col h-full">
      <Navbar />
      <main className="flex-1 flex gap-8 px-12 py-10 overflow-hidden">
        {/* Left column */}
        <div className="flex-1 flex flex-col gap-6 overflow-auto">
          {/* Hero */}
          <div className="relative h-[300px] rounded-[var(--radius)] overflow-hidden bg-gradient-to-br from-[#FF6B35] via-[#2D2D2D] to-[#1A1A1A]">
            <div className="absolute inset-0 flex flex-col justify-end p-7 gap-2">
              <h1 className="font-display text-[28px] font-bold">{event.title}</h1>
              <p className="font-mono text-[13px] text-[var(--text-secondary)]">
                // {new Date(event.event_date).toLocaleDateString("zh-TW")} — {event.venue_name}
              </p>
            </div>
          </div>

          {/* Event info */}
          <div className="bg-[var(--bg-card)] rounded-[var(--radius)] p-5 flex flex-col gap-3">
            <span className="font-display text-base font-semibold text-[var(--accent-orange)]">
              [EVENT_INFO]
            </span>
            <p className="font-mono text-[13px] text-[var(--text-secondary)] leading-relaxed">
              {(event as EventDetail & { description?: string }).description || "活動詳細資訊"}
            </p>
          </div>

          {/* Detail cards */}
          <div className="flex gap-4">
            <div className="flex-1 bg-[var(--bg-card)] rounded-[var(--radius)] p-4 flex flex-col gap-1.5">
              <span className="font-mono text-[11px] text-[var(--text-secondary)]">// date</span>
              <span className="font-mono text-[13px] font-semibold">
                {new Date(event.event_date).toLocaleDateString("zh-TW", {
                  year: "numeric", month: "long", day: "numeric", weekday: "short",
                })}
              </span>
            </div>
            <div className="flex-1 bg-[var(--bg-card)] rounded-[var(--radius)] p-4 flex flex-col gap-1.5">
              <span className="font-mono text-[11px] text-[var(--text-secondary)]">// time</span>
              <span className="font-mono text-[13px] font-semibold">19:30 入場 / 20:00 開演</span>
            </div>
            <div className="flex-1 bg-[var(--bg-card)] rounded-[var(--radius)] p-4 flex flex-col gap-1.5">
              <span className="font-mono text-[11px] text-[var(--text-secondary)]">// venue</span>
              <span className="font-mono text-[13px] font-semibold">{event.venue_name}</span>
            </div>
          </div>
        </div>

        {/* Right sidebar */}
        <div className="w-[380px] flex flex-col gap-5 shrink-0 overflow-auto">
          {/* Countdown */}
          {showCountdown && (
            <div className="bg-[var(--bg-card)] rounded-[var(--radius)] p-6 flex flex-col items-center gap-4">
              <span className="font-display text-base font-semibold text-[var(--accent-orange)]">
                [SALE_COUNTDOWN]
              </span>
              <div className="flex items-center gap-3">
                {[
                  { val: String(days).padStart(2, "0"), label: "DAYS" },
                  { val: String(hours).padStart(2, "0"), label: "HRS" },
                  { val: String(minutes).padStart(2, "0"), label: "MIN" },
                ].map((item, i) => (
                  <div key={item.label} className="flex items-center gap-3">
                    {i > 0 && (
                      <span className="font-display text-4xl font-bold text-[var(--text-secondary)]">:</span>
                    )}
                    <div className="w-16 bg-[var(--bg-elevated)] rounded-xl p-3 flex flex-col items-center gap-1">
                      <span className="font-display text-4xl font-bold">{item.val}</span>
                      <span className="font-mono text-[9px] font-semibold text-[var(--text-secondary)]">
                        {item.label}
                      </span>
                    </div>
                  </div>
                ))}
              </div>
            </div>
          )}

          {/* Ticket sections */}
          <div className="bg-[var(--bg-card)] rounded-[var(--radius)] p-5 flex flex-col gap-3">
            <span className="font-display text-base font-semibold text-[var(--accent-orange)]">
              [TICKET_SECTIONS]
            </span>
            {event.sections.map((section) => (
              <div
                key={section.id}
                className="flex items-center justify-between bg-[var(--bg-elevated)] rounded-xl px-4 py-2.5"
              >
                <div className="flex items-center gap-2">
                  <div className={`w-2 h-2 rounded-full ${getSectionDotColor(section)}`} />
                  <span className="font-mono text-[13px] font-semibold">
                    {section.section_name}
                  </span>
                </div>
                <span className="font-mono text-[13px] font-semibold text-[var(--accent-orange)]">
                  NT$ {section.price.toLocaleString()}
                </span>
              </div>
            ))}
          </div>

          {/* Buy button */}
          <button
            onClick={() => router.push(`/events/${eventId}/queue`)}
            disabled={!canBuy}
            className={`w-full h-[52px] rounded-[var(--radius)] font-display text-lg font-semibold tracking-wide transition flex items-center justify-center gap-2 ${
              canBuy
                ? "bg-[var(--accent-orange)] text-[var(--text-on-accent)] hover:brightness-110"
                : "bg-[var(--bg-placeholder)] text-[var(--text-secondary)] cursor-not-allowed"
            }`}
          >
            {canBuy ? "立即購票" : isSoldOut ? "已售完" : isEnded ? "已結束" : "尚未開賣"}
          </button>
          <p className="font-mono text-[11px] text-[var(--text-secondary)] text-center">
            // max 4 tickets per transaction
          </p>
        </div>
      </main>
    </div>
  );
}
