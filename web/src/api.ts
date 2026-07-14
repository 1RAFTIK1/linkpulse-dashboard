import { getToken, logout } from "./auth";
import type { Link } from "./types";

// Запросы идут относительными путями — в dev их проксирует Vite (см.
// vite.config.ts), в проде — reverse-proxy перед сервисами.

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const token = getToken();
  const resp = await fetch(path, {
    ...init,
    headers: {
      ...init?.headers,
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
    },
  });
  if (resp.status === 401) {
    // Токен протух (TTL 1 час, refresh в MVP нет) — уводим на повторный логин.
    logout();
    throw new Error("сессия истекла");
  }
  if (!resp.ok) {
    const body = await resp.json().catch(() => null);
    throw new Error(body?.error ?? `${init?.method ?? "GET"} ${path}: ${resp.status}`);
  }
  return resp.json();
}

export function fetchLinks(): Promise<Link[]> {
  return request<Link[]>("/api/v1/links");
}

export function createLink(originalUrl: string): Promise<Link> {
  return request<Link>("/api/v1/links", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ original_url: originalUrl }),
  });
}
