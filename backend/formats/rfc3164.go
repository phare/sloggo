package formats

import (
    "errors"
    "regexp"
    "sloggo/models"
    "strconv"
    "strings"
    "time"
)

var (
    // Example: <34>Oct 11 22:14:15 mymachine su[123]: 'su root' failed
    rfc3164Regex = regexp.MustCompile(`^<(?P<pri>\d{1,3})>(?P<ts>[A-Z][a-z]{2}\s+\d{1,2}\s+\d{2}:\d{2}:\d{2})\s+(?P<host>\S+)\s+(?P<tag>[A-Za-z0-9_.\-\/]+)(?:\[(?P<pid>[^\]]+)\])?:\s*(?P<msg>.*)$`)
)

// ParseRFC3164ToLogEntry parses an RFC3164 (BSD) syslog line into a LogEntry
// Best-effort: fills missing fields with defaults compatible with the DB schema
func ParseRFC3164ToLogEntry(line string) (*models.LogEntry, error) {
    line = strings.TrimSpace(line)
    if line == "" {
        return nil, errors.New("empty message")
    }

    m := rfc3164Regex.FindStringSubmatch(line)
    if m == nil {
        return nil, errors.New("not rfc3164 format")
    }

    // Extract named groups
    groups := make(map[string]string)
    for i, name := range rfc3164Regex.SubexpNames() {
        if i != 0 && name != "" {
            groups[name] = m[i]
        }
    }

    // Priority -> facility/severity
    pri, err := strconv.Atoi(groups["pri"])
    if err != nil {
        return nil, err
    }
    facility := uint8(pri / 8)
    severity := uint8(pri % 8)

    // Timestamp (no year) e.g. "Oct 11 22:14:15"
    // Parse with current year in local time, then convert to time.Now() location
    now := time.Now()
    tsStr := groups["ts"]
    // time layout with optional leading space in day
    // Jan _2 15:04:05 handles single-digit days
    tsParsed, err := time.ParseInLocation("Jan _2 15:04:05", tsStr, now.Location())
    if err != nil {
        // Fallback: try RFC822-like without seconds? keep robust
        // If still failing, use current time
        tsParsed = now
    }
    // Inject current year (RFC3164 has no year)
    ts := time.Date(now.Year(), tsParsed.Month(), tsParsed.Day(), tsParsed.Hour(), tsParsed.Minute(), tsParsed.Second(), 0, now.Location())

    hostname := groups["host"]
    if hostname == "" {
        hostname = "-"
    }

    appName := groups["tag"]
    if appName == "" {
        appName = "-"
    }

    procID := groups["pid"]
    if procID == "" {
        procID = "-"
    }

    msg := groups["msg"]

    entry := &models.LogEntry{
        Severity:       severity,
        Facility:       facility,
        Version:        1,
        Timestamp:      ts,
        Hostname:       hostname,
        AppName:        appName,
        ProcID:         procID,
        MsgID:          "-",
        StructuredData: "-",
        Message:        msg,
    }

    return entry, nil
}
