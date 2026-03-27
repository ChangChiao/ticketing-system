"use client";

import { useEffect, useState } from "react";
import { useParams } from "next/navigation";
import Link from "next/link";
import { QRCodeSVG } from "qrcode.react";
import { api, type Order, type OrderItem } from "@/lib/api";
import Navbar from "@/components/Navbar";

export default function ConfirmationPage() {
  const params = useParams();
  const orderId = params.id as string;

  const [order, setOrder] = useState<Order | null>(null);
  const [items, setItems] = useState<OrderItem[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    api
      .getOrder(orderId)
      .then(({ order, items }) => {
        setOrder(order);
        setItems(items);
      })
      .catch(() => {})
      .finally(() => setLoading(false));
  }, [orderId]);

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

  if (!order) {
    return (
      <div className="flex flex-col h-full">
        <Navbar />
        <div className="flex-1 flex items-center justify-center">
          <span className="font-mono text-[var(--text-secondary)]">// order_not_found</span>
        </div>
      </div>
    );
  }

  const isConfirmed = order.status === "confirmed";

  return (
    <div className="flex flex-col h-full">
      <Navbar />
      <main className="flex-1 flex flex-col items-center px-[200px] py-10 gap-8 overflow-auto">
        {/* Success icon */}
        <div
          className={`w-24 h-24 rounded-full border-2 flex items-center justify-center ${
            isConfirmed
              ? "border-[var(--accent-teal)] bg-[#00D4AA22]"
              : "border-[var(--status-yellow)] bg-[#eab30822]"
          }`}
        >
          <span className={`text-4xl ${isConfirmed ? "text-[var(--accent-teal)]" : "text-[var(--status-yellow)]"}`}>
            {isConfirmed ? "✓" : "⏳"}
          </span>
        </div>

        <div className="text-center">
          <h1 className={`font-display text-4xl font-bold ${isConfirmed ? "text-[var(--accent-teal)]" : "text-[var(--status-yellow)]"}`}>
            {isConfirmed ? "PAYMENT SUCCESSFUL" : "PAYMENT PENDING"}
          </h1>
          <p className="font-mono text-[13px] text-[var(--text-secondary)] mt-2">
            // order_id: {order.id.slice(0, 8).toUpperCase()}
          </p>
        </div>

        {/* Ticket card */}
        <div className="w-full flex bg-[var(--bg-card)] rounded-[var(--radius)] overflow-hidden">
          {/* Left - details */}
          <div className="flex-1 p-6 flex flex-col gap-4">
            <span className="font-display text-lg font-semibold">
              訂單 #{order.id.slice(0, 8).toUpperCase()}
            </span>
            <div className="flex flex-col gap-2.5">
              {items.map((item, i) => (
                <div key={i} className="flex items-center justify-between">
                  <span className="font-mono text-[11px] text-[var(--text-secondary)]">// seat {i + 1}</span>
                  <span className="font-mono text-xs font-semibold">
                    {item.section_name} / {item.row_label} / {item.seat_number}號
                  </span>
                </div>
              ))}
              <div className="h-px bg-[var(--bg-elevated)] my-1" />
              <div className="flex items-center justify-between">
                <span className="font-mono text-[11px] text-[var(--text-secondary)]">// total</span>
                <span className="font-display text-xl font-bold text-[var(--accent-orange)]">
                  NT$ {order.total.toLocaleString()}
                </span>
              </div>
            </div>
          </div>

          {/* Right - QR code */}
          {isConfirmed && (
            <div className="w-[180px] bg-[var(--bg-elevated)] flex flex-col items-center justify-center gap-3 p-5">
              <div className="bg-white rounded-lg p-2.5">
                <QRCodeSVG
                  value={JSON.stringify({
                    order_id: order.id,
                    seats: items.map((item) => ({
                      section: item.section_name,
                      row: item.row_label,
                      seat: item.seat_number,
                    })),
                  })}
                  size={100}
                  level="M"
                />
              </div>
              <span className="font-mono text-[9px] text-[var(--text-secondary)]">// scan_to_enter</span>
            </div>
          )}
        </div>

        {/* Actions */}
        <Link
          href="/orders"
          className="w-full h-12 flex items-center justify-center gap-2 bg-[var(--bg-card)] border border-[var(--accent-teal)] rounded-[var(--radius)] font-mono text-[13px] font-semibold text-[var(--accent-teal)] hover:bg-[var(--bg-elevated)] transition"
        >
          查看我的訂單
        </Link>

        <Link
          href="/"
          className="font-mono text-xs text-[var(--text-secondary)] hover:text-[var(--text-primary)] transition"
        >
          // return_to_home
        </Link>
      </main>
    </div>
  );
}
