const net = require("net");

// Configuration
const CONFIG = {
  HOST: "localhost",
  PORT: 6514, // TCP syslog port
  NUM_LOGS: 10000,
  BATCH_SIZE: 100,
  DELAY_BETWEEN_BATCHES: 100, // ms
  RECONNECT_DELAY: 1000, // ms
  MAX_RETRIES: 3,
  KEEPALIVE_DELAY: 1000, // ms
};

// Syslog facilities and their codes
const FACILITIES = {
  kern: 0, // kernel messages
  user: 1, // user-level messages
  mail: 2, // mail system
  daemon: 3, // system daemons
  auth: 4, // security/authorization messages
  syslog: 5, // messages generated internally by syslogd
  lpr: 6, // line printer subsystem
  news: 7, // network news subsystem
  uucp: 8, // UUCP subsystem
  cron: 9, // clock daemon
  authpriv: 10, // security/authorization messages
  ftp: 11, // FTP daemon
  local0: 16,
  local1: 17,
  local2: 18,
  local3: 19,
  local4: 20,
  local5: 21,
  local6: 22,
  local7: 23,
};

// Syslog severities
const SEVERITIES = {
  emerg: 0, // system is unusable
  alert: 1, // action must be taken immediately
  crit: 2, // critical conditions
  err: 3, // error conditions
  warning: 4, // warning conditions
  notice: 5, // normal but significant condition
  info: 6, // informational messages
  debug: 7, // debug-level messages
};

// Sample data
const APPS = [
  "nginx",
  "postgresql",
  "mongodb",
  "redis",
  "api-server",
  "auth-service",
  "user-service",
  "payment-service",
  "notification-service",
  "background-worker",
];

const HOSTS = [
  "prod-web-01",
  "prod-web-02",
  "prod-db-01",
  "prod-cache-01",
  "staging-web-01",
  "staging-db-01",
  "dev-server-01",
];

const STRUCTURED_DATA = [
  "-",
  '[exampleSDID@32473 iut="3" eventSource="Application" eventID="%d"]',
  '[origin@6875 ip="%s" port="%d"]',
  '[meta@1234 version="1.1" timestamp="%s"]',
];

const MESSAGE_TEMPLATES = [
  "User %s logged in successfully",
  "Failed login attempt for user %s from IP %s",
  "Database connection error: %s",
  "API request completed in %dms",
  "Cache miss for key: %s",
  "Memory usage at %d%%",
  "CPU load average: %d",
  "Disk space warning: %d%% used on /dev/%s",
  "Background job %s completed in %ds",
  "Service %s restarted successfully",
  "New user registered: %s",
  "Payment processed: $%d.%d",
  "Email sent to %s",
  "Config changed: %s = %s",
  "Security alert: %s",
];

class SyslogClient {
  constructor() {
    this.client = null;
    this.connected = false;
    this.reconnecting = false;
    this.keepaliveTimer = null;
  }

  async connect() {
    if (this.client) {
      this.cleanup();
    }

    this.client = new net.Socket();
    this.setupEventHandlers();

    return new Promise((resolve, reject) => {
      const timeout = setTimeout(() => {
        reject(new Error("Connection timeout"));
      }, 5000);

      this.client.connect(CONFIG.PORT, CONFIG.HOST, () => {
        clearTimeout(timeout);
        this.connected = true;
        this.startKeepalive();
        resolve();
      });
    });
  }

  setupEventHandlers() {
    this.client.on("error", this.handleError.bind(this));
    this.client.on("close", this.handleClose.bind(this));
    this.client.on("end", this.handleEnd.bind(this));

    // Set TCP keepalive
    this.client.setKeepAlive(true, CONFIG.KEEPALIVE_DELAY);
  }

  handleError(err) {
    console.error("Socket error:", err.message);
    this.cleanup();
  }

  handleClose() {
    console.log("Connection closed");
    this.cleanup();
  }

  handleEnd() {
    console.log("Connection ended");
    this.cleanup();
  }

  cleanup() {
    this.connected = false;
    if (this.keepaliveTimer) {
      clearInterval(this.keepaliveTimer);
      this.keepaliveTimer = null;
    }
    if (this.client) {
      this.client.removeAllListeners();
      this.client.destroy();
      this.client = null;
    }
  }

  startKeepalive() {
    if (this.keepaliveTimer) {
      clearInterval(this.keepaliveTimer);
    }
    this.keepaliveTimer = setInterval(() => {
      if (this.connected && this.client.writable) {
        // Send an empty message as keepalive
        this.client.write("\n");
      }
    }, CONFIG.KEEPALIVE_DELAY);
  }

  async write(data) {
    if (!this.connected || !this.client || !this.client.writable) {
      throw new Error("Not connected");
    }

    return new Promise((resolve, reject) => {
      this.client.write(data, (err) => {
        if (err) reject(err);
        else resolve();
      });
    });
  }
}

// Helper functions
function randomElement(array) {
  return array[Math.floor(Math.random() * array.length)];
}

function randomInt(min, max) {
  return Math.floor(Math.random() * (max - min + 1)) + min;
}

function randomIP() {
  return `${randomInt(1, 255)}.${randomInt(0, 255)}.${randomInt(0, 255)}.${randomInt(0, 255)}`;
}

function randomUsername() {
  const names = [
    "alice",
    "bob",
    "charlie",
    "dave",
    "eve",
    "frank",
    "grace",
    "henry",
  ];
  return randomElement(names);
}

function formatDate(date) {
  const pad = (n) => n.toString().padStart(2, "0");

  const year = date.getUTCFullYear();
  const month = pad(date.getUTCMonth() + 1);
  const day = pad(date.getUTCDate());
  const hours = pad(date.getUTCHours());
  const minutes = pad(date.getUTCMinutes());
  const seconds = pad(date.getUTCSeconds());
  const milliseconds = pad(date.getUTCMilliseconds()).padStart(3, "0");

  const offset = -date.getTimezoneOffset();
  const offsetSign = offset >= 0 ? "+" : "-";
  const offsetHours = pad(Math.abs(Math.floor(offset / 60)));
  const offsetMinutes = pad(Math.abs(offset % 60));

  return `${year}-${month}-${day}T${hours}:${minutes}:${seconds}.${milliseconds}${offsetSign}${offsetHours}:${offsetMinutes}`;
}

function formatStructuredData() {
  const template = randomElement(STRUCTURED_DATA);
  if (template === "-") return "-";

  const now = new Date();
  return template
    .replace(/%d/g, randomInt(1000, 9999))
    .replace(/%s/g, () => randomIP())
    .replace(/%s/g, () => formatDate(now));
}

function formatMessage() {
  const template = randomElement(MESSAGE_TEMPLATES);
  return template
    .replace(/%s/g, () => {
      const types = [
        randomUsername,
        randomIP,
        () => `service_${randomInt(1, 100)}`,
      ];
      return randomElement(types)();
    })
    .replace(/%d/g, () => randomInt(1, 1000));
}

function generateLogEntry(timestamp) {
  const facility = randomElement(Object.values(FACILITIES));
  const severity = randomElement(Object.values(SEVERITIES));
  const pri = facility * 8 + severity;
  const host = randomElement(HOSTS);
  const app = randomElement(APPS);
  const procId = randomInt(1000, 9999);
  const msgId = `MSG${randomInt(10000, 99999)}`;
  const structData = formatStructuredData();
  const msg = formatMessage();

  return `<${pri}>1 ${formatDate(timestamp)} ${host} ${app} ${procId} ${msgId} ${structData} ${msg}\n`;
}

async function sendBatch(client, batch, attempt = 0) {
  const maxAttempts = 3;
  try {
    await client.write(batch.join(""));
    return true;
  } catch (error) {
    if (attempt < maxAttempts) {
      console.log(`Retrying batch (attempt ${attempt + 1}/${maxAttempts})...`);
      try {
        await client.connect();
        return await sendBatch(client, batch, attempt + 1);
      } catch (reconnectError) {
        console.error("Reconnection failed:", reconnectError.message);
        return false;
      }
    }
    console.error("Failed to send batch after max attempts");
    return false;
  }
}

async function main() {
  const client = new SyslogClient();
  let successful = true;

  try {
    await client.connect();
    console.log("Connected to syslog server");

    let logsSent = 0;
    const startTime = new Date();
    const timeSpan = 30 * 24 * 60 * 60 * 1000; // 30 days in milliseconds

    while (logsSent < CONFIG.NUM_LOGS && successful) {
      const batch = [];
      for (
        let i = 0;
        i < CONFIG.BATCH_SIZE && logsSent < CONFIG.NUM_LOGS;
        i++
      ) {
        const timestamp = new Date(startTime - Math.random() * timeSpan);
        batch.push(generateLogEntry(timestamp));
        logsSent++;
      }

      successful = await sendBatch(client, batch);
      if (successful) {
        console.log(`Progress: ${logsSent}/${CONFIG.NUM_LOGS} logs sent`);
        if (logsSent < CONFIG.NUM_LOGS) {
          await new Promise((resolve) =>
            setTimeout(resolve, CONFIG.DELAY_BETWEEN_BATCHES),
          );
        }
      }
    }

    if (successful) {
      console.log("All logs sent successfully");
    } else {
      console.error("Failed to send all logs");
      process.exit(1);
    }
  } catch (error) {
    console.error("Fatal error:", error.message);
    process.exit(1);
  } finally {
    client.cleanup();
  }
}

// Handle process termination
process.on("SIGINT", () => {
  console.log("\nGracefully shutting down...");
  process.exit(0);
});

main();
