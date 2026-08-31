import React, { useState } from 'react';
import { useNavigate, Navigate } from 'react-router-dom';
import { useAuth } from '../context/AuthContext';
import axios from 'axios';
import Input from '../components/ui/Input';
import Button from '../components/ui/Button';
import Alert from '../components/ui/Alert';
import KaabaIcon from '../../../shared/src/components/icons/KaabaIcon';

const LoginPage = () => {
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [rememberMe, setRememberMe] = useState(false);
  const [error, setError] = useState('');
  const [isLoading, setIsLoading] = useState(false);
  
  const { user, login } = useAuth();
  const navigate = useNavigate();

  if (user) {
    return <Navigate to="/" replace />;
  }

  const handleSubmit = async (e) => {
    e.preventDefault();
    setError('');
    setIsLoading(true);

    try {
      const response = await axios.post(`${import.meta.env.VITE_API_BASE_URL}/api/auth/login`, {
        email,
        password
      });

      const { access_token, refresh_token } = response.data;
      const result = login(access_token, refresh_token);
      
      if (result.success) {
        navigate('/');
      } else {
        setError(result.message);
      }
    } catch (err) {
      setError(err.response?.data?.error || 'Terjadi kesalahan saat verifikasi login.');
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <div className="min-h-screen flex flex-col items-center justify-center bg-[#F8FAFC] px-4 py-12 font-body text-neutral-900 antialiased selection:bg-primary-500 selection:text-neutral-900">
      <div className="w-full max-w-[380px] space-y-6">
        
        {/* Header / Logo */}
        <div className="text-center space-y-3">
          <div className="inline-flex items-center justify-center w-12 h-12 rounded-2xl bg-[#181C1F] border border-neutral-800 text-primary-500 shadow-sm">
            <KaabaIcon className="w-6 h-6" />
          </div>
          <div>
            <h1 className="text-2xl font-heading font-bold tracking-tight text-neutral-900">
              Azhan ERP
            </h1>
            <p className="text-xs text-neutral-500 mt-1 font-medium">
              Travel Dashboard
            </p>
          </div>
        </div>

        {/* Card Form */}
        <div className="bg-white p-7 rounded-2xl border border-neutral-200/80 shadow-sm space-y-5">
          {error && (
            <Alert variant="error" onClose={() => setError('')}>
              {error}
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
                  className="rounded border-neutral-300 text-primary-500 focus:ring-primary-500/40 w-3.5 h-3.5 cursor-pointer" 
                />
                <span>Ingat saya</span>
              </label>
              <a 
                href="https://wa.me/62812345678" 
                target="_blank" 
                rel="noopener noreferrer" 
                className="text-neutral-500 hover:text-neutral-900 transition-colors"
              >
                Lupa sandi?
              </a>
            </div>

            <Button
              type="submit"
              variant="primary"
              isLoading={isLoading}
              disabled={isLoading}
              className="w-full h-10 text-sm font-semibold justify-center shadow-xs mt-2"
            >
              {isLoading ? "Memproses..." : "Masuk"}
            </Button>
          </form>
        </div>

        {/* Footer */}
        <p className="text-center text-xs text-neutral-400">
          &copy; {new Date().getFullYear()} Azhan Grup. All rights reserved.
        </p>

      </div>
    </div>
  );
};

export default LoginPage;
