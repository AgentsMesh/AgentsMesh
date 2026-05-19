import { type ClassValue, clsx } from "clsx";
import { twMerge } from "tailwind-merge";
import { ApiError } from "@/lib/api/api-types";

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}

export function getErrorMessage(error: unknown, fallback: string): string {
  if (error instanceof ApiError) {
    return error.serverMessage || fallback;
  }
  if (error instanceof Error) {
    return error.message;
  }
  if (typeof error === "object" && error !== null && "message" in error) {
    return (error as { message: string }).message;
  }
  return fallback;
}
