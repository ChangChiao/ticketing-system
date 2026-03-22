"use client";

import { useEffect } from "react";

export default function PaymentPage() {
  useEffect(() => {
    // In real flow, user would be redirected to LINE Pay
    // After LINE Pay confirms, they return to /api/payments/confirm
    // which redirects to /orders/[id]/confirmation
  }, []);

  return (
    <main className="min-h-screen flex items-center justify-center">
      <div className="bg-white rounded-lg shadow-lg p-10 text-center max-w-md">
        <div className="animate-spin rounded-full h-16 w-16 border-4 border-green-200 border-t-green-500 mx-auto mb-6" />
        <h2 className="text-xl font-bold mb-2">正在跳轉至 LINE Pay</h2>
        <p className="text-gray-500">請在 LINE Pay 頁面完成付款</p>
        <p className="text-sm text-gray-400 mt-4">
          完成後將自動返回此頁面
        </p>
      </div>
    </main>
  );
}
