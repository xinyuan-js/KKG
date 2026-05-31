"use client";

import { changeMyPassword, getCurrentUser, updateMyProfile, uploadImage } from "@/lib/api";
import { getAccessToken, getUserProfile, setUserProfile } from "@/lib/auth";
import { toZhError } from "@/lib/errors";
import { logoutAuthSession } from "@/lib/session-bridge";
import { Avatar } from "@/components/avatar";
import { useRouter } from "next/navigation";
import { type ChangeEvent, useEffect, useMemo, useRef, useState } from "react";

export default function MePage() {
  const router = useRouter();
  const [profile, setProfile] = useState<{
    id: number;
    username: string;
    email: string;
    avatar_url?: string;
    role?: string;
  } | null>(null);
  const [profileError, setProfileError] = useState("");
  const [profileSuccess, setProfileSuccess] = useState("");
  const [securityError, setSecurityError] = useState("");
  const [securitySuccess, setSecuritySuccess] = useState("");

  const [username, setUsername] = useState("");
  const [email, setEmail] = useState("");
  const [avatarURL, setAvatarURL] = useState("");
  const [savingProfile, setSavingProfile] = useState(false);
  const [uploadingAvatar, setUploadingAvatar] = useState(false);
  const avatarInputRef = useRef<HTMLInputElement | null>(null);

  const [oldPassword, setOldPassword] = useState("");
  const [newPassword, setNewPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [changingPassword, setChangingPassword] = useState(false);
  const [initialUsername, setInitialUsername] = useState("");
  const [initialEmail, setInitialEmail] = useState("");
  const [initialAvatarURL, setInitialAvatarURL] = useState("");

  useEffect(() => {
    const p = getUserProfile();
    if (p) {
      setProfile(p);
      setUsername(p.username || "");
      setEmail(p.email || "");
      setAvatarURL(p.avatar_url || "");
      setInitialUsername(p.username || "");
      setInitialEmail(p.email || "");
      setInitialAvatarURL(p.avatar_url || "");
    }
    void loadProfile();
  }, [router]);

  async function loadProfile() {
    try {
      const p = await getCurrentUser();
      setProfile(p);
      setUsername(p.username || "");
      setEmail(p.email || "");
      setAvatarURL(p.avatar_url || "");
      setInitialUsername(p.username || "");
      setInitialEmail(p.email || "");
      setInitialAvatarURL(p.avatar_url || "");
      setUserProfile(p);
    } catch {
      if (!getUserProfile()) {
        router.replace("/login?redirect=/me");
      }
    }
  }

  async function logout() {
    if (!window.confirm("确认退出当前登录吗？")) return;
    await logoutAuthSession();
    router.push("/login");
  }

  async function onSaveProfile() {
    setProfileError("");
    setProfileSuccess("");
    setSavingProfile(true);
    try {
      const token = getAccessToken();
      if (!token) {
        router.replace("/login?redirect=/me");
        return;
      }
      if (username.trim().length < 2) {
        throw new Error("用户名至少 2 个字符");
      }
      if (!email.includes("@")) {
        throw new Error("邮箱格式不正确");
      }
      const p = await updateMyProfile(token, { username, email, avatar_url: avatarURL });
      setProfile(p);
      setUserProfile(p);
      setAvatarURL(p.avatar_url || "");
      setInitialUsername(p.username || "");
      setInitialEmail(p.email || "");
      setInitialAvatarURL(p.avatar_url || "");
      setProfileSuccess("个人资料已更新");
    } catch (err) {
      setProfileError(toZhError(err, "更新资料失败"));
    } finally {
      setSavingProfile(false);
    }
  }

  async function onChangePassword() {
    setSecurityError("");
    setSecuritySuccess("");
    setChangingPassword(true);
    try {
      const token = getAccessToken();
      if (!token) {
        router.replace("/login?redirect=/me");
        return;
      }
      if (oldPassword.trim().length === 0) {
        throw new Error("请输入当前密码");
      }
      if (newPassword.length < 8) {
        throw new Error("新密码至少 8 位");
      }
      if (newPassword === oldPassword) {
        throw new Error("新密码不能与当前密码相同");
      }
      if (newPassword !== confirmPassword) {
        throw new Error("两次输入的新密码不一致");
      }
      await changeMyPassword(token, { old_password: oldPassword, new_password: newPassword });
      setOldPassword("");
      setNewPassword("");
      setConfirmPassword("");
      setSecuritySuccess("密码修改成功");
    } catch (err) {
      setSecurityError(toZhError(err, "修改密码失败"));
    } finally {
      setChangingPassword(false);
    }
  }

  const profileDirty = useMemo(
    () =>
      username.trim() !== initialUsername.trim() ||
      email.trim() !== initialEmail.trim() ||
      avatarURL.trim() !== initialAvatarURL.trim(),
    [username, email, avatarURL, initialUsername, initialEmail, initialAvatarURL]
  );

  const passwordRules = useMemo(
    () => ({
      length: newPassword.length >= 8,
      mixedCase: /[a-z]/.test(newPassword) && /[A-Z]/.test(newPassword),
      number: /\d/.test(newPassword),
      symbol: /[^A-Za-z0-9]/.test(newPassword),
      confirmed: newPassword.length > 0 && newPassword === confirmPassword
    }),
    [newPassword, confirmPassword]
  );

  const canSubmitPassword =
    oldPassword.trim().length > 0 &&
    passwordRules.length &&
    passwordRules.confirmed &&
    newPassword !== oldPassword;

  async function onPickAvatarFile(e: ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0];
    if (!file) return;
    setProfileError("");
    setProfileSuccess("");
    setUploadingAvatar(true);
    try {
      const token = getAccessToken();
      if (!token) {
        router.replace("/login?redirect=/me");
        return;
      }
      if (!file.type.startsWith("image/")) {
        throw new Error("仅支持图片文件");
      }
      if (file.size > 5 * 1024 * 1024) {
        throw new Error("图片大小不能超过 5MB");
      }
      const url = await uploadImage(token, file);
      setAvatarURL(url);
      setProfileSuccess("头像已上传，点击“保存资料”后生效");
    } catch (err) {
      setProfileError(toZhError(err, "头像上传失败"));
    } finally {
      setUploadingAvatar(false);
      if (avatarInputRef.current) {
        avatarInputRef.current.value = "";
      }
    }
  }

  return (
    <>
      <header className="page-header">
        <h1 className="page-title">个人信息</h1>
        <p className="tip">管理你的公开资料与账号安全。</p>
      </header>
      {!profile ? (
        <p className="tip">加载中...</p>
      ) : (
        <div className="me-grid">
          <section className="card section-gap me-panel">
            <h2 className="me-panel-title">账号资料</h2>
            <p className="meta">
              ID: {profile.id} | 角色: {profile.role || "user"}
            </p>
            <div className="profile-avatar-row">
              <Avatar
                className="profile-avatar"
                fallbackClassName="profile-avatar profile-avatar-fallback"
                src={avatarURL}
                name={username || profile.username}
                loading="eager"
              />
              <div className="section-gap">
                <input
                  ref={avatarInputRef}
                  type="file"
                  accept="image/*"
                  onChange={onPickAvatarFile}
                  disabled={uploadingAvatar}
                />
                <p className="tip me-zero-gap">
                  {uploadingAvatar ? "头像上传中..." : "选择图片后上传，完成后再点“保存资料”"}
                </p>
              </div>
            </div>
            <input placeholder="用户名" value={username} onChange={(e) => setUsername(e.target.value)} />
            <input placeholder="邮箱" value={email} onChange={(e) => setEmail(e.target.value)} />
            <div className="toolbar-row">
              <button type="button" onClick={onSaveProfile} disabled={savingProfile}>
                {savingProfile ? "保存中..." : profileDirty ? "保存资料" : "暂无修改"}
              </button>
              <button type="button" className="ghost" onClick={logout}>
                退出登录
              </button>
            </div>
            {profileError ? <p className="error">{profileError}</p> : null}
            {profileSuccess ? <p className="success">{profileSuccess}</p> : null}
          </section>

          <section className="card section-gap me-panel">
            <h2 className="me-panel-title">安全设置</h2>
            <input
              type="password"
              placeholder="当前密码"
              value={oldPassword}
              onChange={(e) => setOldPassword(e.target.value)}
            />
            <input
              type="password"
              placeholder="新密码（至少8位）"
              value={newPassword}
              onChange={(e) => setNewPassword(e.target.value)}
            />
            <input
              type="password"
              placeholder="确认新密码"
              value={confirmPassword}
              onChange={(e) => setConfirmPassword(e.target.value)}
            />
            <p className="tip me-zero-gap">建议使用大小写字母 + 数字 + 符号组合。</p>
            <ul className="security-rules">
              <li className={passwordRules.length ? "ok" : ""}>至少 8 位</li>
              <li className={passwordRules.mixedCase ? "ok" : ""}>包含大小写字母</li>
              <li className={passwordRules.number ? "ok" : ""}>包含数字</li>
              <li className={passwordRules.symbol ? "ok" : ""}>包含符号</li>
              <li className={passwordRules.confirmed ? "ok" : ""}>两次输入一致</li>
            </ul>
            <button type="button" onClick={onChangePassword} disabled={changingPassword || !canSubmitPassword}>
              {changingPassword ? "提交中..." : "修改密码"}
            </button>
            {securityError ? <p className="error">{securityError}</p> : null}
            {securitySuccess ? <p className="success">{securitySuccess}</p> : null}
          </section>
        </div>
      )}
    </>
  );
}
