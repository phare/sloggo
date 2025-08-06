// RFC 5424 Syslog Severity Levels
export const SEVERITY_VALUES = [
  "emergency",
  "alert",
  "critical",
  "error",
  "warning",
  "notice",
  "info",
  "debug",
] as const;

export const SEVERITY_LABELS = [
  "Emergency",
  "Alert",
  "Critical",
  "Error",
  "Warning",
  "Notice",
  "Info",
  "Debug",
] as const;
