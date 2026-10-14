import React, { useEffect, useState } from 'react';
import { useNavigate, Navigate } from 'react-router-dom';
import { useAuth } from '../context/AuthContext';
import axios from 'axios';
import Input from 'shared/ui/Input';
import Button from 'shared/ui/Button';
import Alert from 'shared/ui/Alert';
import KaabaIcon from '../../../shared/src/components/icons/KaabaIcon';

const RATE_LIMIT_STORAGE_KEY = 'travel_login_retry_until';

const getStoredRetryUntil = () => {
  if (typeof window === 'undefined') return 0;

  const storedValue = Number(window.sessionStorage.getItem(RATE_LIMIT_STORAGE_KEY));
  return Number.isFinite(storedValue) && storedValue > Date.now() ? storedValue : 0;
};

const formatCountdown = (seconds) => {
  const minutes = Math.floor(seconds / 60);
  const remainingSeconds = seconds % 60;
  return `${String(minutes).padStart(2, '0')}:${String(remainingSeconds).padStart(2, '0')}`;
};

const LoginPage = () => {
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [rememberMe, setRememberMe] = useState(false);
  const [error, setError] = useState('');
  const [isLoading, setIsLoading] = useState(false);
  const [retryUntil, setRetryUntil] = useState(getStoredRetryUntil);
  const [currentTime, setCurrentTime] = useState(() => Date.now());
  
  const { user, login } = useAuth();
  const navigate = useNavigate();
  const retrySeconds = Math.max(0, Math.ceil((retryUntil - currentTime) / 1000));

  useEffect(() => {
    if (!retryUntil) return undefined;

    const updateCountdown = () => {
      const now = Date.now();
      setCurrentTime(now);
      if (now >= retryUntil) {
        window.sessionStorage.removeItem(RATE_LIMIT_STORAGE_KEY);
        setRetryUntil(0);
      }
    };

    updateCountdown();
    const intervalId = window.setInterval(updateCountdown, 1000);
    return () => window.clearInterval(intervalId);
  }, [retryUntil]);

  if (user) {
    return <Navigate to="/" replace />;
  }

  const handleSubmit = async (e) => {
    e.preventDefault();
    if (retrySeconds > 0) return;

    setError('');
    setIsLoading(true);

    try {
      const response = await axios.post(`${import.meta.env.VITE_API_BASE_URL}/api/auth/login`, {
        email,
        password
      });

      const { access_token, refresh_token, email: accountEmail, display_name: displayName } = response.data;
      const result = await login(access_token, refresh_token, accountEmail || email, displayName || '');
      
      if (result.success) {
        navigate('/');
      } else {
        setError(result.message);
      }
    } catch (err) {
      if (err.response?.status === 429) {
        const serverRetryAfter = Number(
          err.response?.data?.retry_after_seconds ?? err.response?.headers?.['retry-after'],
        );
        const retryAfterSeconds = Number.isFinite(serverRetryAfter) && serverRetryAfter > 0
          ? Math.ceil(serverRetryAfter)
          : 15 * 60;
        const nextRetryUntil = Date.now() + retryAfterSeconds * 1000;

        window.sessionStorage.setItem(RATE_LIMIT_STORAGE_KEY, String(nextRetryUntil));
        setCurrentTime(Date.now());
        setRetryUntil(nextRetryUntil);
        setError('');
      } else {
        setError(err.response?.data?.error || 'Terjadi kesalahan saat verifikasi login.');
      }
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <div className="min-h-screen bg-animated-gradient font-body text-neutral-900 antialiased selection:bg-primary-200 selection:text-neutral-900 lg:p-6">
      <div className="mx-auto grid min-h-screen max-w-[1280px] overflow-hidden bg-white lg:min-h-[calc(100vh-3rem)] lg:grid-cols-[1.1fr_1fr] lg:rounded-[28px] lg:shadow-card">

        {/* Panel video */}
        <div className="relative hidden bg-neutral-900 lg:block">
          <video
            className="absolute inset-0 h-full w-full object-cover"
            src={`${import.meta.env.BASE_URL}makkah.mp4`}
            autoPlay
            muted
            loop
            playsInline
            preload="auto"
            aria-hidden="true"
          />
          <div className="absolute inset-0 bg-gradient-to-t from-black/70 via-black/20 to-black/30" />
          <div className="relative flex h-full flex-col justify-between p-10 text-white">
            <div className="flex items-center gap-3">
              <div className="inline-flex h-10 w-10 items-center justify-center rounded-xl bg-primary-500 text-white">
                <KaabaIcon className="h-5 w-5" />
              </div>
              <span className="text-sm font-semibold tracking-wide">Azhan Corp</span>
            </div>
            <div className="max-w-md space-y-3">
              <h2 className="font-heading text-4xl font-bold leading-tight tracking-tight">
                Layani setiap perjalanan ibadah dengan lebih tenang.
              </h2>
              <p className="text-sm text-white/80">
                Paket, jadwal, dan keberangkatan jamaah dalam satu dashboard.
              </p>
            </div>
          </div>
        </div>

        {/* Panel form */}
        <div className="flex items-center justify-center px-6 py-12 sm:px-12">
          <div className="w-full max-w-[380px] space-y-8">

            {/* Header */}
            <div className="space-y-4">
              <div className="inline-flex h-12 w-12 items-center justify-center rounded-2xl bg-primary-500 text-white shadow-card lg:hidden">
                <KaabaIcon className="h-6 w-6" />
              </div>
              <h1 className="font-heading text-3xl font-bold tracking-tight text-neutral-900">
                Travel Dashboard
              </h1>
            </div>

            {/* Card Form */}
            <div className="space-y-5">
              {(error || retrySeconds > 0) && (
                <Alert variant="error" onClose={retrySeconds > 0 ? undefined : () => setError('')}>
                  {retrySeconds > 0
                    ? <>Terlalu banyak percobaan. Coba lagi dalam <strong>{formatCountdown(retrySeconds)}</strong>.</>
                    : error}
                </Alert>
              )}

              <form onSubmit={handleSubmit} className="space-y-4">
                <Input
                  label="Email"
                  type="email"
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  placeholder="admin@azhan.id"
                  required
                  autoFocus
                  className="!mb-0"
                />

                <Input
                  label="Kata Sandi"
                  type="password"
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  placeholder="••••••••"
                  required
                  className="!mb-0"
                />

                <div className="flex items-center justify-between text-xs pt-1">
                  <label className="flex items-center gap-2 text-neutral-600 cursor-pointer select-none">
                    <input 
                      type="checkbox" 
                      checked={rememberMe}
                      onChange={(e) => setRememberMe(e.target.checked)}
                      className="rounded border-neutral-300 accent-primary-500 focus:ring-primary-500/40 w-3.5 h-3.5 cursor-pointer" 
                    />
                    <span>Ingat saya</span>
                  </label>
                  <a 
                    href="https://wa.me/62812345678" 
                    target="_blank" 
                    rel="noopener noreferrer" 
                    className="text-neutral-500 hover:text-primary-600 transition-colors"
                  >
                    Lupa sandi?
                  </a>
                </div>

                <Button
                  type="submit"
                  variant="primary"
                  isLoading={isLoading}
                  disabled={isLoading || retrySeconds > 0}
                  className="w-full h-10 text-sm font-semibold justify-center mt-2"
                >
                  {retrySeconds > 0
                    ? `Coba lagi ${formatCountdown(retrySeconds)}`
                    : (isLoading ? 'Memproses...' : 'Masuk')}
                </Button>
              </form>
            </div>

            {/* Footer */}
            <p className="text-center text-xs text-neutral-400">
              &copy; {new Date().getFullYear()} Azhan Corp. All rights reserved.
            </p>

          </div>
        </div>
      </div>
    </div>
  );
};

export default LoginPage;
