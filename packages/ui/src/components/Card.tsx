import type { ReactNode } from 'react';
import { cn } from '../cn';

interface CardProps {
  children: ReactNode;
  className?: string;
  onClick?: () => void;
}

export function Card({ children, className, onClick }: CardProps) {
  return (
    <div
      onClick={onClick}
      className={cn(
        'rounded-xl border border-gray-100 bg-white p-5 shadow-sm',
        onClick && 'cursor-pointer transition-shadow hover:border-gray-200 hover:shadow-md',
        className,
      )}
    >
      {children}
    </div>
  );
}
