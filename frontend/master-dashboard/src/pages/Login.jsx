import React, { useState, useContext } from 'react';
import { useNavigate, Navigate } from 'react-router-dom';
import { AuthContext } from '../context/AuthContext';
import Input from 'shared/ui/Input';
import Button from 'shared/ui/Button';
import Alert from 'shared/ui/Alert';
import KaabaIcon from '../../../shared/src/components/icons/KaabaIcon';

const Login = () => {
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [rememberMe, setRememberMe] = useState(false);
  const [loading, setLoading] = useState(false);
  const [errorMsg, setErrorMsg] = useState('');
  
  const { login, accessToken } = useContext(AuthContext);
  const navigate = useNavigate();

  if (accessToken) {
    return <Navigate to="/" replace />;
  }

  const handleSubmit = async (e) => {
    e.preventDefault();
    setLoading(true);
    setErrorMsg('');
    try {
      await login(email, password);
      navigate('/');
    } catch (err) {
      setErrorMsg(err.message);
    } finally {
      setLoading(false);
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
                Satu tempat untuk seluruh operasional Azhan.
              </h2>
              <p className="text-sm text-white/80">
                Kelola jamaah, keuangan, dan tim dengan rapi dan terintegrasi.
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
                Office Dashboard
              </h1>
            </div>

            {/* Card Form */}
            <div className="space-y-5">
              {errorMsg && (
                <Alert variant="error">
                  {errorMsg}
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
                  isLoading={loading}
                  disabled={loading}
                  className="w-full h-10 text-sm font-semibold justify-center mt-2"
                >
                  {loading ? "Memproses..." : "Masuk"}
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

export default Login;
