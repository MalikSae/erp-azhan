import React, { useState, useContext } from 'react';
import { useNavigate, Navigate } from 'react-router-dom';
import { AuthContext } from '../context/AuthContext';
import Input from '../components/ui/Input';
import Button from '../components/ui/Button';
import Alert from '../components/ui/Alert';
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
              Master Dashboard
            </p>
          </div>
        </div>

        {/* Card Form */}
        <div className="bg-white p-7 rounded-2xl border border-neutral-200/80 shadow-sm space-y-5">
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
              isLoading={loading}
              disabled={loading}
              className="w-full h-10 text-sm font-semibold justify-center shadow-xs mt-2"
            >
              {loading ? "Memproses..." : "Masuk"}
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

export default Login;
