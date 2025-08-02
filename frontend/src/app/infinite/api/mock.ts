import { LEVELS } from "@/constants/levels";
import { SyslogSchema } from "../schema";
import { subMinutes } from "date-fns";

const DAYS = 20;

// Syslog facilities
const SYSLOG_FACILITIES = [
  "kernel", "user", "mail", "daemon", "auth", "syslog", "lpr", "news",
  "uucp", "cron", "authpriv", "ftp", "ntp", "audit", "alert", "clock",
  "local0", "local1", "local2", "local3", "local4", "local5", "local6", "local7"
];

// Syslog severity levels
const SYSLOG_SEVERITIES = [
  "emergency", "alert", "critical", "error", "warning", "notice", "info", "debug"
];

// Application names for syslog
const APP_NAMES = [
  "nginx", "apache", "mysql", "postgresql", "redis", "docker", "kubernetes",
  "systemd", "sshd", "cron", "rsyslog", "logrotate", "fail2ban", "ufw"
];

// Hostnames for syslog
const HOSTNAMES = [
  "web-server-01", "db-server-01", "app-server-01", "load-balancer-01",
  "cache-server-01", "monitoring-01", "backup-server-01", "mail-server-01"
];

// Message IDs for syslog
const MSG_IDS = [
  "ID001", "ID002", "ID003", "ID004", "ID005", "ID006", "ID007", "ID008",
  "ID009", "ID010", "ID011", "ID012", "ID013", "ID014", "ID015"
];

// Sample syslog messages
const SYSLOG_MESSAGES = [
  "Connection from 192.168.1.100 established",
  "User authentication failed for user admin",
  "Database connection pool exhausted",
  "SSL certificate expired",
  "Disk space usage exceeded 90%",
  "Service nginx started successfully",
  "Failed to connect to external API",
  "Backup completed successfully",
  "Memory usage high: 85% utilized",
  "Network interface eth0 down",
  "Cron job completed with exit code 0",
  "Firewall rule added: block 10.0.0.0/8",
  "Log rotation completed",
  "System reboot initiated",
  "Package update available"
];

function getRandomFacility(): number {
  return Math.floor(Math.random() * 24);
}

function getRandomSeverity(): number {
  return Math.floor(Math.random() * 8);
}

function calculatePriority(facility: number, severity: number): number {
  return facility * 8 + severity;
}

function getLevelFromSeverity(severity: number): (typeof LEVELS)[number] {
  if (severity <= 1) return "error";
  if (severity <= 3) return "warning";
  return "success";
}

function getRandomStructuredData(): Record<string, string> | undefined {
  const rand = Math.random();
  if (rand < 0.3) {
    return undefined;
  } else if (rand < 0.5) {
    return {
      "src": "192.168.1.100",
      "dst": "10.0.0.1",
      "proto": "tcp"
    };
  } else if (rand < 0.7) {
    return {
      "user": "admin",
      "session": crypto.randomUUID().slice(0, 8),
      "action": "login"
    };
  } else {
    return {
      "service": "database",
      "connection": "pool-1",
      "status": "active"
    };
  }
}

function getRandomMessage(): string {
  return SYSLOG_MESSAGES[Math.floor(Math.random() * SYSLOG_MESSAGES.length)];
}

export function createMockSyslogData({
  minutes = 0,
}: {
  size?: number;
  minutes?: number;
}): SyslogSchema[] {
  const timestamp = subMinutes(new Date(), minutes);

  return Array.from({ length: 6 }, () => {
    const facility = getRandomFacility();
    const severity = getRandomSeverity();
    const priority = calculatePriority(facility, severity);
    const level = getLevelFromSeverity(severity);

    return {
      uuid: crypto.randomUUID(),
      facility,
      severity,
      priority,
      version: Math.random() > 0.5 ? 1 : 2,
      timestamp,
      hostname: HOSTNAMES[Math.floor(Math.random() * HOSTNAMES.length)],
      appName: APP_NAMES[Math.floor(Math.random() * APP_NAMES.length)],
      procId: Math.floor(Math.random() * 10000).toString(),
      msgId: MSG_IDS[Math.floor(Math.random() * MSG_IDS.length)],
      structuredData: getRandomStructuredData(),
      message: getRandomMessage(),
      level,
    };
  });
}

export const mock = Array.from({ length: DAYS * 24 })
  .map((_, i) => createMockSyslogData({ minutes: i * 60 }))
  .reduce((prev, curr) => prev.concat(curr), []) satisfies SyslogSchema[];

export const mockLive = Array.from({ length: 10 })
  .map((_, i) => createMockSyslogData({ minutes: -((i + 1) * 0.3) }))
  .reduce((prev, curr) => prev.concat(curr), [])
  .reverse() satisfies SyslogSchema[];
