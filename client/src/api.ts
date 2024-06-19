import { useQuery } from "@tanstack/react-query";
import type { Event, Picks } from "./types";

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

export function useEvent(eventId: string) {
  return useQuery<Event>({
    queryKey: [`events/${eventId}`],
    queryFn: () => callApi<Event>(`events/${eventId}`),
    staleTime: 1000 * 60 * 20,
  });
}

export function useEventPicks(eventId: string) {
  return useQuery<Picks>({
    queryKey: [`events/${eventId}/picks`],
    queryFn: () => callApi<Picks>(`events/${eventId}/picks`),
    staleTime: 1000 * 60 * 20,
    retry(failureCount, error) {
      if (error.message === "Not Found") {
        return false;
      }
      return failureCount < 3;
    },
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
