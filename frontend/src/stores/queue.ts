import { create } from "zustand";

interface QueueState {
  position: number;
  estimatedWait: string;
  status: "idle" | "waiting" | "your_turn" | "selecting";
  ws: WebSocket | null;
  setQueueStatus: (position: number, estimatedWait: string) => void;
  setStatus: (status: QueueState["status"]) => void;
  setWs: (ws: WebSocket | null) => void;
  reset: () => void;
}

export const useQueueStore = create<QueueState>((set) => ({
  position: 0,
  estimatedWait: "",
  status: "idle",
  ws: null,

  setQueueStatus: (position, estimatedWait) =>
    set({ position, estimatedWait }),

  setStatus: (status) => set({ status }),

  setWs: (ws) => set({ ws }),

  reset: () => {
    set((state) => {
      state.ws?.close();
      return { position: 0, estimatedWait: "", status: "idle", ws: null };
    });
  },
}));
