"use client";

import { useEffect, useState } from "react";
import { useParams, useRouter } from "next/navigation";
import { useAuthStore } from "@/stores/auth";
import { api, type SeatInfo } from "@/lib/api";

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

  // Countdown timer
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
      const order = await api.createOrder({
        event_id: allocation.event_id,
        seats: allocation.seats,
        price_per_seat: allocation.price_per_seat,
      });

      // Store order ID for payment flow
      sessionStorage.setItem("pending_order_id", order.id);
      sessionStorage.removeItem("allocation");

      // In real implementation, redirect to LINE Pay URL returned by backend
      // For now, redirect to payment page
      router.push(`/events/${eventId}/payment`);
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

  return (
    <main className="max-w-2xl mx-auto px-4 py-8">
      <div className="bg-white rounded-lg shadow-lg overflow-hidden">
        {/* Countdown bar */}
        <div
          className={`px-6 py-3 text-center font-bold text-white ${
            isUrgent ? "bg-red-500 animate-pulse" : "bg-purple-600"
          }`}
        >
          付款剩餘時間: {minutes}:{seconds.toString().padStart(2, "0")}
        </div>

        <div className="p-6">
          <h1 className="text-2xl font-bold mb-6">訂單確認</h1>

          <div className="border rounded-lg divide-y mb-6">
            {allocation.seats.map((seat, i) => (
              <div key={i} className="flex justify-between px-4 py-3">
                <span>
                  {seat.section_name} / {seat.row_label} / {seat.seat_number} 號
                </span>
                <span className="font-semibold">
                  NT$ {allocation.price_per_seat.toLocaleString()}
                </span>
              </div>
            ))}
          </div>

          <div className="flex justify-between text-xl font-bold mb-6">
            <span>總計</span>
            <span className="text-purple-600">
              NT$ {total.toLocaleString()}
            </span>
          </div>

          <div className="bg-gray-50 rounded-lg p-4 mb-6">
            <p className="text-sm text-gray-500">付款方式</p>
            <div className="flex items-center gap-2 mt-2">
              <div className="bg-green-500 text-white px-3 py-1 rounded text-sm font-bold">
                LINE Pay
              </div>
              <span className="text-sm text-gray-600">
                點擊下方按鈕後將跳轉至 LINE Pay 付款
              </span>
            </div>
          </div>

          {error && <p className="text-red-500 text-sm mb-4">{error}</p>}

          <button
            onClick={handleCheckout}
            disabled={submitting}
            className="w-full bg-green-500 text-white py-4 rounded-lg text-lg font-bold hover:bg-green-600 disabled:bg-gray-400 transition"
          >
            {submitting ? "處理中..." : `以 LINE Pay 付款 NT$ ${total.toLocaleString()}`}
          </button>
        </div>
      </div>
    </main>
  );
}
