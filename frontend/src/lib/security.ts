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
