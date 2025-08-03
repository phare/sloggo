"use client";

import { useDataTable } from "@/components/data-table/data-table-provider";
import { Button } from "@/components/ui/button";
import { useHotKey } from "@/hooks/use-hot-key";
import { cn } from "@/lib/utils";
import type { FetchPreviousPageOptions } from "@tanstack/react-query";
import { CirclePause, CirclePlay } from "lucide-react";
import { useQueryStates } from "nuqs";
import * as React from "react";
import { searchParamsParser } from "../search-params";

const REFRESH_INTERVAL = 4_000;

interface LiveButtonProps {
  fetchPreviousPage?: (
    options?: FetchPreviousPageOptions | undefined,
  ) => Promise<unknown>;
}

export function LiveButton({ fetchPreviousPage }: LiveButtonProps) {
  const [{ live, date, sort }, setSearch] = useQueryStates(searchParamsParser);
  const { table } = useDataTable();
  useHotKey(handleClick, "j");

  React.useEffect(() => {
    let timeoutId: NodeJS.Timeout;

    async function fetchData() {
      if (live) {
        // For live mode, always fetch with updated cursor
        await fetchPreviousPage?.({
          cancelRefetch: false,
        });
        timeoutId = setTimeout(fetchData, REFRESH_INTERVAL);
      } else {
        clearTimeout(timeoutId);
      }
    }

    fetchData();

    return () => {
      clearTimeout(timeoutId);
    };
  }, [live, fetchPreviousPage]);

  // REMINDER: make sure to reset live when date is set
  // TODO: test properly
  React.useEffect(() => {
    if ((date || sort) && live) {
      setSearch((prev) => ({ ...prev, live: null }));
    }
  }, [date, sort]);

  function handleClick() {
    // When activating live mode, update cursor to current time
    const newLiveState = !live;
    setSearch((prev) => ({
      ...prev,
      live: newLiveState,
      // Reset date and sort when toggling live mode
      date: null,
      sort: null,
      // Set cursor to current time when activating live mode
      cursor: newLiveState ? new Date() : prev.cursor,
    }));
    table.getColumn("timestamp")?.setFilterValue(undefined);
    table.resetSorting();
  }

  return (
    <Button
      className={cn(live && "border-info text-info hover:text-info")}
      onClick={handleClick}
      variant="outline"
      size="sm"
    >
      {live ? (
        <CirclePause className="mr-2 h-4 w-4" />
      ) : (
        <CirclePlay className="mr-2 h-4 w-4" />
      )}
      Live
    </Button>
  );
}
