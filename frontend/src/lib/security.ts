const REQUEST_SIGN_SECRET = process.env.NEXT_PUBLIC_REQUEST_SIGN_SECRET || "";

/**
 * Generate a simple device fingerprint based on browser properties.
 * This is a lightweight approach — for production, consider FingerprintJS.
 */
export function generateFingerprint(): string {
  if (typeof window === "undefined") return "";

  const components = [
    navigator.userAgent,
    navigator.language,
    screen.width + "x" + screen.height,
    screen.colorDepth.toString(),
    new Date().getTimezoneOffset().toString(),
    navigator.hardwareConcurrency?.toString() || "",
    (navigator as { deviceMemory?: number }).deviceMemory?.toString() || "",
  ];

  // Simple hash using string reduction
  const raw = components.join("|");
  let hash = 0;
  for (let i = 0; i < raw.length; i++) {
    const char = raw.charCodeAt(i);
    hash = ((hash << 5) - hash + char) | 0;
  }
  return Math.abs(hash).toString(36);
}

/**
 * Generate HMAC-SHA256 request signature.
 * Signs: method + path + timestamp
 */
export async function signRequest(
  method: string,
  path: string,
  timestamp: string
): Promise<string> {
  if (!REQUEST_SIGN_SECRET) return "";

  const message = method + path + timestamp;
  const encoder = new TextEncoder();
  const key = await crypto.subtle.importKey(
    "raw",
    encoder.encode(REQUEST_SIGN_SECRET),
    { name: "HMAC", hash: "SHA-256" },
    false,
    ["sign"]
  );
  const signature = await crypto.subtle.sign("HMAC", key, encoder.encode(message));
  return Array.from(new Uint8Array(signature))
    .map((b) => b.toString(16).padStart(2, "0"))
    .join("");
}
