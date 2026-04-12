import { formatDistanceToNow, format, parseISO } from "date-fns";

export function timeAgo(dateStr: string): string {
  return formatDistanceToNow(parseISO(dateStr), { addSuffix: true });
}

export function formatDate(dateStr: string): string {
  return format(parseISO(dateStr), "MMM d, yyyy 'at' h:mm a");
}

export function formatDateShort(dateStr: string): string {
  return format(parseISO(dateStr), "MMM d, yyyy");
}
