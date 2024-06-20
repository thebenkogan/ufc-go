import { useQuery } from "@tanstack/react-query";
import type { EventWithPicks } from "./types";

const API_URL = "http://localhost:8080/";

async function callApi<T>(path: string, opts?: RequestInit): Promise<T> {
  const response = await fetch(`${API_URL}${path}`, {
    credentials: "include",
    ...opts,
  });
  if (!response.ok) {
    if (response.status === 401) {
      window.location.assign(API_URL + "login");
    }
    throw new Error(response.statusText);
  }
  return (await response.json().catch(() => null)) as T;
}

export function authCallback() {
  return callApi("auth/google/callback" + window.location.search);
}

export function useEventWithPicks(eventId: string) {
  return useQuery<EventWithPicks>({
    queryKey: [`events/${eventId}/picks`],
    queryFn: () =>
      callApi<EventWithPicks>(`events/${eventId}/picks?with_event=true`),
    staleTime: 1000 * 60 * 20,
  });
}

export function postPicks(eventId: string, picks: string[]) {
  return callApi(`events/${eventId}/picks`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify({ winners: picks }),
  });
}
