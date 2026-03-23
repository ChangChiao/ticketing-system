"use client";

import { useEffect, useState } from "react";
import { useParams, useRouter } from "next/navigation";
import { api, type EventDetail, type AllocatedSeats } from "@/lib/api";
import VenueMap from "@/components/VenueMap";
import Navbar from "@/components/Navbar";

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
  const [timer, setTimer] = useState(600); // 10 min countdown

  useEffect(() => {
    api
      .getEvent(eventId)
      .then(setEvent)
      .catch(() => {})
      .finally(() => setLoading(false));
  }, [eventId]);

  useEffect(() => {
    const interval = setInterval(() => {
      setTimer((t) => (t > 0 ? t - 1 : 0));
    }, 1000);
    return () => clearInterval(interval);
  }, []);

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
      <div className="flex flex-col h-full">
        <Navbar />
        <div className="flex-1 flex items-center justify-center">
          <span className="font-mono text-[var(--text-secondary)]">// loading_venue_map...</span>
        </div>
      </div>
    );
  }

  const mins = Math.floor(timer / 60);
  const secs = timer % 60;

  return (
    <div className="flex flex-col h-full">
      <Navbar />
      <main className="flex-1 flex gap-8 px-12 py-8 overflow-hidden">
        {/* Map column */}
        <div className="flex-1 flex flex-col gap-5 min-h-0">
          <div className="flex items-center justify-between">
            <h1 className="font-display text-2xl font-bold">VENUE MAP</h1>
            <div className="flex items-center gap-2 bg-[var(--bg-card)] rounded-xl px-3.5 py-1.5">
              <span className="font-mono text-[13px] font-semibold text-[var(--accent-orange)]">
                {String(mins).padStart(2, "0")}:{String(secs).padStart(2, "0")}
              </span>
            </div>
          </div>

          <div className="flex-1 min-h-0">
            <VenueMap
              sections={event.sections}
              stageConfig={
                (event.layout_data as { stage?: { x: number; y: number; width: number; height: number } })?.stage
              }
              onSelect={setSelectedSection}
              selectedSectionId={selectedSection}
            />
          </div>
        </div>

        {/* Side panel */}
        <div className="w-[340px] shrink-0 flex flex-col gap-5 overflow-auto">
          {/* Legend */}
          <div className="bg-[var(--bg-card)] rounded-[var(--radius)] p-5 flex flex-col gap-3">
            <span className="font-display text-sm font-semibold text-[var(--accent-orange)]">[LEGEND]</span>
            {[
              { color: "bg-[var(--status-green)]", label: "> 50% available" },
              { color: "bg-[var(--status-yellow)]", label: "10-50% available" },
              { color: "bg-[var(--status-red)]", label: "< 10% available" },
              { color: "bg-[var(--status-grey)]", label: "sold_out" },
            ].map((item) => (
              <div key={item.label} className="flex items-center gap-2">
                <div className={`w-2.5 h-2.5 rounded-sm ${item.color}`} />
                <span className="font-mono text-[11px] text-[var(--text-secondary)]">{item.label}</span>
              </div>
            ))}
          </div>

          {/* Selection */}
          <div className="bg-[var(--bg-card)] rounded-[var(--radius)] p-5 flex flex-col gap-4">
            <span className="font-display text-sm font-semibold text-[var(--accent-orange)]">[SELECTED_SECTION]</span>

            {selectedSectionData ? (
              <>
                <div className="flex flex-col gap-1.5">
                  <span className="font-mono text-[11px] text-[var(--text-secondary)]">// section</span>
                  <div className="bg-[var(--bg-elevated)] rounded-xl px-3.5 h-10 flex items-center justify-between">
                    <span className="font-mono text-xs font-semibold">
                      {selectedSectionData.section_name} — NT$ {selectedSectionData.price.toLocaleString()}
                    </span>
                  </div>
                </div>

                <div className="flex flex-col gap-1.5">
                  <span className="font-mono text-[11px] text-[var(--text-secondary)]">// quantity (max 4)</span>
                  <div className="flex gap-2">
                    {[1, 2, 3, 4].map((n) => (
                      <button
                        key={n}
                        onClick={() => setQuantity(n)}
                        disabled={n > selectedSectionData.remaining}
                        className={`flex-1 h-10 rounded-xl font-mono text-sm font-semibold transition ${
                          quantity === n
                            ? "bg-[var(--accent-orange)] text-[var(--text-on-accent)]"
                            : n > selectedSectionData.remaining
                              ? "bg-[var(--bg-placeholder)] text-[var(--text-secondary)] cursor-not-allowed opacity-50"
                              : "bg-[var(--bg-elevated)] text-[var(--text-secondary)] hover:text-[var(--text-primary)]"
                        }`}
                      >
                        {n}
                      </button>
                    ))}
                  </div>
                </div>

                <div className="flex items-center justify-between bg-[var(--bg-elevated)] rounded-xl px-4 py-2.5">
                  <span className="font-mono text-xs text-[var(--text-secondary)]">// total</span>
                  <span className="font-display text-xl font-bold text-[var(--accent-orange)]">
                    NT$ {(selectedSectionData.price * quantity).toLocaleString()}
                  </span>
                </div>
              </>
            ) : (
              <p className="font-mono text-xs text-[var(--text-secondary)] text-center py-8">
                // click_a_section_on_the_map
              </p>
            )}
          </div>

          {error && (
            <p className="font-mono text-xs text-[var(--status-red)] text-center">{error}</p>
          )}

          <button
            onClick={handleAllocate}
            disabled={!selectedSection || allocating}
            className={`w-full h-12 rounded-[var(--radius)] font-display text-base font-semibold tracking-wide transition flex items-center justify-center gap-2 ${
              selectedSection && !allocating
                ? "bg-[var(--accent-orange)] text-[var(--text-on-accent)] hover:brightness-110"
                : "bg-[var(--bg-placeholder)] text-[var(--text-secondary)] cursor-not-allowed"
            }`}
          >
            {allocating ? "// allocating..." : "確認選位"}
          </button>
        </div>
      </main>
    </div>
  );
}
