"use client";

import { useEffect, useState } from "react";
import { useParams, useRouter } from "next/navigation";
import { api, type EventDetail } from "@/lib/api";

export default function EventDetailPage() {
  const params = useParams();
  const router = useRouter();
  const [event, setEvent] = useState<EventDetail | null>(null);
  const [loading, setLoading] = useState(true);
  const [countdown, setCountdown] = useState("");

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
      const now = Date.now();
      const diff = saleStart - now;

      if (diff <= 0) {
        setCountdown("");
        clearInterval(timer);
        return;
      }

      const days = Math.floor(diff / (1000 * 60 * 60 * 24));
      const hours = Math.floor((diff % (1000 * 60 * 60 * 24)) / (1000 * 60 * 60));
      const minutes = Math.floor((diff % (1000 * 60 * 60)) / (1000 * 60));
      const seconds = Math.floor((diff % (1000 * 60)) / 1000);

      setCountdown(`${days}天 ${hours}時 ${minutes}分 ${seconds}秒`);
    }, 1000);

    return () => clearInterval(timer);
  }, [event]);

  if (loading) {
    return (
      <div className="flex items-center justify-center min-h-screen">
        <div className="text-lg text-gray-500">載入中...</div>
      </div>
    );
  }

  if (!event) {
    return (
      <div className="flex items-center justify-center min-h-screen">
        <div className="text-lg text-gray-500">找不到此活動</div>
      </div>
    );
  }

  const isSaleStarted = new Date(event.sale_start).getTime() <= Date.now();

  return (
    <main className="max-w-4xl mx-auto px-4 py-8">
      <div className="bg-white rounded-lg shadow-lg overflow-hidden">
        <div className="h-64 bg-gradient-to-br from-purple-600 to-pink-500 flex items-center justify-center">
          <h1 className="text-4xl font-bold text-white">{event.title}</h1>
        </div>

        <div className="p-6">
          <div className="grid grid-cols-2 gap-4 mb-6">
            <div>
              <p className="text-sm text-gray-500">演出日期</p>
              <p className="text-lg font-semibold">
                {new Date(event.event_date).toLocaleDateString("zh-TW", {
                  year: "numeric",
                  month: "long",
                  day: "numeric",
                  weekday: "long",
                })}
              </p>
            </div>
            <div>
              <p className="text-sm text-gray-500">演出場館</p>
              <p className="text-lg font-semibold">{event.venue_name}</p>
            </div>
          </div>

          {countdown && (
            <div className="bg-yellow-50 border border-yellow-200 rounded-lg p-4 mb-6 text-center">
              <p className="text-sm text-yellow-600 mb-1">距離開賣</p>
              <p className="text-2xl font-bold text-yellow-700">{countdown}</p>
            </div>
          )}

          <h2 className="text-xl font-bold mb-4">票價資訊</h2>
          <div className="border rounded-lg overflow-hidden mb-6">
            <table className="w-full">
              <thead className="bg-gray-50">
                <tr>
                  <th className="px-4 py-3 text-left text-sm font-medium text-gray-600">
                    區域
                  </th>
                  <th className="px-4 py-3 text-right text-sm font-medium text-gray-600">
                    票價
                  </th>
                  <th className="px-4 py-3 text-right text-sm font-medium text-gray-600">
                    剩餘
                  </th>
                </tr>
              </thead>
              <tbody>
                {event.sections.map((section) => (
                  <tr key={section.id} className="border-t">
                    <td className="px-4 py-3 font-medium">
                      {section.section_name}
                    </td>
                    <td className="px-4 py-3 text-right text-purple-600 font-semibold">
                      NT$ {section.price.toLocaleString()}
                    </td>
                    <td className="px-4 py-3 text-right">
                      <span
                        className={`px-2 py-1 rounded text-sm ${
                          section.remaining === 0
                            ? "bg-gray-100 text-gray-500"
                            : section.remaining < section.quota * 0.1
                              ? "bg-red-100 text-red-600"
                              : "bg-green-100 text-green-600"
                        }`}
                      >
                        {section.remaining === 0
                          ? "售完"
                          : `${section.remaining} 張`}
                      </span>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>

          <button
            onClick={() => router.push(`/events/${eventId}/queue`)}
            disabled={!isSaleStarted}
            className={`w-full py-4 rounded-lg text-lg font-bold transition ${
              isSaleStarted
                ? "bg-purple-600 text-white hover:bg-purple-700"
                : "bg-gray-300 text-gray-500 cursor-not-allowed"
            }`}
          >
            {isSaleStarted ? "立即購票" : "尚未開賣"}
          </button>
        </div>
      </div>
    </main>
  );
}
