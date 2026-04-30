"use client";

import { useEffect, useState } from "react";
import { useParams, useRouter } from "next/navigation";
import { useAuthStore } from "@/stores/auth";
import { getWebSocketBaseURL } from "@/lib/ws";
import Navbar from "@/components/Navbar";

export default function PaymentPage() {
  const params = useParams();
  const router = useRouter();
  const eventId = params.id as string;
  const token = useAuthStore((s) => s.token);
  const [status, setStatus] = useState<"redirecting" | "timeout" | "error">("redirecting");
  const [countdown, setCountdown] = useState(600); // 10 minutes
  const [warning, setWarning] = useState("");

  useEffect(() => {
    const pendingOrderId = sessionStorage.getItem("pending_order_id");
    const paymentUrl = sessionStorage.getItem("pending_payment_url");
    if (!pendingOrderId) {
      // No pending order — user landed here directly
      router.push(`/events/${eventId}/select`);
      return;
    }

    if (paymentUrl) {
      const redirect = setTimeout(() => {
        window.location.href = paymentUrl;
      }, 1200);
      return () => clearTimeout(redirect);
    }
  }, [eventId, router]);

  useEffect(() => {
    const pendingOrderId = sessionStorage.getItem("pending_order_id");
    if (!pendingOrderId || !token) return;

    const wsBase = getWebSocketBaseURL();
    const params = new URLSearchParams({ event_id: eventId, token });
    const ws = new WebSocket(`${wsBase}/ws?${params.toString()}`);

    ws.onmessage = (event) => {
      try {
        const msg = JSON.parse(event.data);
        if (
          msg.type === "payment_warning" &&
          msg.data?.order_id === pendingOrderId
        ) {
          setWarning(msg.data.message || "付款期限即將到期");
          setCountdown((current) => Math.min(current, 120));
        }
      } catch {
        // ignore parse errors
      }
    };

    return () => ws.close();
  }, [eventId, token]);

  useEffect(() => {
    const pendingOrderId = sessionStorage.getItem("pending_order_id");
    if (!pendingOrderId) return;

    // Start countdown for the 10-minute payment window
    const timer = setInterval(() => {
      setCountdown((prev) => {
        if (prev <= 1) {
          clearInterval(timer);
          setStatus("timeout");
          sessionStorage.removeItem("pending_order_id");
          sessionStorage.removeItem("pending_payment_url");
          return 0;
        }
        return prev - 1;
      });
    }, 1000);

    return () => clearInterval(timer);
  }, []);

  const minutes = Math.floor(countdown / 60);
  const seconds = countdown % 60;
  const isUrgent = countdown <= 120;

  if (status === "timeout") {
    return (
      <div className="flex flex-col h-full">
        <Navbar />
        <main className="flex-1 flex flex-col items-center justify-center gap-6 px-[200px]">
          <div className="w-24 h-24 rounded-full border-2 border-[var(--status-red)] bg-[#ef444422] flex items-center justify-center">
            <span className="text-4xl text-[var(--status-red)]">✕</span>
          </div>
          <h1 className="font-display text-3xl font-bold text-[var(--status-red)]">
            付款逾時，座位已釋出
          </h1>
          <p className="font-mono text-[13px] text-[var(--text-secondary)]">
            // payment_timeout — seats_released
          </p>
          <button
            onClick={() => router.push(`/events/${eventId}`)}
            className="mt-4 px-8 py-3 bg-[var(--bg-card)] border border-[var(--accent-orange)] rounded-[var(--radius)] font-mono text-sm font-semibold text-[var(--accent-orange)] hover:bg-[var(--bg-elevated)] transition"
          >
            返回活動頁面
          </button>
        </main>
      </div>
    );
  }

  return (
    <div className="flex flex-col h-full">
      <Navbar />
      <main className="flex-1 flex flex-col items-center justify-center gap-8 px-[200px]">
        <div className="animate-spin rounded-full h-16 w-16 border-4 border-[var(--accent-teal)] border-t-transparent" />

        <div className="text-center">
          <h1 className="font-display text-2xl font-bold">正在跳轉至 LINE Pay</h1>
          <p className="font-mono text-[13px] text-[var(--text-secondary)] mt-2">
            // redirecting_to_line_pay
          </p>
        </div>

        {warning && (
          <div className="rounded-[var(--radius)] border border-[var(--status-red)] bg-[#ef444422] px-5 py-3">
            <span className="font-mono text-sm font-semibold text-[var(--status-red)]">
              {warning}
            </span>
          </div>
        )}

        {/* Countdown */}
        <div
          className={`flex items-center justify-center gap-3 rounded-[var(--radius)] px-6 py-3 border ${
            isUrgent
              ? "bg-[#ef444422] border-[var(--status-red)]"
              : "bg-[var(--bg-card)] border-[var(--bg-elevated)]"
          }`}
        >
          <span
            className={`font-mono text-sm font-semibold ${
              isUrgent ? "text-[var(--status-red)]" : "text-[var(--accent-orange)]"
            }`}
          >
            付款期限 {String(minutes).padStart(2, "0")}:{String(seconds).padStart(2, "0")}
          </span>
        </div>

        <div className="text-center space-y-2">
          <p className="font-mono text-xs text-[var(--text-secondary)]">
            請在 LINE Pay 頁面完成付款，完成後將自動返回
          </p>
          <p className="font-mono text-xs text-[var(--text-secondary)]">
            請勿關閉此頁面
          </p>
        </div>

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
