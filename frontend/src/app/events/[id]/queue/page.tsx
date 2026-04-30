"use client";

import { useEffect, useState, useRef, useCallback } from "react";
import { useParams, useRouter } from "next/navigation";
import { Turnstile } from "@marsidev/react-turnstile";
import { useAuthStore } from "@/stores/auth";
import { generateFingerprint } from "@/lib/security";
import { getWebSocketBaseURL } from "@/lib/ws";
import Navbar from "@/components/Navbar";

const TURNSTILE_SITE_KEY = process.env.NEXT_PUBLIC_TURNSTILE_SITE_KEY || "";

export default function QueuePage() {
  const params = useParams();
  const router = useRouter();
  const eventId = params.id as string;
  const user = useAuthStore((s) => s.user);
  const token = useAuthStore((s) => s.token);

  const [position, setPosition] = useState<number | null>(null);
  const [totalInQueue, setTotalInQueue] = useState(0);
  const [estimatedWait, setEstimatedWait] = useState("");
  const [status, setStatus] = useState<"captcha" | "joining" | "waiting" | "your_turn">(
    TURNSTILE_SITE_KEY ? "captcha" : "joining"
  );
  const [captchaToken, setCaptchaToken] = useState("");
  const [error, setError] = useState("");
  const wsRef = useRef<WebSocket | null>(null);
  const turnTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  const joinQueue = useCallback(async (captcha: string) => {
    if (!token) return;
    setStatus("joining");

    try {
      const fingerprint = generateFingerprint();

      // Request goes through BFF proxy which adds HMAC signature server-side
      const res = await fetch(`/api/protected/events/${eventId}/queue/join`, {
        method: "POST",
        headers: {
          Authorization: `Bearer ${token}`,
          "Content-Type": "application/json",
          "X-Captcha-Token": captcha,
          "X-Device-Fingerprint": fingerprint,
        },
      });
      const data = await res.json();
      if (!res.ok) {
        setError(data.error || "加入排隊失敗");
        return;
      }
      setPosition(data.position);
      setTotalInQueue(data.total_in_queue || data.position + 100);
      setEstimatedWait(data.estimated_wait);
      setStatus("waiting");
    } catch {
      setError("網路錯誤，請重試");
    }
  }, [eventId, token]);

  useEffect(() => {
    if (!token) {
      router.push("/auth");
      return;
    }

    // If no CAPTCHA configured, join immediately
    if (!TURNSTILE_SITE_KEY) {
      joinQueue("");
    }
  }, [token, router, joinQueue]);

  const handleCaptchaSuccess = (captcha: string) => {
    setCaptchaToken(captcha);
    joinQueue(captcha);
  };

  useEffect(() => {
    if (status !== "waiting" || !user) return;

    const wsBase = getWebSocketBaseURL();
    const wsUrl = `${wsBase}/ws?event_id=${eventId}&token=${encodeURIComponent(token || "")}`;

    const connect = () => {
      const ws = new WebSocket(wsUrl);
      wsRef.current = ws;

      ws.onmessage = (event) => {
        try {
          const msg = JSON.parse(event.data);
          if (msg.type === "queue_update") {
            setPosition(msg.data.position);
            setTotalInQueue(msg.data.total_in_queue || totalInQueue);
            setEstimatedWait(msg.data.estimated_wait);
            if (msg.data.status === "your_turn") {
              setStatus("your_turn");
              turnTimerRef.current = setTimeout(() => {
                setError("您未在時間內進入選位頁面，請重新排隊");
              }, 60000);
            }
          }
        } catch {
          // ignore parse errors
        }
      };

      ws.onclose = () => {
        setTimeout(() => {
          if (wsRef.current === ws) {
            connect();
          }
        }, 3000);
      };
    };

    connect();

    return () => {
      const ws = wsRef.current;
      wsRef.current = null;
      ws?.close();
      if (turnTimerRef.current) clearTimeout(turnTimerRef.current);
    };
  }, [status, user, eventId, token, totalInQueue]);

  const handleEnterSelection = () => {
    if (turnTimerRef.current) clearTimeout(turnTimerRef.current);
    wsRef.current?.close();
    router.push(`/events/${eventId}/select`);
  };

  if (error) {
    return (
      <div className="flex flex-col h-full">
        <Navbar />
        <div className="flex-1 flex items-center justify-center">
          <div className="bg-[var(--bg-card)] rounded-[var(--radius)] p-8 text-center max-w-md">
            <p className="font-mono text-[var(--status-red)] text-lg mb-4">{error}</p>
            <button
              onClick={() => window.location.reload()}
              className="bg-[var(--accent-orange)] text-[var(--text-on-accent)] font-mono text-sm font-semibold px-6 py-2 rounded-[var(--radius)] hover:brightness-110 transition"
            >
              重試
            </button>
          </div>
        </div>
      </div>
    );
  }

  const positionDisplay = position !== null ? (position + 1).toLocaleString() : "...";
  const peopleAhead = position !== null ? position.toLocaleString() : "...";
  const progressPercent = position !== null && totalInQueue > 0
    ? Math.max(2, ((totalInQueue - position) / totalInQueue) * 100)
    : 0;

  return (
    <div className="flex flex-col h-full">
      <Navbar />
      <main className="flex-1 flex flex-col items-center justify-center gap-10 px-12 py-10">
        {status === "captcha" && (
          <div className="flex flex-col items-center gap-6">
            <h1 className="font-display text-3xl font-bold">VERIFY</h1>
            <p className="font-mono text-[13px] text-[var(--text-secondary)]">
              // 請完成人機驗證以加入排隊
            </p>
            <Turnstile
              siteKey={TURNSTILE_SITE_KEY}
              onSuccess={handleCaptchaSuccess}
              options={{ theme: "dark" }}
            />
          </div>
        )}

        {status === "joining" && (
          <>
            <div className="w-20 h-20 bg-[var(--bg-card)] rounded-full flex items-center justify-center">
              <div className="w-9 h-9 border-4 border-[var(--accent-orange)] border-t-transparent rounded-full animate-spin" />
            </div>
            <span className="font-mono text-[var(--text-secondary)]">// joining_queue...</span>
          </>
        )}

        {status === "waiting" && (
          <>
            <div className="w-20 h-20 bg-[var(--bg-card)] rounded-full flex items-center justify-center">
              <div className="w-9 h-9 border-4 border-[var(--accent-orange)] border-t-transparent rounded-full animate-spin" />
            </div>

            <div className="text-center">
              <h1 className="font-display text-4xl font-bold">WAITING ROOM</h1>
              <p className="font-mono text-[13px] text-[var(--text-secondary)] mt-2">
                // 排隊等候中
              </p>
            </div>

            {/* Stats */}
            <div className="flex gap-8">
              <div className="w-[200px] bg-[var(--bg-card)] rounded-[var(--radius)] p-6 flex flex-col items-center gap-2">
                <span className="font-mono text-[11px] text-[var(--text-secondary)]">// your_position</span>
                <span className="font-display text-5xl font-bold text-[var(--accent-orange)]">{positionDisplay}</span>
                <span className="font-mono text-[11px] text-[var(--text-secondary)]">of {totalInQueue.toLocaleString()} in queue</span>
              </div>
              <div className="w-[200px] bg-[var(--bg-card)] rounded-[var(--radius)] p-6 flex flex-col items-center gap-2">
                <span className="font-mono text-[11px] text-[var(--text-secondary)]">// est_wait_time</span>
                <span className="font-display text-5xl font-bold text-[var(--accent-teal)]">~{estimatedWait || "?"}</span>
                <span className="font-mono text-[11px] text-[var(--text-secondary)]">minutes remaining</span>
              </div>
              <div className="w-[200px] bg-[var(--bg-card)] rounded-[var(--radius)] p-6 flex flex-col items-center gap-2">
                <span className="font-mono text-[11px] text-[var(--text-secondary)]">// people_ahead</span>
                <span className="font-display text-5xl font-bold">{peopleAhead}</span>
                <span className="font-mono text-[11px] text-[var(--text-secondary)]">users before you</span>
              </div>
            </div>

            {/* Progress bar */}
            <div className="w-[664px] flex flex-col items-center gap-3">
              <div className="w-full h-2 bg-[var(--bg-card)] rounded-full overflow-hidden">
                <div
                  className="h-full rounded-full bg-gradient-to-r from-[var(--accent-orange)] to-[var(--accent-teal)] transition-all duration-1000"
                  style={{ width: `${progressPercent}%` }}
                />
              </div>
              <div className="flex items-center gap-2">
                <div className="w-1.5 h-1.5 rounded-full bg-[var(--accent-teal)]" />
                <span className="font-mono text-[11px] text-[var(--text-secondary)]">
                  // connection_active
                </span>
              </div>
            </div>

            {/* Warning */}
            <div className="flex items-center gap-2 bg-[var(--bg-card)] rounded-xl px-5 py-2.5">
              <span className="text-[var(--status-yellow)]">⚠</span>
              <span className="font-mono text-xs text-[var(--text-secondary)]">
                請勿關閉此頁面，離開超過 30 秒將失去排隊資格
              </span>
            </div>
          </>
        )}

        {status === "your_turn" && (
          <>
            <div className="w-24 h-24 rounded-full border-2 border-[var(--accent-teal)] bg-[#00D4AA22] flex items-center justify-center">
              <span className="text-[var(--accent-teal)] text-4xl">✓</span>
            </div>

            <div className="text-center">
              <h1 className="font-display text-4xl font-bold text-[var(--accent-teal)]">
                YOUR TURN
              </h1>
              <p className="font-mono text-[13px] text-[var(--text-secondary)] mt-2">
                // 請在 60 秒內進入選位頁面
              </p>
            </div>

            <button
              onClick={handleEnterSelection}
              className="w-80 h-[52px] bg-[var(--accent-orange)] text-[var(--text-on-accent)] rounded-[var(--radius)] font-display text-lg font-semibold hover:brightness-110 transition animate-pulse"
            >
              進入選位
            </button>
          </>
        )}
      </main>
    </div>
  );
}
