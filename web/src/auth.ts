// Управление токеном на фронтенде.
//
// Auth service после OAuth возвращает браузер на FRONTEND_URL#token=<jwt> —
// токен во fragment: он не уходит на сервер и не оседает в access-логах.
// Здесь мы его снимаем с URL, прячем в localStorage и чистим адресную строку.

const STORAGE_KEY = "linkpulse_token";

export function captureTokenFromURL(): void {
  const match = location.hash.match(/#token=([^&]+)/);
  if (!match) return;
  localStorage.setItem(STORAGE_KEY, match[1]);
  // Убираем токен из адресной строки (и из истории браузера).
  history.replaceState(null, "", location.pathname + location.search);
}

export function getToken(): string | null {
  return localStorage.getItem(STORAGE_KEY);
}

export function logout(): void {
  localStorage.removeItem(STORAGE_KEY);
  location.reload();
}

// loginURL — эндпоинт Auth service; в dev проксируется vite (/auth → :8083).
export const loginURL = "/auth/github/login";
