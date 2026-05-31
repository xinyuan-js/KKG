"use client";

import { useEffect, useState } from "react";
import type { CSSProperties } from "react";
import { avatarInitial, normalizeAvatarURL } from "@/lib/avatar";

type AvatarProps = {
  src?: string | null;
  name?: string | null;
  className: string;
  fallbackClassName?: string;
  alt?: string;
  style?: CSSProperties;
  loading?: "eager" | "lazy";
};

export function Avatar({
  src,
  name,
  className,
  fallbackClassName,
  alt,
  style,
  loading = "lazy"
}: AvatarProps) {
  const normalized = normalizeAvatarURL(src);
  const [broken, setBroken] = useState(false);

  useEffect(() => {
    setBroken(false);
  }, [normalized]);

  if (normalized && !broken) {
    return (
      <img
        className={className}
        src={normalized}
        alt={alt || name || "avatar"}
        style={style}
        loading={loading}
        onError={() => setBroken(true)}
      />
    );
  }

  return (
    <span className={fallbackClassName || `${className} ${className}-fallback`} style={style}>
      {avatarInitial(name)}
    </span>
  );
}
