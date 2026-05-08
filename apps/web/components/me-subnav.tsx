"use client";

import { AUTH_CHANGED_EVENT, getUserProfile } from "@/lib/auth";
import Link from "next/link";
import { usePathname } from "next/navigation";
import { useEffect, useMemo, useState } from "react";

const baseItems = [
  { href: "/me", label: "资料" },
  { href: "/me/posts", label: "文章" },
  { href: "/me/favorites", label: "收藏" },
  { href: "/me/inbox", label: "通知" }
];

export function MeSubnav() {
  const pathname = usePathname();
  const [role, setRole] = useState("");
  useEffect(() => {
    const sync = () => setRole(getUserProfile()?.role || "");
    sync();
    window.addEventListener(AUTH_CHANGED_EVENT, sync);
    window.addEventListener("storage", sync);
    return () => {
      window.removeEventListener(AUTH_CHANGED_EVENT, sync);
      window.removeEventListener("storage", sync);
    };
  }, []);
  const items = useMemo(() => {
    if (role === "admin" || role === "super_admin") {
      return [...baseItems, { href: "/me/admin", label: "管理" }];
    }
    return baseItems;
  }, [role]);
  const activeIndex = Math.max(
    0,
    items.findIndex((item) => pathname === item.href || (item.href !== "/me" && pathname.startsWith(item.href)))
  );
  return (
    <nav className={`subnav seg-switch ${items.length === 5 ? "seg-switch-5" : "seg-switch-4"}`} data-active={activeIndex}>
      <span className="seg-switch-thumb" aria-hidden="true" />
      {items.map((item) => {
        const active = pathname === item.href || (item.href !== "/me" && pathname.startsWith(item.href));
        return (
          <Link key={item.href} href={item.href} className={`subnav-link${active ? " active" : ""}`}>
            {item.label}
          </Link>
        );
      })}
    </nav>
  );
}
