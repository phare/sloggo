import type { Metadata } from "next";
import "@/styles/globals.css";
import { ThemeProvider } from "@/components/theme/theme-provider";
import { Toaster } from "@/components/ui/sonner";
import { ReactQueryProvider } from "@/providers/react-query";
import { GeistMono } from "geist/font/mono";
import { GeistSans } from "geist/font/sans";
import { NuqsAdapter } from "nuqs/adapters/next/app";

export const metadata: Metadata = {
  title: "Sloggo",
  description:
    "Minimal RFC 5424 syslog collector and viewer. SQLite-backed. Runs as a single, resource-friendly process.",
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html
      lang="en"
      className={`${GeistSans.variable} ${GeistMono.variable}`}
      suppressHydrationWarning
    >
      {process.env.NODE_ENV === "development" ||
      process.env.NEXT_PUBLIC_REACT_SCAN === "true" ? (
        <head>
          <script
            src="https://unpkg.com/react-scan/dist/auto.global.js"
            async
          />
        </head>
      ) : null}
      <body className="min-h-screen bg-background antialiased">
        <ReactQueryProvider>
          <NuqsAdapter>
            <ThemeProvider attribute="class" defaultTheme="system" enableSystem>
              {children}
              <Toaster richColors />
            </ThemeProvider>
          </NuqsAdapter>
        </ReactQueryProvider>
      </body>
    </html>
  );
}
