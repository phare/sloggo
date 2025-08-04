"use client";

import { TextWithTooltip } from "@/components/custom/text-with-tooltip";
import { DataTableColumnHeader } from "@/components/data-table/data-table-column-header";
import { DataTableColumnLevelIndicator } from "@/components/data-table/data-table-column/data-table-column-level-indicator";
import { cn } from "@/lib/utils";
import type { ColumnDef } from "@tanstack/react-table";
import { HoverCardTimestamp } from "./_components/hover-card-timestamp";
import type { ColumnSchema } from "./schema";

// Facility names for display
const FACILITY_NAMES = [
  "Kernel",
  "User",
  "Mail",
  "Daemon",
  "Auth",
  "Syslog",
  "LPR",
  "News",
  "UUCP",
  "Cron",
  "AuthPriv",
  "FTP",
  "NTP",
  "Audit",
  "Alert",
  "Clock",
  "Local0",
  "Local1",
  "Local2",
  "Local3",
  "Local4",
  "Local5",
  "Local6",
  "Local7",
];

// Severity names for display
const SEVERITY_NAMES = [
  "Emergency",
  "Alert",
  "Critical",
  "Error",
  "Warning",
  "Notice",
  "Info",
  "Debug",
];

export const columns: ColumnDef<ColumnSchema>[] = [
  {
    accessorKey: "level",
    header: "",
    cell: ({ row }) => {
      const level = row.getValue<ColumnSchema["level"]>("level");
      return <DataTableColumnLevelIndicator value={level} />;
    },
    enableHiding: false,
    enableResizing: false,
    filterFn: "arrSome",
    size: 27,
    minSize: 27,
    maxSize: 27,
    meta: {
      headerClassName:
        "w-[--header-level-size] max-w-[--header-level-size] min-w-[--header-level-size]",
      cellClassName:
        "w-[--col-level-size] max-w-[--col-level-size] min-w-[--col-level-size]",
    },
  },
  {
    accessorKey: "id",
    header: "ID",
    cell: ({ row }) => {
      const id = row.getValue<ColumnSchema["id"]>("id");
      return <span className="font-mono">{id}</span>;
    },
    enableResizing: false,
    size: 70,
    minSize: 70,
    meta: {
      headerClassName:
        "w-[--header-id-size] max-w-[--header-id-size] min-w-[--header-id-size]",
      cellClassName:
        "font-mono w-[--col-id-size] max-w-[--col-id-size] min-w-[--col-id-size]",
    },
  },
  {
    accessorKey: "timestamp",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="Timestamp" />
    ),
    cell: ({ row }) => {
      const date = new Date(
        row.getValue<ColumnSchema["timestamp"]>("timestamp"),
      );
      return <HoverCardTimestamp date={date} />;
    },
    filterFn: "inDateRange",
    enableResizing: false,
    size: 200,
    minSize: 200,
    meta: {
      headerClassName:
        "w-[--header-timestamp-size] max-w-[--header-timestamp-size] min-w-[--header-timestamp-size]",
      cellClassName:
        "font-mono w-[--col-timestamp-size] max-w-[--col-timestamp-size] min-w-[--col-timestamp-size]",
    },
  },
  // Using SQLite rowid instead of UUID
  {
    accessorKey: "severity",
    header: "Severity",
    cell: ({ row }) => {
      const severity = row.getValue<ColumnSchema["severity"]>("severity");
      return (
        <div className="flex items-baseline gap-2">
          <span className="font-mono text-sm"> {SEVERITY_NAMES[severity]}</span>
          <span className="text-xs text-muted-foreground">{severity}</span>
        </div>
      );
    },
    filterFn: "arrIncludesSome",
    enableResizing: false,
    size: 100,
    minSize: 100,
    meta: {
      headerClassName:
        "w-[--header-severity-size] max-w-[--header-severity-size] min-w-[--header-severity-size]",
      cellClassName:
        "font-mono w-[--col-severity-size] max-w-[--col-severity-size] min-w-[--col-severity-size]",
    },
  },
  {
    accessorKey: "priority",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="Priority" />
    ),
    cell: ({ row }) => {
      const priority = row.getValue<ColumnSchema["priority"]>("priority");
      return <span className="font-mono">{priority}</span>;
    },
    filterFn: "inNumberRange",
    enableResizing: false,
    size: 80,
    minSize: 80,
    meta: {
      headerClassName:
        "w-[--header-priority-size] max-w-[--header-priority-size] min-w-[--header-priority-size]",
      cellClassName:
        "font-mono w-[--col-priority-size] max-w-[--col-priority-size] min-w-[--col-priority-size]",
    },
  },
  {
    accessorKey: "message",
    header: "Message",
    cell: ({ row }) => {
      const value = row.getValue<ColumnSchema["message"]>("message");
      return <TextWithTooltip text={value} />;
    },
    size: 300,
    minSize: 200,
    meta: {
      cellClassName:
        "font-mono w-[--col-message-size] max-w-[--col-message-size]",
      headerClassName:
        "min-w-[--header-message-size] w-[--header-message-size]",
    },
  },
  {
    accessorKey: "hostname",
    header: "Hostname",
    cell: ({ row }) => {
      const value = row.getValue<ColumnSchema["hostname"]>("hostname");
      return <TextWithTooltip text={value} />;
    },
    size: 125,
    minSize: 125,
    meta: {
      cellClassName:
        "font-mono w-[--col-hostname-size] max-w-[--col-hostname-size]",
      headerClassName:
        "min-w-[--header-hostname-size] w-[--header-hostname-size]",
    },
  },
  {
    accessorKey: "appName",
    header: "App Name",
    cell: ({ row }) => {
      const value = row.getValue<ColumnSchema["appName"]>("appName");
      return <TextWithTooltip text={value} />;
    },
    size: 100,
    minSize: 100,
    meta: {
      cellClassName:
        "font-mono w-[--col-appname-size] max-w-[--col-appname-size]",
      headerClassName:
        "min-w-[--header-appname-size] w-[--header-appname-size]",
    },
  },
  {
    accessorKey: "procId",
    header: "Proc ID",
    cell: ({ row }) => {
      const value = row.getValue<ColumnSchema["procId"]>("procId");
      return <span className="font-mono">{value}</span>;
    },
    size: 80,
    minSize: 80,
    meta: {
      cellClassName:
        "font-mono w-[--col-procid-size] max-w-[--col-procid-size]",
      headerClassName: "min-w-[--header-procid-size] w-[--header-procid-size]",
    },
  },
  {
    accessorKey: "msgId",
    header: "Msg ID",
    cell: ({ row }) => {
      const value = row.getValue<ColumnSchema["msgId"]>("msgId");
      return <span className="font-mono">{value}</span>;
    },
    size: 80,
    minSize: 80,
    meta: {
      cellClassName: "font-mono w-[--col-msgid-size] max-w-[--col-msgid-size]",
      headerClassName: "min-w-[--header-msgid-size] w-[--header-msgid-size]",
    },
  },
  {
    accessorKey: "facility",
    header: "Facility",
    cell: ({ row }) => {
      const facility = row.getValue<ColumnSchema["facility"]>("facility");
      return (
        <div className="flex items-baseline gap-2">
          <span className="font-mono text-sm">{FACILITY_NAMES[facility]}</span>
          <span className="text-xs text-muted-foreground">{facility}</span>
        </div>
      );
    },
    filterFn: "arrIncludesSome",
    enableResizing: false,
    size: 100,
    minSize: 100,
    meta: {
      headerClassName:
        "w-[--header-facility-size] max-w-[--header-facility-size] min-w-[--header-facility-size]",
      cellClassName:
        "font-mono w-[--col-facility-size] max-w-[--col-facility-size] min-w-[--col-facility-size]",
    },
  },
  {
    accessorKey: "structuredData",
    header: "Structured Data",
    cell: ({ row }) => {
      const value =
        row.getValue<ColumnSchema["structuredData"]>("structuredData");
      if (!value || Object.keys(value).length === 0) {
        return <span className="text-muted-foreground">-</span>;
      }
      return (
        <TextWithTooltip
          text={Object.entries(value)
            .map(
              ([sdId, kvPairs]) =>
                `${sdId}:{${Object.entries(kvPairs || {})
                  .map(([k, v]) => `${k}=${v}`)
                  .join(", ")}}`,
            )
            .join(", ")}
        />
      );
    },
    size: 150,
    minSize: 150,
    meta: {
      cellClassName:
        "font-mono w-[--col-structureddata-size] max-w-[--col-structureddata-size]",
      headerClassName:
        "min-w-[--header-structureddata-size] w-[--header-structureddata-size]",
    },
  },
];
