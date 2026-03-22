"use client";

import { useEffect, useState } from "react";
import Link from "next/link";

interface EventItem {
  id: string;
  title: string;
  event_date: string;
  venue_name: string;
  price_range: string;
  sale_status: string;
  sale_start: string;
  image_url: string;
}

export default function HomePage() {
  const [events, setEvents] = useState<EventItem[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    fetch("/api/events")
      .then((res) => res.json())
      .then((data) => setEvents(data.events || []))
      .catch(() => setEvents([]))
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
    <main className="max-w-6xl mx-auto px-4 py-8">
      <h1 className="text-3xl font-bold mb-8">演唱會售票</h1>
      {events.length === 0 ? (
        <p className="text-gray-500">目前沒有活動</p>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
          {events.map((event) => (
            <Link
              key={event.id}
              href={`/events/${event.id}`}
              className="bg-white rounded-lg shadow hover:shadow-lg transition-shadow overflow-hidden"
            >
              <div className="h-48 bg-gradient-to-br from-purple-500 to-pink-500 flex items-center justify-center">
                <span className="text-white text-xl font-bold">
                  {event.title}
                </span>
              </div>
              <div className="p-4">
                <h2 className="text-lg font-semibold mb-2">{event.title}</h2>
                <p className="text-sm text-gray-600 mb-1">
                  {new Date(event.event_date).toLocaleDateString("zh-TW")}
                </p>
                <p className="text-sm text-gray-600 mb-2">
                  {event.venue_name}
                </p>
                <div className="flex justify-between items-center">
                  <span className="text-sm font-medium text-purple-600">
                    NT$ {event.price_range}
                  </span>
                  <span
                    className={`text-xs px-2 py-1 rounded-full ${
                      event.sale_status === "熱賣中"
                        ? "bg-green-100 text-green-700"
                        : event.sale_status === "即將開賣"
                          ? "bg-yellow-100 text-yellow-700"
                          : "bg-gray-100 text-gray-500"
                    }`}
                  >
                    {event.sale_status}
                  </span>
                </div>
              </div>
            </Link>
          ))}
        </div>
      )}
    </main>
  );
}
