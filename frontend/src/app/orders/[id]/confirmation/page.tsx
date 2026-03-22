"use client";

import { useEffect, useState } from "react";
import { useParams } from "next/navigation";
import { api, type Order, type OrderItem } from "@/lib/api";

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
      <div className="flex items-center justify-center min-h-screen">
        <div className="text-lg text-gray-500">載入中...</div>
      </div>
    );
  }

  if (!order) {
    return (
      <div className="flex items-center justify-center min-h-screen">
        <div className="text-lg text-gray-500">找不到此訂單</div>
      </div>
    );
  }

  return (
    <main className="max-w-2xl mx-auto px-4 py-8">
      <div className="bg-white rounded-lg shadow-lg overflow-hidden">
        <div className="bg-green-500 px-6 py-8 text-center text-white">
          <div className="text-5xl mb-4">✓</div>
          <h1 className="text-2xl font-bold">購票成功！</h1>
          <p className="text-green-100 mt-2">訂單編號: {order.id.slice(0, 8).toUpperCase()}</p>
        </div>

        <div className="p-6">
          <h2 className="text-lg font-bold mb-4">座位資訊</h2>
          <div className="border rounded-lg divide-y mb-6">
            {items.map((item, i) => (
              <div key={i} className="flex justify-between px-4 py-3">
                <span>
                  {item.section_name} / {item.row_label} / {item.seat_number} 號
                </span>
                <span className="font-semibold">
                  NT$ {item.price.toLocaleString()}
                </span>
              </div>
            ))}
          </div>

          <div className="flex justify-between text-xl font-bold mb-6">
            <span>總計</span>
            <span className="text-purple-600">
              NT$ {order.total.toLocaleString()}
            </span>
          </div>

          {/* QR Code placeholder */}
          <div className="border-2 border-dashed rounded-lg p-8 text-center mb-6">
            <div className="bg-gray-100 w-48 h-48 mx-auto rounded-lg flex items-center justify-center mb-4">
              <span className="text-gray-400 text-sm">
                QR Code<br />
                {order.id.slice(0, 8).toUpperCase()}
              </span>
            </div>
            <p className="text-sm text-gray-500">
              入場時請出示此 QR Code
            </p>
          </div>

          <a
            href="/orders"
            className="block w-full text-center bg-purple-600 text-white py-3 rounded-lg font-bold hover:bg-purple-700 transition"
          >
            查看我的訂單
          </a>
        </div>
      </div>
    </main>
  );
}
