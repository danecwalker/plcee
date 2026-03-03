import clsx from "clsx"
import { twMerge } from "tailwind-merge"

export const cn = (...classes: any[]) => {
  return twMerge(clsx(...classes))
}

export const requestSnapshot = async () => {
  const response = await fetch("/snapshot", { method: "GET" });
  if (!response.ok) {
    throw new Error("Failed to fetch snapshot");
  }
  const obj = await response.json();
  return obj
}

export const requestData = async () => {
  const response = await fetch("/data", { method: "GET" });
  if (!response.ok) {
    throw new Error("Failed to fetch data");
  }
  const obj = await response.json();
  return obj
}

export const sendCommand = async (command: string, value: any = null) => {
  const response = await fetch("/command", {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify({ Name: command, Data: value }),
  });

  const bodyText = await response.text();
  let body: any = null;
  if (bodyText) {
    try {
      body = JSON.parse(bodyText);
    } catch {
      body = null;
    }
  }

  if (!response.ok) {
    const message = body?.error || bodyText || "Failed to send command";
    throw new Error(message);
  }

  return body;
}

export const connectSSE = (onMessage: (data: any) => void, onOpen?: () => void): (() => void) => {
  const eventSource = new EventSource(`${import.meta.env.DEV ? "http://iriv.local:8080" : ""}/snapshot/stream?t=100`);

  eventSource.onmessage = (event) => {
    const data = JSON.parse(event.data);
    onMessage(data);
  };

  eventSource.onopen = () => {
    if (onOpen) onOpen();
  };

  return () => {
    eventSource.close();
  };
}

export const authenticate = async (password: string) => {
  const response = await fetch("/auth", {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify({ password }),
    credentials: "include",
  });
  if (!response.ok) {
    throw new Error("Authentication failed");
  }
}

export const logout = async () => {
  const response = await fetch("/auth", {
    method: "DELETE",
    credentials: "include",
  });
  if (!response.ok) {
    throw new Error("Logout failed");
  }
}
