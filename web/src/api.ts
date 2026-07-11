import type { Link } from "./types";

// Запросы идут относительными путями — в dev их проксирует Vite (см.
// vite.config.ts), в проде — reverse-proxy перед сервисами.

export async function fetchLinks(): Promise<Link[]> {
  const resp = await fetch("/api/v1/links");
  if (!resp.ok) throw new Error(`GET /links: ${resp.status}`);
  return resp.json();
}

export async function createLink(originalUrl: string): Promise<Link> {
  const resp = await fetch("/api/v1/links", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ original_url: originalUrl }),
  });
  if (!resp.ok) {
    const body = await resp.json().catch(() => null);
    throw new Error(body?.error ?? `POST /links: ${resp.status}`);
  }
  return resp.json();
}
