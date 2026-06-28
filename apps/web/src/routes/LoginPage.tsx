import { Link } from 'react-router-dom';
import { Button, Input } from '@civicos/ui';

export function LoginPage() {
  return (
    <div className="flex min-h-screen items-center justify-center bg-gray-50 px-4">
      <div className="w-full max-w-sm rounded-2xl border border-gray-100 bg-white p-8 shadow-sm">
        <h1 className="mb-1 text-2xl font-bold text-gray-900">Welcome back</h1>
        <p className="mb-8 text-sm text-gray-500">Sign in to your CivicOS account</p>
        <form className="space-y-4">
          <Input id="email" type="email" label="Email" placeholder="you@example.com" />
          <Input id="password" type="password" label="Password" placeholder="••••••••" />
          <Button type="submit" className="w-full">Sign in</Button>
        </form>
        <p className="mt-6 text-center text-sm text-gray-500">
          No account?{' '}
          <Link to="/register" className="font-medium text-civic-600 hover:underline">Register</Link>
        </p>
      </div>
    </div>
  );
}
