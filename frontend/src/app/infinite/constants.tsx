"use client";

import { CopyToClipboardContainer } from "@/components/custom/copy-to-clipboard-container";
import { KVTabs } from "@/components/custom/kv-tabs";
import type {
  DataTableFilterField,
  Option,
  SheetField,
} from "@/components/data-table/types";
import { LEVELS } from "@/constants/levels";
import { getLevelColor, getLevelLabel } from "@/lib/request/level";
import { cn } from "@/lib/utils";
import { format } from "date-fns";
import type { SyslogMeta } from "./query-options";
import { type SyslogSchema } from "./schema";

// Syslog facility names
const SYSLOG_FACILITIES = [
  { label: "Kernel messages", value: 0 },
  { label: "User-level messages", value: 1 },
  { label: "Mail system", value: 2 },
  { label: "System daemons", value: 3 },
  { label: "Security/authorization messages", value: 4 },
  { label: "Messages generated internally by syslogd", value: 5 },
  { label: "Line printer subsystem", value: 6 },
  { label: "Network news subsystem", value: 7 },
  { label: "UUCP subsystem", value: 8 },
  { label: "Clock daemon", value: 9 },
  { label: "Security/authorization messages", value: 10 },
  { label: "FTP daemon", value: 11 },
  { label: "NTP subsystem", value: 12 },
  { label: "Log audit", value: 13 },
  { label: "Log alert", value: 14 },
  { label: "Clock daemon", value: 15 },
  { label: "Local use 0", value: 16 },
  { label: "Local use 1", value: 17 },
  { label: "Local use 2", value: 18 },
  { label: "Local use 3", value: 19 },
  { label: "Local use 4", value: 20 },
  { label: "Local use 5", value: 21 },
  { label: "Local use 6", value: 22 },
  { label: "Local use 7", value: 23 },
];

// Syslog severity levels
const SYSLOG_SEVERITIES = [
  { label: "Emergency", value: 0 },
  { label: "Alert", value: 1 },
  { label: "Critical", value: 2 },
  { label: "Error", value: 3 },
  { label: "Warning", value: 4 },
  { label: "Notice", value: 5 },
  { label: "Informational", value: 6 },
  { label: "Debug", value: 7 },
];

export const filterFields = [
  {
    label: "Time Range",
    value: "timestamp",
    type: "timerange",
    defaultOpen: true,
    commandDisabled: true,
  },
  {
    label: "Level",
    value: "level",
    type: "checkbox",
    defaultOpen: true,
    options: LEVELS.map((level) => ({ label: level, value: level })),
    component: (props: Option) => {
      const value = props.value as (typeof LEVELS)[number];
      return (
        <div className="flex w-full max-w-28 items-center justify-between gap-2 font-mono">
          <span className="capitalize text-foreground/70 group-hover:text-accent-foreground">
            {props.label}
          </span>
          <div className="flex items-center gap-2">
            <div
              className={cn(
                "h-2.5 w-2.5 rounded-[2px]",
                getLevelColor(value).bg,
              )}
            />
            <span className="text-xs text-muted-foreground/70">
              {getLevelLabel(value)}
            </span>
          </div>
        </div>
      );
    },
  },
  {
    label: "Hostname",
    value: "hostname",
    type: "input",
  },
  {
    label: "App Name",
    value: "appName",
    type: "input",
  },
  {
    label: "Proc ID",
    value: "procId",
    type: "input",
  },
  {
    label: "Msg ID",
    value: "msgId",
    type: "input",
  },
  {
    label: "Facility",
    value: "facility",
    type: "checkbox",
    options: SYSLOG_FACILITIES,
    component: (props: Option) => {
      return <span className="font-mono">{props.value}</span>;
    },
  },
  {
    label: "Severity",
    value: "severity",
    type: "checkbox",
    options: SYSLOG_SEVERITIES,
    component: (props: Option) => {
      return <span className="font-mono">{props.value}</span>;
    },
  },
  {
    label: "Priority",
    value: "priority",
    type: "slider",
    min: 0,
    max: 191,
  },
] satisfies DataTableFilterField<SyslogSchema>[];

export const sheetFields = [
  {
    id: "uuid",
    label: "Message ID",
    type: "readonly",
    skeletonClassName: "w-64",
  },
  {
    id: "timestamp",
    label: "Timestamp",
    type: "timerange",
    component: (props) => format(new Date(props.timestamp), "LLL dd, y HH:mm:ss"),
    skeletonClassName: "w-36",
  },
  {
    id: "priority",
    label: "Priority",
    type: "readonly",
    component: (props) => {
      const facility = props.facility;
      const severity = props.severity;
      const facilityNames = [
        "Kernel", "User", "Mail", "Daemon", "Auth", "Syslog", "LPR", "News",
        "UUCP", "Cron", "AuthPriv", "FTP", "NTP", "Audit", "Alert", "Clock",
        "Local0", "Local1", "Local2", "Local3", "Local4", "Local5", "Local6", "Local7"
      ];
      const severityNames = [
        "Emergency", "Alert", "Critical", "Error", "Warning", "Notice", "Info", "Debug"
      ];
      return (
        <div className="flex flex-col">
          <span className="font-mono">{props.priority}</span>
          <span className="text-sm text-muted-foreground">
            {facilityNames[facility]} ({facility}) • {severityNames[severity]} ({severity})
          </span>
        </div>
      );
    },
    skeletonClassName: "w-16",
  },
  {
    id: "version",
    label: "Version",
    type: "readonly",
    component: (props) => {
      return <span className="font-mono">{props.version}</span>;
    },
    skeletonClassName: "w-12",
  },
  {
    id: "hostname",
    label: "Hostname",
    type: "input",
    skeletonClassName: "w-24",
  },
  {
    id: "appName",
    label: "App Name",
    type: "input",
    skeletonClassName: "w-20",
  },
  {
    id: "procId",
    label: "Proc ID",
    type: "input",
    skeletonClassName: "w-16",
  },
  {
    id: "msgId",
    label: "Msg ID",
    type: "input",
    skeletonClassName: "w-16",
  },
  {
    id: "structuredData",
    label: "Structured Data",
    type: "readonly",
    condition: (props) => props.structuredData !== undefined && Object.keys(props.structuredData).length > 0,
    component: (props) => (
      <KVTabs data={props.structuredData || {}} className="-mt-[22px]" />
    ),
    className: "flex-col items-start w-full gap-1",
  },
  {
    id: "message",
    label: "Message",
    type: "readonly",
    component: (props) => (
      <CopyToClipboardContainer variant="destructive">
        {props.message}
      </CopyToClipboardContainer>
    ),
    className: "flex-col items-start w-full gap-1",
  },
] satisfies SheetField<SyslogSchema, SyslogMeta>[];
