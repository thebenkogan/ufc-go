import { useQuery } from "@tanstack/react-query";
import type { Event } from "./types";

const API_URL = "http://localhost:8080/";

export async function get<T>(path: string): Promise<T> {
  const response = await fetch(`${API_URL}${path}`, {
    credentials: "include",
  });
  if (!response.ok) {
    if (response.status === 401) {
      window.location.assign(API_URL + "login");
    }
    throw new Error(response.statusText);
  }
  return (await response.json().catch(() => null)) as T;
}

export function useLatestEvent() {
  const { data, error } = useQuery<Event>({
    queryKey: ["events/latest"],
    queryFn: () => get<Event>("events/latest"),
    staleTime: 1000 * 60 * 20,
  });
  return { data, error };
}
