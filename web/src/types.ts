// Типы зеркалят JSON-контракты бэкендов.
// Все id — строки: Snowflake int64 не влезает в Number без потери точности.

export interface Link {
  id: string;
  short_code: string;
  short_url: string;
  original_url: string;
  created_at: string;
  expires_at?: string;
}

export interface ClickData {
  event_id: string;
  link_id: string;
  short_code: string;
  original_url: string;
  clicked_at: string;
  referrer: string;
  country: string;
  user_agent: string;
}

// Исходящие сообщения WS-протокола (спека §8).
export type ClientMessage =
  | { type: "auth"; token: string }
  | { type: "subscribe"; link_id: string }
  | { type: "unsubscribe"; link_id: string };

// Входящие сообщения.
export type ServerMessage =
  | { type: "auth_ok" }
  | { type: "error"; error: string; link_id?: string }
  | { type: "click"; data: ClickData }
  | { type: "shutdown" };

export type WsStatus = "connecting" | "live" | "reconnecting" | "error";
