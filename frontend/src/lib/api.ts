const API_BASE = "/api";

async function fetchAPI<T>(path: string, options?: RequestInit): Promise<T> {
  const token =
    typeof window !== "undefined" ? localStorage.getItem("token") : null;

  const headers: Record<string, string> = {
    "Content-Type": "application/json",
    ...(options?.headers as Record<string, string>),
  };

  if (token) {
    headers["Authorization"] = `Bearer ${token}`;
  }

  const res = await fetch(`${API_BASE}${path}`, {
    ...options,
    headers,
  });

  if (!res.ok) {
    const body = await res.json().catch(() => ({}));
    throw new Error(body.error || `API Error: ${res.status}`);
  }

  return res.json();
}

export const api = {
  // Events
  listEvents: () => fetchAPI<{ events: EventItem[] }>("/events"),
  getEvent: (id: string) => fetchAPI<EventDetail>(`/events/${id}`),
  getAvailability: (id: string) =>
    fetchAPI<{ sections: SectionAvailability[] }>(
      `/events/${id}/availability`
    ),

  // Auth
  register: (data: { email: string; password: string; name: string }) =>
    fetchAPI<{ user: User; token: string }>("/auth/register", {
      method: "POST",
      body: JSON.stringify(data),
    }),
  login: (data: { email: string; password: string }) =>
    fetchAPI<{ user: User; token: string }>("/auth/login", {
      method: "POST",
      body: JSON.stringify(data),
    }),

  // Seats
  allocateSeats: (
    eventId: string,
    data: { section_id: string; quantity: number }
  ) =>
    fetchAPI<AllocatedSeats>(`/events/${eventId}/allocate`, {
      method: "POST",
      body: JSON.stringify(data),
    }),

  // Orders
  createOrder: (data: {
    event_id: string;
    seats: SeatInfo[];
    price_per_seat: number;
  }) =>
    fetchAPI<CreateOrderResponse>("/orders", {
      method: "POST",
      body: JSON.stringify(data),
    }),
  listOrders: () => fetchAPI<{ orders: Order[] }>("/orders"),
  getOrder: (id: string) =>
    fetchAPI<{ order: Order; items: OrderItem[] }>(`/orders/${id}`),
};

// Types
export interface EventItem {
  id: string;
  title: string;
  event_date: string;
  venue_name: string;
  price_range: string;
  sale_status: string;
  sale_start: string;
  image_url: string;
}

export interface EventDetail {
  id: string;
  title: string;
  event_date: string;
  sale_start: string;
  sale_end: string;
  status: string;
  venue_name: string;
  layout_data: unknown;
  sections: EventSectionDetail[];
}

export interface EventSectionDetail {
  id: string;
  event_id: string;
  section_id: string;
  price: number;
  quota: number;
  section_name: string;
  polygon: number[][];
  remaining: number;
}

export interface SectionAvailability {
  section_id: string;
  name: string;
  remaining: number;
  quota: number;
}

export interface User {
  id: string;
  email: string;
  name: string;
}

export interface AllocatedSeats {
  session_id: string;
  seats: SeatInfo[];
  expires_at: string;
}

export interface SeatInfo {
  event_seat_id: string;
  section_name: string;
  row_label: string;
  seat_number: number;
}

export interface CreateOrderResponse {
  id: string;
  status: string;
  total: number;
  payment_url: string;
  transaction_id: number;
}

export interface Order {
  id: string;
  user_id: string;
  event_id: string;
  status: string;
  total: number;
  created_at: string;
}

export interface OrderItem {
  id: string;
  order_id: string;
  event_seat_id: string;
  section_name: string;
  row_label: string;
  seat_number: number;
  price: number;
}
