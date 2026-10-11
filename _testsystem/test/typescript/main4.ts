export async function fetchJSON1<T>(url: string): Promise<T> {
  const res = await fetch(url);
  if (!res.ok) throw new Error(`fetch failed: ${res.status}`);
  return res.json() as Promise<T>;
}

export async function postJSON1<T>(url: string, body: unknown): Promise<T> {
  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
  if (!res.ok) throw new Error(`post failed: ${res.status}`);
  return res.json() as Promise<T>;
}

export async function deleteResource1(url: string): Promise<void> {
  const res = await fetch(url, { method: "DELETE" });
  if (!res.ok) throw new Error(`delete failed: ${res.status}`);
}
