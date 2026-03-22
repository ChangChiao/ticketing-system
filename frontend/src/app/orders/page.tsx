"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { api, type Order } from "@/lib/api";

const statusMap: Record<string, { label: string; color: string }> = {
  pending: { label: "待付款", color: "bg-yellow-100 text-yellow-700" },
  confirmed: { label: "已確認", color: "bg-green-100 text-green-700" },
  cancelled: { label: "已取消", color: "bg-gray-100 text-gray-500" },
  payment_pending: { label: "付款處理中", color: "bg-blue-100 text-blue-700" },
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
      <div className="flex items-center justify-center min-h-screen">
        <div className="text-lg text-gray-500">載入中...</div>
      </div>
    );
  }

  return (
    <main className="max-w-4xl mx-auto px-4 py-8">
      <h1 className="text-2xl font-bold mb-6">我的訂單</h1>

      {orders.length === 0 ? (
        <div className="bg-white rounded-lg shadow p-8 text-center">
          <p className="text-gray-400 text-lg">尚無訂單紀錄</p>
          <Link
            href="/"
            className="inline-block mt-4 text-purple-600 hover:underline"
          >
            瀏覽活動
          </Link>
        </div>
      ) : (
        <div className="space-y-4">
          {orders.map((order) => {
            const status = statusMap[order.status] || {
              label: order.status,
              color: "bg-gray-100 text-gray-500",
            };
            return (
              <Link
                key={order.id}
                href={`/orders/${order.id}/confirmation`}
                className="block bg-white rounded-lg shadow hover:shadow-lg transition-shadow p-6"
              >
                <div className="flex justify-between items-start">
                  <div>
                    <p className="text-sm text-gray-400">
                      訂單 #{order.id.slice(0, 8).toUpperCase()}
                    </p>
                    <p className="font-semibold mt-1">
                      NT$ {order.total.toLocaleString()}
                    </p>
                    <p className="text-sm text-gray-500 mt-1">
                      {new Date(order.created_at).toLocaleDateString("zh-TW")}
                    </p>
                  </div>
                  <span
                    className={`px-3 py-1 rounded-full text-sm font-medium ${status.color}`}
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
  );
}
