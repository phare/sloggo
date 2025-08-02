import { ModeToggle } from "@/components/theme/toggle-mode";
import { Button } from "@/components/ui/button";
import NextLink from "next/link";
import { Link } from "@/components/custom/link";
import { Github } from "@/components/icons/github";
import { X } from "@/components/icons/x";
import { Bluesky } from "@/components/icons/bluesky";

export function SocialsFooter() {
  return (
    <div className="flex flex-col gap-1">
      <div className="flex justify-center items-center gap-2 p-1">
        <Button variant="ghost" size="sm" className="w-9 px-0" asChild>
          <NextLink href="https://github.com/phare/sloggo">
            <Github className="h-4 w-4" />
          </NextLink>
        </Button>
        <ModeToggle className="[&>svg]:h-4 [&>svg]:w-4" />
      </div>
      <p className="text-muted-foreground text-center text-sm">
        Powered by <Link href="https://openstatus.dev">OpenStatus</Link>
      </p>
    </div>
  );
}
