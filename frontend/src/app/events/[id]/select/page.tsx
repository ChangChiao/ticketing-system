"use client";

import { useEffect, useState } from "react";
import { useParams, useRouter } from "next/navigation";
import { api, type EventDetail, type AllocatedSeats } from "@/lib/api";
import VenueMap from "@/components/VenueMap";

export default function SelectPage() {
  const params = useParams();
  const router = useRouter();
  const eventId = params.id as string;

  const [event, setEvent] = useState<EventDetail | null>(null);
  const [selectedSection, setSelectedSection] = useState<string | null>(null);
  const [quantity, setQuantity] = useState(1);
  const [loading, setLoading] = useState(true);
  const [allocating, setAllocating] = useState(false);
  const [error, setError] = useState("");

  useEffect(() => {
    api
      .getEvent(eventId)
      .then(setEvent)
      .catch(() => {})
      .finally(() => setLoading(false));
  }, [eventId]);

  const selectedSectionData = event?.sections.find(
    (s) => s.section_id === selectedSection
  );

  const handleAllocate = async () => {
    if (!selectedSection) return;
    setAllocating(true);
    setError("");

    try {
      const result: AllocatedSeats = await api.allocateSeats(eventId, {
        section_id: selectedSection,
        quantity,
      });

      // Store allocation in sessionStorage for checkout
      sessionStorage.setItem(
        "allocation",
        JSON.stringify({
          ...result,
          event_id: eventId,
          price_per_seat: selectedSectionData?.price || 0,
        })
      );

      router.push(`/events/${eventId}/checkout`);
    } catch (err) {
      setError(err instanceof Error ? err.message : "選位失敗");
    } finally {
      setAllocating(false);
    }
  };

  if (loading || !event) {
    return (
      <div className="flex items-center justify-center min-h-screen">
        <div className="text-lg text-gray-500">載入場館圖...</div>
      </div>
    );
  }

  return (
    <main className="max-w-6xl mx-auto px-4 py-8">
      <h1 className="text-2xl font-bold mb-2">{event.title}</h1>
      <p className="text-gray-500 mb-6">請選擇區域和張數</p>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        <div className="lg:col-span-2">
          <VenueMap
            sections={event.sections}
            stageConfig={
              (event.layout_data as { stage?: { x: number; y: number; width: number; height: number } })?.stage
            }
            onSelect={setSelectedSection}
            selectedSectionId={selectedSection}
          />
        </div>

        <div className="bg-white rounded-lg shadow p-6">
          {selectedSectionData ? (
            <>
              <h2 className="text-lg font-bold mb-4">
                {selectedSectionData.section_name}
              </h2>
              <p className="text-sm text-gray-500 mb-2">
                票價: NT$ {selectedSectionData.price.toLocaleString()}
              </p>
              <p className="text-sm text-gray-500 mb-4">
                剩餘: {selectedSectionData.remaining} 張
              </p>

              <div className="mb-4">
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  張數
                </label>
                <div className="flex gap-2">
                  {[1, 2, 3, 4].map((n) => (
                    <button
                      key={n}
                      onClick={() => setQuantity(n)}
                      disabled={n > selectedSectionData.remaining}
                      className={`w-12 h-12 rounded-lg text-lg font-bold transition ${
                        quantity === n
                          ? "bg-purple-600 text-white"
                          : n > selectedSectionData.remaining
                            ? "bg-gray-100 text-gray-300 cursor-not-allowed"
                            : "bg-gray-100 text-gray-700 hover:bg-gray-200"
                      }`}
                    >
                      {n}
                    </button>
                  ))}
                </div>
              </div>

              <div className="border-t pt-4 mb-4">
                <div className="flex justify-between text-lg font-bold">
                  <span>小計</span>
                  <span className="text-purple-600">
                    NT${" "}
                    {(selectedSectionData.price * quantity).toLocaleString()}
                  </span>
                </div>
              </div>

              {error && <p className="text-red-500 text-sm mb-4">{error}</p>}

              <button
                onClick={handleAllocate}
                disabled={allocating}
                className="w-full bg-purple-600 text-white py-3 rounded-lg font-bold hover:bg-purple-700 disabled:bg-gray-400 transition"
              >
                {allocating ? "座位分配中..." : "確認選位"}
              </button>
            </>
          ) : (
            <div className="text-center text-gray-400 py-12">
              <p className="text-lg">請在場館圖上選擇區域</p>
            </div>
          )}
        </div>
      </div>
    </main>
  );
}
