import React, { useState, useContext } from 'react';
import { useNavigate, Navigate } from 'react-router-dom';
import { AuthContext } from '../context/AuthContext';
import Input from '../components/ui/Input';
import Button from '../components/ui/Button';
import Alert from '../components/ui/Alert';

const Login = () => {
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
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
    <div className="min-h-screen flex bg-white font-body">
      {/* Left Column - Form */}
      <div className="w-full md:w-1/2 flex items-center justify-center p-8 sm:p-12 md:p-24 relative z-10">
        <div className="w-full max-w-sm space-y-8">
          
          <div className="space-y-2">
            <h2 className="text-3xl font-bold tracking-tight text-neutral-900 font-heading">
              Bismillah...
            </h2>
            <p className="text-neutral-500 text-sm">
              Please enter your details to access the master dashboard.
            </p>
          </div>

          <form onSubmit={handleSubmit} className="space-y-5">
            <Input
              label="Email address"
              type="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              placeholder="admin@azhan.id"
              required
            />

            <Input
              label="Password"
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              placeholder="••••••••"
              required
            />

            <div className="flex items-center justify-between text-sm">
              <label className="flex items-center gap-2 text-neutral-600 cursor-pointer">
                <input type="checkbox" className="rounded border-neutral-300 text-primary-500 focus:ring-primary-500 w-4 h-4" />
                <span>Remember me</span>
              </label>
              <a href="#" className="text-primary-600 hover:text-primary-700 font-medium transition-colors">
                Forgot password?
              </a>
            </div>

            {errorMsg && (
              <Alert variant="error" className="animate-in fade-in slide-in-from-top-1">
                {errorMsg}
              </Alert>
            )}

            <Button
              type="submit"
              variant="primary"
              isLoading={loading}
              className="w-full py-2.5 text-base mt-2"
            >
              Sign in
            </Button>
          </form>

          <p className="text-center text-sm text-neutral-500 pt-6">
            &copy; {new Date().getFullYear()} Azhan Technologies. All rights reserved.
          </p>
        </div>
      </div>

      {/* Right Column - Branding/Illustration */}
      <div 
        className="hidden md:flex w-1/2 relative overflow-hidden bg-neutral-900"
        style={{ backgroundImage: "url('/haram-bg.jpg')", backgroundSize: 'cover', backgroundPosition: 'bottom center' }}
      >
        {/* Black Transparent Overlay */}
        <div className="absolute inset-0 bg-black/70 backdrop-blur-[2px] z-0"></div>
        
        {/* Branding Content */}
        <div className="relative z-10 flex flex-col justify-center p-12 md:p-24 w-full text-white h-full">
          <div>
            <h2 className="text-4xl md:text-5xl font-bold font-heading leading-tight mb-6">
              Bersama Menuju <br/>
              <span className="text-transparent bg-clip-text bg-gradient-to-r from-primary-200 to-primary-400">
                Azhan Maju, Besar & Berkah
              </span>
            </h2>
            <p className="text-lg text-primary-100 max-w-md opacity-90 font-light leading-relaxed">
              Satu langkah hari ini, membawa kita lebih dekat pada masa depan yang lebih baik.
            </p>
          </div>
        </div>
      </div>
    </div>
  );
};

export default Login;
