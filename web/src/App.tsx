import { useEffect, useState } from "react";
import { createLink, fetchLinks } from "./api";
import { captureTokenFromURL, getToken, loginURL, logout } from "./auth";
import { Chart } from "./Chart";
import type { Link } from "./types";
import { useLiveClicks } from "./useLiveClicks";

// Снимаем токен из #fragment ДО первого рендера (модульный top-level код).
captureTokenFromURL();

// authRequired управляется сборкой стека: dev-режим без Auth service
// работает по заглушкам, полный стек требует логин через GitHub.
const authRequired = import.meta.env.VITE_AUTH_REQUIRED === "true";

const statusLabel: Record<string, string> = {
  connecting: "подключение…",
  live: "live",
  reconnecting: "переподключение…",
  error: "ошибка",
};

export default function App() {
  const [links, setLinks] = useState<Link[]>([]);
  const [selected, setSelected] = useState<Link | null>(null);
  const [newUrl, setNewUrl] = useState("");
  const [formError, setFormError] = useState<string | null>(null);
  const { clicks, status } = useLiveClicks(selected?.id ?? null);

  const authed = !authRequired || getToken() !== null;

  useEffect(() => {
    if (!authed) return; // без токена не дёргаем API — всё равно будет 401
    fetchLinks()
      .then((ls) => {
        setLinks(ls);
        if (ls.length > 0) setSelected(ls[0]);
      })
      .catch((e) => setFormError(String(e)));
  }, [authed]);

  // Логин-гейт: return строго ПОСЛЕ всех хуков (правила хуков React —
  // одинаковый порядок вызовов на каждом рендере).
  if (!authed) {
    return (
      <div className="layout login-screen">
        <h1>LinkPulse</h1>
        <p>Короткие ссылки с live-аналитикой кликов.</p>
        <a className="login-btn" href={loginURL}>
          Войти через GitHub
        </a>
      </div>
    );
  }

  async function onCreate(e: React.FormEvent) {
    e.preventDefault();
    setFormError(null);
    try {
      const link = await createLink(newUrl);
      setLinks((prev) => [link, ...prev]);
      setSelected(link);
      setNewUrl("");
    } catch (err) {
      setFormError(err instanceof Error ? err.message : String(err));
    }
  }

  return (
    <div className="layout">
      <header>
        <h1>
          LinkPulse <span className="pulse-dot" data-status={status} />
        </h1>
        <span className="ws-status">{statusLabel[status]}</span>
        {authRequired && (
          <button className="logout" onClick={logout}>
            выйти
          </button>
        )}
      </header>

      <section className="create">
        <form onSubmit={onCreate}>
          <input
            type="url"
            required
            placeholder="https://example.com/very/long/url"
            value={newUrl}
            onChange={(e) => setNewUrl(e.target.value)}
          />
          <button type="submit">Сократить</button>
        </form>
        {formError && <p className="error">{formError}</p>}
      </section>

      <div className="columns">
        <aside>
          <h2>Мои ссылки</h2>
          <ul className="links">
            {links.map((l) => (
              <li key={l.id}>
                <button
                  className={selected?.id === l.id ? "selected" : ""}
                  onClick={() => setSelected(l)}
                >
                  <code>/{l.short_code}</code>
                  <small>{l.original_url}</small>
                </button>
              </li>
            ))}
            {links.length === 0 && <li className="empty">пока пусто — создай первую</li>}
          </ul>
        </aside>

        <main>
          {selected ? (
            <>
              <div className="panel-head">
                <h2>
                  <a href={selected.short_url} target="_blank" rel="noreferrer">
                    {selected.short_url}
                  </a>
                </h2>
                <div className="counter">
                  <strong>{clicks.length}</strong>
                  <span>кликов за сессию</span>
                </div>
              </div>
              <Chart clicks={clicks} />
              <h3>Лента</h3>
              <ul className="feed">
                {clicks.map((c) => (
                  <li key={c.event_id}>
                    <time>{new Date(c.clicked_at).toLocaleTimeString()}</time>
                    <span className="ref">{c.referrer || "прямой переход"}</span>
                    {c.country && <span className="country">{c.country}</span>}
                  </li>
                ))}
                {clicks.length === 0 && <li className="empty">ждём кликов…</li>}
              </ul>
            </>
          ) : (
            <p className="empty">Выбери ссылку слева</p>
          )}
        </main>
      </div>
    </div>
  );
}
