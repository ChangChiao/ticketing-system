"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { api } from "@/lib/api";
import { useAuthStore } from "@/stores/auth";
import Navbar from "@/components/Navbar";

export default function AuthPage() {
  const router = useRouter();
  const setAuth = useAuthStore((s) => s.setAuth);
  const [isLogin, setIsLogin] = useState(true);
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [name, setName] = useState("");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError("");
    setLoading(true);

    try {
      if (isLogin) {
        const { user, token } = await api.login({ email, password });
        setAuth(user, token);
      } else {
        const { user, token } = await api.register({ email, password, name });
        setAuth(user, token);
      }
      router.push("/");
    } catch (err) {
      setError(err instanceof Error ? err.message : "發生錯誤");
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="flex flex-col h-full">
      <Navbar />
      <main className="flex-1 flex items-center justify-center px-4">
        <div className="bg-[var(--bg-card)] rounded-[var(--radius)] p-8 w-full max-w-md">
          <h1 className="font-display text-2xl font-bold text-center mb-2">
            {isLogin ? "LOGIN" : "REGISTER"}
          </h1>
          <p className="font-mono text-[13px] text-[var(--text-secondary)] text-center mb-6">
            // {isLogin ? "sign_in_to_your_account" : "create_a_new_account"}
          </p>

          <form onSubmit={handleSubmit} className="flex flex-col gap-4">
            {!isLogin && (
              <div className="flex flex-col gap-1.5">
                <label className="font-mono text-[11px] text-[var(--text-secondary)]">
                  // name
                </label>
                <input
                  type="text"
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                  className="w-full bg-[var(--bg-elevated)] rounded-xl px-4 py-2.5 font-mono text-[13px] text-[var(--text-primary)] outline-none focus:ring-1 focus:ring-[var(--accent-orange)] transition"
                  required
                />
              </div>
            )}

            <div className="flex flex-col gap-1.5">
              <label className="font-mono text-[11px] text-[var(--text-secondary)]">
                // email
              </label>
              <input
                type="email"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                className="w-full bg-[var(--bg-elevated)] rounded-xl px-4 py-2.5 font-mono text-[13px] text-[var(--text-primary)] outline-none focus:ring-1 focus:ring-[var(--accent-orange)] transition"
                required
              />
            </div>

            <div className="flex flex-col gap-1.5">
              <label className="font-mono text-[11px] text-[var(--text-secondary)]">
                // password
              </label>
              <input
                type="password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                className="w-full bg-[var(--bg-elevated)] rounded-xl px-4 py-2.5 font-mono text-[13px] text-[var(--text-primary)] outline-none focus:ring-1 focus:ring-[var(--accent-orange)] transition"
                required
                minLength={8}
              />
            </div>

            {error && (
              <p className="font-mono text-xs text-[var(--status-red)]">{error}</p>
            )}

            <button
              type="submit"
              disabled={loading}
              className="w-full h-12 bg-[var(--accent-orange)] text-[var(--text-on-accent)] rounded-[var(--radius)] font-display text-base font-semibold hover:brightness-110 disabled:opacity-50 transition mt-2"
            >
              {loading ? "// processing..." : isLogin ? "登入" : "註冊"}
            </button>
          </form>

          <p className="text-center mt-5 font-mono text-xs text-[var(--text-secondary)]">
            {isLogin ? "// no_account?" : "// have_account?"}
            <button
              onClick={() => {
                setIsLogin(!isLogin);
                setError("");
              }}
              className="text-[var(--accent-orange)] font-semibold ml-1 hover:underline"
            >
              {isLogin ? "register" : "login"}
            </button>
          </p>
        </div>
      </main>
    </div>
  );
}
