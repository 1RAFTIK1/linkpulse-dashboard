import { useEffect, useRef, useState } from "react";
import { getToken } from "./auth";
import type { ClickData, ClientMessage, ServerMessage, WsStatus } from "./types";

const MAX_FEED = 50; // сколько последних кликов держим в ленте
const RECONNECT_DELAY_MS = 2000;

// useLiveClicks — жизненный цикл WS-подписки на живые клики одной ссылки:
// connect → auth → subscribe → приём кликов; авто-переподключение при
// {"type":"shutdown"} (плановая остановка сервера) и при обрыве.
export function useLiveClicks(linkId: string | null) {
  const [clicks, setClicks] = useState<ClickData[]>([]);
  const [status, setStatus] = useState<WsStatus>("connecting");
  const wsRef = useRef<WebSocket | null>(null);

  useEffect(() => {
    if (!linkId) return;

    let closedByUs = false;
    let reconnectTimer: number | undefined;
    setClicks([]);

    const connect = () => {
      setStatus((s) => (s === "connecting" ? s : "reconnecting"));
      const proto = location.protocol === "https:" ? "wss" : "ws";
      const ws = new WebSocket(`${proto}://${location.host}/ws`);
      wsRef.current = ws;

      const send = (msg: ClientMessage) => ws.send(JSON.stringify(msg));

      ws.onopen = () => {
        // Настоящий JWT из OAuth-логина; если его нет (dev без Auth),
        // бэкенд-заглушка примет любой непустой токен.
        send({ type: "auth", token: getToken() ?? "dev-stub-token" });
      };

      ws.onmessage = (raw) => {
        const msg: ServerMessage = JSON.parse(raw.data);
        switch (msg.type) {
          case "auth_ok":
            setStatus("live");
            send({ type: "subscribe", link_id: linkId });
            break;
          case "click":
            setClicks((prev) => [msg.data, ...prev].slice(0, MAX_FEED));
            break;
          case "shutdown":
            // Сервер честно предупредил — переподключаемся, не считаем ошибкой.
            setStatus("reconnecting");
            break;
          case "error":
            console.warn("ws error message:", msg.error);
            setStatus("error");
            break;
        }
      };

      ws.onclose = () => {
        if (closedByUs) return;
        setStatus("reconnecting");
        reconnectTimer = window.setTimeout(connect, RECONNECT_DELAY_MS);
      };
    };

    connect();

    return () => {
      closedByUs = true;
      window.clearTimeout(reconnectTimer);
      wsRef.current?.close();
    };
  }, [linkId]);

  return { clicks, status };
}
