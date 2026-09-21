import type { Metadata } from "next";
import type { ReactNode } from "react";
import Nav from "@/components/Nav";
import "./globals.css";

export const metadata: Metadata = {
  title: "Start & Found",
  description: "A social platform where founders publish as themselves and as their projects.",
};

export default function RootLayout({ children }: { children: ReactNode }) {
  return (
    <html lang="en" className="h-full">
      <body className="flex min-h-full flex-col">
        <Nav />
        <main className="mx-auto w-full max-w-2xl flex-1 px-4 py-6">{children}</main>
        <footer className="mx-auto w-full max-w-2xl px-4 py-8 text-xs text-zinc-600">
          Posts are published by a user or by one of their projects.
        </footer>
      </body>
    </html>
  );
}
