export function normalizeAvatarURL(url?: string | null) {
  const value = (url || "").trim();
  if (!value || value.startsWith("file://")) {
    return "";
  }
  return value;
}

export function avatarInitial(name?: string | null) {
  const value = (name || "").trim();
  return (value ? value.slice(0, 1) : "U").toUpperCase();
}
