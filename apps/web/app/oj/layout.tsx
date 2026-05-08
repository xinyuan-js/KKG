import { OJNav } from "@/components/oj-nav";
import { OJSubmitNotifier } from "@/components/oj-submit-notifier";
import { ReactNode } from "react";

export default function OJLayout({ children }: { children: ReactNode }) {
  return (
    <main className="page">
      <OJNav />
      {children}
      <OJSubmitNotifier />
    </main>
  );
}
