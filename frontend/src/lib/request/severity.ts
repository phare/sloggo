import { SEVERITY_VALUES } from "@/constants/severity";
import { cn } from "../utils";

export function getSeverityColor(
  value: (typeof SEVERITY_VALUES)[number],
): Record<"text" | "bg" | "border", string> {
  switch (value) {
    case "emergency":
      return {
        text: "text-emergency",
        bg: "bg-emergency",
        border: "border-emergency",
      };
    case "alert":
      return {
        text: "text-alert",
        bg: "bg-alert",
        border: "border-alert",
      };
    case "critical":
      return {
        text: "text-critical",
        bg: "bg-critical",
        border: "border-critical",
      };
    case "error":
      return {
        text: "text-error",
        bg: "bg-error",
        border: "border-error",
      };
    case "warning":
      return {
        text: "text-warning",
        bg: "bg-warning",
        border: "border-warning",
      };
    case "notice":
      return {
        text: "text-notice",
        bg: "bg-notice",
        border: "border-notice",
      };
    case "info":
      return {
        text: "text-info",
        bg: "bg-info",
        border: "border-info",
      };
    case "debug":
    default:
      return {
        text: "text-debug",
        bg: "bg-debug",
        border: "border-debug",
      };
  }
}

export function getSeverityRowClassName(
  value: (typeof SEVERITY_VALUES)[number],
): string {
  switch (value) {
    case "emergency":
      return cn(
        "bg-red-600/10 hover:bg-red-600/20 focus-visible:bg-red-600/20 data-[state=selected]:bg-red-600/30",
        "dark:bg-red-600/20 dark:hover:bg-red-600/30 dark:focus-visible:bg-red-600/30 dark:data-[state=selected]:bg-red-600/40",
      );
    case "alert":
      return cn(
        "bg-red-500/10 hover:bg-red-500/20 focus-visible:bg-red-500/20 data-[state=selected]:bg-red-500/30",
        "dark:bg-red-500/20 dark:hover:bg-red-500/30 dark:focus-visible:bg-red-500/30 dark:data-[state=selected]:bg-red-500/40",
      );
    case "critical":
      return cn(
        "bg-red-400/10 hover:bg-red-400/20 focus-visible:bg-red-400/20 data-[state=selected]:bg-red-400/30",
        "dark:bg-red-400/20 dark:hover:bg-red-400/30 dark:focus-visible:bg-red-400/30 dark:data-[state=selected]:bg-red-400/40",
      );
    case "error":
      return cn(
        "bg-error/5 hover:bg-error/10 focus-visible:bg-error/10 data-[state=selected]:bg-error/20",
        "dark:bg-error/10 dark:hover:bg-error/20 dark:focus-visible:bg-error/20 dark:data-[state=selected]:bg-error/30",
      );
    case "warning":
      return cn(
        "bg-warning/5 hover:bg-warning/10 focus-visible:bg-warning/10 data-[state=selected]:bg-warning/20",
        "dark:bg-warning/10 dark:hover:bg-warning/20 dark:focus-visible:bg-warning/20 dark:data-[state=selected]:bg-warning/30",
      );
    case "notice":
      return cn(
        "bg-yellow-500/5 hover:bg-yellow-500/10 focus-visible:bg-yellow-500/10 data-[state=selected]:bg-yellow-500/20",
        "dark:bg-yellow-500/10 dark:hover:bg-yellow-500/20 dark:focus-visible:bg-yellow-500/20 dark:data-[state=selected]:bg-yellow-500/30",
      );
    case "info":
      return "";
    case "debug":
      return cn(
        "bg-info/5 hover:bg-info/10 focus-visible:bg-info/10 data-[state=selected]:bg-info/20",
        "dark:bg-info/10 dark:hover:bg-info/20 dark:focus-visible:bg-info/20 dark:data-[state=selected]:bg-info/30",
      );
    default:
      return "";
  }
}
