"use client";

import { useEffect } from "react";
import { useAuthStore } from "@/stores/auth";

export default function AuthBootstrap() {
  const loadFromStorage = useAuthStore((s) => s.loadFromStorage);

  useEffect(() => {
    loadFromStorage();
  }, [loadFromStorage]);

  return null;
}
