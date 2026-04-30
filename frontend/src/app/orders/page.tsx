"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { api, type Order } from "@/lib/api";
import Navbar from "@/components/Navbar";

const statusMap: Record<string, { label: string; bg: string }> = {
  pending: { label: "PENDING", bg: "bg-[var(--status-yellow)]" },
  confirmed: { label: "CONFIRMED", bg: "bg-[var(--accent-teal)]" },
  cancelled: { label: "CANCELLED", bg: "bg-[var(--status-grey)]" },
  payment_pending: { label: "PROCESSING", bg: "bg-[var(--accent-orange)]" },
};

export default function OrdersPage() {
  const [orders, setOrders] = useState<Order[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    api
      .listOrders()
      .then(({ orders }) => setOrders(orders || []))
      .catch(() => setOrders([]))
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
      <main className="flex-1 px-12 py-10 overflow-auto">
        <div className="flex items-end justify-between mb-6">
          <h1 className="font-display text-4xl font-bold">MY ORDERS</h1>
          <span className="font-mono text-[13px] text-[var(--text-secondary)]">
            // {orders.length} orders found
          </span>
        </div>

        {orders.length === 0 ? (
          <div className="bg-[var(--bg-card)] rounded-[var(--radius)] p-12 text-center">
            <p className="font-mono text-[var(--text-secondary)]">// 尚無訂單紀錄</p>
            <Link
              href="/"
              className="inline-block mt-4 font-mono text-sm text-[var(--accent-orange)] hover:underline"
            >
              // browse_events
            </Link>
          </div>
        ) : (
          <div className="flex flex-col gap-4">
            {orders.map((order) => {
              const status = statusMap[order.status] || {
                label: order.status.toUpperCase(),
                bg: "bg-[var(--bg-placeholder)]",
              };
              return (
                <Link
                  key={order.id}
                  href={`/orders/${order.id}/confirmation`}
                  className="flex items-center justify-between bg-[var(--bg-card)] rounded-[var(--radius)] p-5 hover:ring-1 hover:ring-[var(--accent-orange)] transition-all"
                >
                  <div className="flex flex-col gap-2">
                    <span className="font-display text-base font-semibold">
                      {order.event_title || `訂單 #${order.id.slice(0, 8).toUpperCase()}`}
                    </span>
                    <div className="flex gap-4">
                      <span className="font-mono text-[11px] text-[var(--text-secondary)]">
                        {order.event_date
                          ? new Date(order.event_date).toLocaleDateString("zh-TW")
                          : new Date(order.created_at).toLocaleDateString("zh-TW")}
                      </span>
                      <span className="font-mono text-[11px] text-[var(--text-secondary)]">
                        {order.venue_name || "venue"}
                      </span>
                      <span className="font-mono text-[11px] text-[var(--text-secondary)]">
                        {order.ticket_count || 0} 張
                      </span>
                    </div>
                  </div>

                  <div className="flex flex-col items-end gap-2">
                    <span className="font-display text-lg font-bold text-[var(--accent-orange)]">
                      NT$ {order.total.toLocaleString()}
                    </span>
                    <span
                      className={`${status.bg} text-[var(--text-on-accent)] font-mono text-[10px] font-semibold px-3 py-1 rounded-lg`}
                    >
                      {status.label}
                    </span>
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
