import { NextRequest, NextResponse } from "next/server";
import { createHmac } from "crypto";

const REQUEST_SIGN_SECRET = process.env.REQUEST_SIGN_SECRET || "";
const BACKEND_URL = process.env.BACKEND_URL || "http://localhost:8080";

/**
 * Generate HMAC-SHA256 signature server-side.
 * Signs: method + path + timestamp (matching backend expectation).
 */
function signRequest(method: string, path: string, timestamp: string): string {
  if (!REQUEST_SIGN_SECRET) return "";
  const message = method + path + timestamp;
  return createHmac("sha256", REQUEST_SIGN_SECRET)
    .update(message)
    .digest("hex");
}

/**
 * BFF Proxy: forwards protected requests to backend with server-side HMAC signing.
 * This keeps REQUEST_SIGN_SECRET on the server — never exposed to the browser.
 */
async function proxyRequest(req: NextRequest, method: string) {
  const pathSegments = req.nextUrl.pathname.replace("/api/protected/", "");
  const backendPath = `/api/${pathSegments}`;
  const backendUrl = new URL(backendPath, BACKEND_URL);
  backendUrl.search = req.nextUrl.search;

  const timestamp = Date.now().toString();
  const signature = signRequest(method, backendPath, timestamp);

  const headers: Record<string, string> = {
    "X-Request-Signature": signature,
    "X-Request-Timestamp": timestamp,
  };

  // Pass through relevant headers from the original request
  const passHeaders = [
    "Authorization",
    "Content-Type",
    "X-Captcha-Token",
    "X-Device-Fingerprint",
  ];
  for (const name of passHeaders) {
    const value = req.headers.get(name);
    if (value) headers[name] = value;
  }

  const fetchOptions: RequestInit = {
    method,
    headers,
  };

  if (method === "POST" || method === "PUT" || method === "PATCH") {
    fetchOptions.body = await req.text();
  }

  const backendRes = await fetch(backendUrl.toString(), fetchOptions);
  const body = await backendRes.text();

  return new NextResponse(body, {
    status: backendRes.status,
    headers: { "Content-Type": backendRes.headers.get("Content-Type") || "application/json" },
  });
}

export async function GET(req: NextRequest) {
  return proxyRequest(req, "GET");
}

export async function POST(req: NextRequest) {
  return proxyRequest(req, "POST");
}
