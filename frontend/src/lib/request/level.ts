import { _LEVELS } from "@/constants/levels";
import { cn } from "../utils";

export function getLevelColor(
  value: (typeof _LEVELS)[number]
): Record<"text" | "bg" | "border", string> {
  switch (value) {
    case "info":
      return {
        text: "text-blue-500",
        bg: "bg-blue-500",
        border: "border-blue-500",
      };
    case "warning":
      return {
        text: "text-warning",
        bg: "bg-warning",
        border: "border-warning",
      };
    case "error":
      return {
        text: "text-error",
        bg: "bg-error",
        border: "border-error",
      };
    case "debug":
    default:
      return {
        text: "text-info",
        bg: "bg-info",
        border: "border-info",
      };
  }
}

export function getLevelRowClassName(value: (typeof _LEVELS)[number]): string {
  switch (value) {
    case "info":
      return "";
    case "warning":
      return cn(
        "bg-warning/5 hover:bg-warning/10 data-[state=selected]:bg-warning/20 focus-visible:bg-warning/10",
        "dark:bg-warning/10 dark:hover:bg-warning/20 dark:data-[state=selected]:bg-warning/30 dark:focus-visible:bg-warning/20"
      );
    case "error":
      return cn(
        "bg-error/5 hover:bg-error/10 data-[state=selected]:bg-error/20 focus-visible:bg-error/10",
        "dark:bg-error/10 dark:hover:bg-error/20 dark:data-[state=selected]:bg-error/30 dark:focus-visible:bg-error/20"
      );
    case "debug":
      return cn(
        "bg-info/5 hover:bg-info/10 data-[state=selected]:bg-info/20 focus-visible:bg-info/10",
        "dark:bg-info/10 dark:hover:bg-info/20 dark:data-[state=selected]:bg-info/30 dark:focus-visible:bg-info/20"
      );
    default:
      return "";
  }
}

export function getLevelLabel(value: (typeof _LEVELS)[number]): string {
  switch (value) {
    case "info":
      return "Info";
    case "warning":
      return "Warning";
    case "error":
      return "Error";
    case "debug":
      return "Debug";
    default:
      return "Unknown";
  }
}
