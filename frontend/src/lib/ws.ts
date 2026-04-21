export function getWebSocketBaseURL(): string {
  if (typeof window === "undefined") {
    return process.env.NEXT_PUBLIC_WS_URL || "ws://localhost:8080";
  }

  if (process.env.NEXT_PUBLIC_WS_URL) {
    return process.env.NEXT_PUBLIC_WS_URL;
  }

  const wsProtocol = window.location.protocol === "https:" ? "wss:" : "ws:";
  const { hostname, port } = window.location;

  // Local dev commonly serves Next.js on 3000 and Go API/WebSocket on 8080.
  if (hostname === "localhost" || hostname === "127.0.0.1") {
    return `${wsProtocol}//${hostname}:8080`;
  }

  return `${wsProtocol}//${window.location.host}`;
}
