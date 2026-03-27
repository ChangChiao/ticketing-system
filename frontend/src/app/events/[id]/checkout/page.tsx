"use client";

import { useEffect, useState } from "react";
import { useParams, useRouter } from "next/navigation";
import { useAuthStore } from "@/stores/auth";
import { api, type SeatInfo } from "@/lib/api";
import Navbar from "@/components/Navbar";

interface AllocationData {
  session_id: string;
  seats: SeatInfo[];
  expires_at: string;
  event_id: string;
  price_per_seat: number;
}

export default function CheckoutPage() {
  const params = useParams();
  const router = useRouter();
  const eventId = params.id as string;
  const token = useAuthStore((s) => s.token);

  const [allocation, setAllocation] = useState<AllocationData | null>(null);
  const [countdown, setCountdown] = useState(0);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState("");

  useEffect(() => {
    const stored = sessionStorage.getItem("allocation");
    if (!stored) {
      router.push(`/events/${eventId}/select`);
      return;
    }
    setAllocation(JSON.parse(stored));
  }, [eventId, router]);

  useEffect(() => {
    if (!allocation) return;

    const expiresAt = new Date(allocation.expires_at).getTime();

    const timer = setInterval(() => {
      const remaining = Math.max(0, Math.floor((expiresAt - Date.now()) / 1000));
      setCountdown(remaining);

      if (remaining === 0) {
        clearInterval(timer);
        sessionStorage.removeItem("allocation");
        router.push(`/events/${eventId}/select`);
      }
    }, 1000);

    return () => clearInterval(timer);
  }, [allocation, eventId, router]);

  const handleCheckout = async () => {
    if (!allocation || !token) return;
    setSubmitting(true);
    setError("");

    try {
      const result = await api.createOrder({
        event_id: allocation.event_id,
        seats: allocation.seats,
        price_per_seat: allocation.price_per_seat,
      });

      sessionStorage.setItem("pending_order_id", result.id);
      sessionStorage.removeItem("allocation");

      // Redirect to LINE Pay payment page
      if (result.payment_url) {
        window.location.href = result.payment_url;
      } else {
        router.push(`/events/${eventId}/payment`);
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : "建立訂單失敗");
    } finally {
      setSubmitting(false);
    }
  };

  if (!allocation) return null;

  const total = allocation.price_per_seat * allocation.seats.length;
  const minutes = Math.floor(countdown / 60);
  const seconds = countdown % 60;
  const isUrgent = countdown <= 120;

  const orderRows = [
    { label: "// event", value: "演唱會活動" },
    { label: "// seats", value: allocation.seats.map((s) => `${s.section_name} / ${s.row_label} / ${s.seat_number}號`).join(", ") },
    { label: "// quantity", value: `${allocation.seats.length} 張` },
  ];

  return (
    <div className="flex flex-col h-full">
      <Navbar />
      <main className="flex-1 flex flex-col items-center px-[200px] py-10 gap-8 overflow-auto">
        <div className="text-center">
          <h1 className="font-display text-[32px] font-bold">ORDER CONFIRMATION</h1>
          <p className="font-mono text-[13px] text-[var(--text-secondary)] mt-2">
            // please_review_and_complete_payment
          </p>
        </div>

        {/* Timer banner */}
        <div
          className={`w-full flex items-center justify-center gap-3 rounded-[var(--radius)] px-5 py-3.5 border ${
            isUrgent
              ? "bg-[#ef444422] border-[var(--status-red)]"
              : "bg-[var(--bg-card)] border-[var(--bg-elevated)]"
          }`}
        >
          <span className={`font-mono text-[13px] font-semibold ${isUrgent ? "text-[var(--status-red)]" : "text-[var(--accent-orange)]"}`}>
            座位保留倒數 {String(minutes).padStart(2, "0")}:{String(seconds).padStart(2, "0")} — 逾時座位將自動釋出
          </span>
        </div>

        {/* Order card */}
        <div className="w-full bg-[var(--bg-card)] rounded-[var(--radius)] p-6 flex flex-col gap-5">
          <span className="font-display text-base font-semibold text-[var(--accent-orange)]">
            [ORDER_DETAILS]
          </span>

          {orderRows.map((row, i) => (
            <div key={i}>
              <div className="flex items-center justify-between">
                <span className="font-mono text-xs text-[var(--text-secondary)]">{row.label}</span>
                <span className="font-mono text-xs font-semibold">{row.value}</span>
              </div>
              {i < orderRows.length - 1 && (
                <div className="h-px bg-[var(--bg-elevated)] mt-5" />
              )}
            </div>
          ))}

          <div className="h-px bg-[var(--bg-elevated)]" />

          <div className="flex items-center justify-between">
            <span className="font-mono text-sm font-semibold">// total_amount</span>
            <span className="font-display text-[28px] font-bold text-[var(--accent-orange)]">
              NT$ {total.toLocaleString()}
            </span>
          </div>
        </div>

        {error && (
          <p className="font-mono text-xs text-[var(--status-red)]">{error}</p>
        )}

        {/* LINE Pay button */}
        <button
          onClick={handleCheckout}
          disabled={submitting}
          className="w-full h-14 bg-[#00C300] rounded-[var(--radius)] font-display text-xl font-semibold text-white hover:brightness-110 disabled:opacity-50 transition"
        >
          {submitting ? "// processing..." : "LINE Pay 付款"}
        </button>

        <button
          onClick={() => router.push(`/events/${eventId}/select`)}
          className="font-mono text-xs text-[var(--text-secondary)] hover:text-[var(--text-primary)] transition"
        >
          // cancel_and_return_to_selection
        </button>
      </main>
    </div>
  );
}
