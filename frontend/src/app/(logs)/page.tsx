"use client";

import * as React from "react";
import { Suspense } from "react";
import { Client } from "./client";

// Force static generation
export const dynamic = "force-static";

export default function LogsPage() {
  return (
    <Suspense fallback={<div>Loading...</div>}>
      <Client />
    </Suspense>
  );
}
