"use client";

import { useEffect, useState, useRef } from "react";
import { useParams, useRouter } from "next/navigation";
import { useAuthStore } from "@/stores/auth";

export default function QueuePage() {
  const params = useParams();
  const router = useRouter();
  const eventId = params.id as string;
  const user = useAuthStore((s) => s.user);
  const token = useAuthStore((s) => s.token);

  const [position, setPosition] = useState<number | null>(null);
  const [estimatedWait, setEstimatedWait] = useState("");
  const [status, setStatus] = useState<"joining" | "waiting" | "your_turn">("joining");
  const [error, setError] = useState("");
  const wsRef = useRef<WebSocket | null>(null);
  const turnTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  // Join queue on mount
  useEffect(() => {
    if (!token) {
      router.push("/auth");
      return;
    }

    const joinQueue = async () => {
      try {
        const res = await fetch(`/api/events/${eventId}/queue/join`, {
          method: "POST",
          headers: {
            Authorization: `Bearer ${token}`,
            "Content-Type": "application/json",
          },
        });
        const data = await res.json();
        if (!res.ok) {
          setError(data.error || "加入排隊失敗");
          return;
        }
        setPosition(data.position);
        setEstimatedWait(data.estimated_wait);
        setStatus("waiting");
      } catch {
        setError("網路錯誤，請重試");
      }
    };

    joinQueue();
  }, [eventId, token, router]);

  // WebSocket connection
  useEffect(() => {
    if (status !== "waiting" || !user) return;

    const wsBase = process.env.NEXT_PUBLIC_WS_URL || "ws://localhost:8080";
    const wsUrl = `${wsBase}/ws?user_id=${user.id}&event_id=${eventId}`;
    const ws = new WebSocket(wsUrl);
    wsRef.current = ws;

    ws.onmessage = (event) => {
      try {
        const msg = JSON.parse(event.data);
        if (msg.type === "queue_update") {
          setPosition(msg.data.position);
          setEstimatedWait(msg.data.estimated_wait);
          if (msg.data.status === "your_turn") {
            setStatus("your_turn");
            // 60 second window to enter selection page
            turnTimerRef.current = setTimeout(() => {
              setError("您未在時間內進入選位頁面，已重新排隊");
              setStatus("waiting");
            }, 60000);
          }
        }
      } catch {}
    };

    ws.onclose = () => {
      // Attempt reconnection within 30 seconds
      setTimeout(() => {
        if (wsRef.current === ws) {
          const reconnect = new WebSocket(wsUrl);
          wsRef.current = reconnect;
        }
      }, 3000);
    };

    return () => {
      ws.close();
      if (turnTimerRef.current) clearTimeout(turnTimerRef.current);
    };
  }, [status, user, eventId]);

  const handleEnterSelection = () => {
    if (turnTimerRef.current) clearTimeout(turnTimerRef.current);
    wsRef.current?.close();
    router.push(`/events/${eventId}/select`);
  };

  if (error) {
    return (
      <main className="min-h-screen flex items-center justify-center">
        <div className="bg-white rounded-lg shadow-lg p-8 text-center max-w-md">
          <p className="text-red-500 text-lg mb-4">{error}</p>
          <button
            onClick={() => window.location.reload()}
            className="bg-purple-600 text-white px-6 py-2 rounded-lg hover:bg-purple-700"
          >
            重試
          </button>
        </div>
      </main>
    );
  }

  return (
    <main className="min-h-screen flex items-center justify-center bg-gradient-to-br from-purple-900 to-indigo-900">
      <div className="bg-white rounded-2xl shadow-2xl p-10 text-center max-w-md w-full mx-4">
        {status === "joining" && (
          <>
            <div className="animate-spin rounded-full h-16 w-16 border-4 border-purple-200 border-t-purple-600 mx-auto mb-6" />
            <p className="text-lg text-gray-600">正在加入排隊...</p>
          </>
        )}

        {status === "waiting" && (
          <>
            <div className="mb-6">
              <div className="relative w-24 h-24 mx-auto">
                <div className="absolute inset-0 rounded-full border-4 border-purple-100" />
                <div className="absolute inset-0 rounded-full border-4 border-purple-500 border-t-transparent animate-spin" />
                <div className="absolute inset-0 flex items-center justify-center">
                  <span className="text-2xl font-bold text-purple-600">
                    {position !== null ? position + 1 : "..."}
                  </span>
                </div>
              </div>
            </div>

            <h2 className="text-xl font-bold mb-2">排隊等候中</h2>
            <p className="text-gray-500 mb-4">
              您前方還有 <span className="font-bold text-purple-600">{position}</span> 人
            </p>
            <p className="text-sm text-gray-400">預估等待時間: {estimatedWait}</p>

            <div className="mt-8 bg-purple-50 rounded-lg p-4">
              <p className="text-xs text-purple-600">
                請勿關閉此頁面，輪到您時將自動通知
              </p>
            </div>
          </>
        )}

        {status === "your_turn" && (
          <>
            <div className="text-6xl mb-6">🎉</div>
            <h2 className="text-2xl font-bold text-purple-600 mb-4">
              輪到您了！
            </h2>
            <p className="text-gray-500 mb-6">
              請在 60 秒內進入選位頁面
            </p>
            <button
              onClick={handleEnterSelection}
              className="w-full bg-purple-600 text-white py-4 rounded-lg text-lg font-bold hover:bg-purple-700 transition animate-pulse"
            >
              進入選位
            </button>
          </>
        )}
      </div>
    </main>
  );
}
