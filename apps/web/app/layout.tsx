import type { Metadata } from "next";
import { cookies } from "next/headers";
import "./globals.css";
import "@toast-ui/editor/dist/toastui-editor.css";
import { BackToTop } from "@/components/back-to-top";
import { TopNotice } from "@/components/top-notice";

export const metadata: Metadata = {
  title: "KKG",
  description: "KKG Blog System"
};

export default async function RootLayout({ children }: { children: React.ReactNode }) {
  const cookieStore = await cookies();
  const themeCookie = cookieStore.get("theme_mode")?.value;
  const initialTheme = themeCookie === "dark" || themeCookie === "light" ? themeCookie : undefined;

  return (
    <html lang="zh-CN" data-theme={initialTheme} suppressHydrationWarning>
      <head>
        <script
          dangerouslySetInnerHTML={{
            __html:
              "(function(){try{var s=localStorage.getItem('theme_mode');var d=window.matchMedia&&window.matchMedia('(prefers-color-scheme: dark)').matches;var t=(s==='light'||s==='dark')?s:(d?'dark':'light');document.documentElement.setAttribute('data-theme',t);}catch(e){}})();"
          }}
        />
      </head>
      <body>
        {children}
        <TopNotice />
        <BackToTop />
      </body>
    </html>
  );
}
