import { useQuery } from "@tanstack/react-query";
import type { User, Event, Picks, PicksWithEvent, EventInfo } from "./types";

const API_URL = "http://localhost:8000/";

async function callApi<T>(path: string, opts?: RequestInit): Promise<T> {
	const response = await fetch(API_URL + path, {
		credentials: "include",
		...opts,
	});
	if (!response.ok) {
		if (response.status === 401) {
			window.location.assign("/");
		}
		throw new Error(response.statusText);
	}
	return (await response.json().catch(() => null)) as T;
}

export function startLogin() {
	window.location.assign(`${API_URL}login`);
}

export function authCallback() {
	return callApi(`auth/google/callback${window.location.search}`);
}

export function useUser() {
	return useQuery<User | null>({
		queryKey: ["user"],
		queryFn: async () => {
			const res = await fetch(`${API_URL}me`, {
				credentials: "include",
			});
			if (res.status === 401) {
				return null;
			}
			return res.json();
		},
		staleTime: Number.POSITIVE_INFINITY,
	});
}

export function useEvent(eventId: string) {
	return useQuery<Event>({
		queryKey: [`events/${eventId}`],
		queryFn: () => callApi<Event>(`events/${eventId}`),
		refetchInterval: 1000 * 60 * 5,
		staleTime: Number.POSITIVE_INFINITY,
	});
}

export function useSchedule() {
	return useQuery<EventInfo[]>({
		queryKey: ["schedule"],
		queryFn: () => callApi<EventInfo[]>("schedule"),
		refetchInterval: 1000 * 60 * 60,
		staleTime: Number.POSITIVE_INFINITY,
	});
}

export function usePicks(eventId: string) {
	return useQuery<Picks>({
		queryKey: [`events/${eventId}/picks`],
		queryFn: () => callApi<Picks>(`events/${eventId}/picks`),
		staleTime: Number.POSITIVE_INFINITY,
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

export function useAllPicks() {
	return useQuery<PicksWithEvent[]>({
		queryKey: ["events/picks"],
		queryFn: () => callApi<PicksWithEvent[]>("events/picks"),
	});
}
