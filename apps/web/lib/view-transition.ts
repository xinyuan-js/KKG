import type { AppRouterInstance } from "next/dist/shared/lib/app-router-context.shared-runtime";

type SwipeDir = "to-oj" | "to-blog";

type DocumentWithVT = Document & {
  startViewTransition?: (update: () => void | Promise<void>) => {
    finished: Promise<void>;
  };
};

export function swipeNavigate(router: AppRouterInstance, href: string, dir: SwipeDir) {
  if (typeof window === "undefined" || typeof document === "undefined") {
    router.push(href);
    return;
  }

  const doc = document as DocumentWithVT;
  document.documentElement.setAttribute("data-route-swipe", dir);

  const clear = () => {
    window.setTimeout(() => {
      document.documentElement.removeAttribute("data-route-swipe");
    }, 460);
  };

  if (!doc.startViewTransition) {
    router.push(href);
    clear();
    return;
  }

  const transition = doc.startViewTransition(() => {
    router.push(href);
  });
  transition.finished.finally(clear);
}

