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
