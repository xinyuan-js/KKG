import { Nav } from "@/components/nav";
import { MeSubnav } from "@/components/me-subnav";

export default function MeLayout({ children }: { children: React.ReactNode }) {
  return (
    <main className="page">
      <Nav />
      <MeSubnav />
      {children}
    </main>
  );
}
