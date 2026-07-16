'use client';

import { useCallback, useState } from 'react';

interface StarRatingProps {
  value: number;
  onChange?: (value: number) => void;
  readOnly?: boolean;
  size?: 'sm' | 'md' | 'lg';
}

const SIZE_MAP = { sm: 'h-4 w-4', md: 'h-5 w-5', lg: 'h-6 w-6' } as const;
const STAR_D =
  'M11.48 3.499a.562.562 0 011.04 0l2.125 5.111a.563.563 0 00.475.345l5.518.442c.499.04.701.663.321.988l-4.204 3.602a.563.563 0 00-.182.557l1.285 5.385a.562.562 0 01-.84.61l-4.725-2.885a.563.563 0 00-.586 0L6.982 20.54a.562.562 0 01-.84-.61l1.285-5.386a.562.562 0 00-.182-.557l-4.204-3.602a.563.563 0 01.321-.988l5.518-.442a.563.563 0 00.475-.345L11.48 3.5z';

function StarSvg({ fill, sizeClass, starIdx }: { fill: 'full' | 'half' | 'empty'; sizeClass: string; starIdx: number }) {
  const colorClass = fill === 'empty' ? 'text-stone-300 dark:text-stone-600' : 'text-amber-500 dark:text-amber-400';
  return (
    <svg viewBox="0 0 24 24" aria-hidden="true" className={`${sizeClass} ${colorClass} transition-colors`}>
      {fill === 'half' ? (
        <>
          <defs>
            <clipPath id={`half-${starIdx}`}>
              <rect x="0" y="0" width="12" height="24" />
            </clipPath>
          </defs>
          <path d={STAR_D} fill="currentColor" clipPath={`url(#half-${starIdx})`} />
          <path d={STAR_D} fill="none" stroke="currentColor" strokeWidth={1.5} />
        </>
      ) : (
        <path d={STAR_D} fill={fill === 'full' ? 'currentColor' : 'none'}
          stroke="currentColor" strokeWidth={1.5} />
      )}
    </svg>
  );
}

function getFill(displayValue: number, star: number): 'full' | 'half' | 'empty' {
  if (displayValue >= star) return 'full';
  if (displayValue >= star - 0.5) return 'half';
  return 'empty';
}

export default function StarRating({
  value,
  onChange,
  readOnly = false,
  size = 'md',
}: StarRatingProps) {
  const [hoverValue, setHoverValue] = useState<number | null>(null);
  const displayValue = hoverValue ?? value;
  const sizeClass = SIZE_MAP[size];

  const handleMouseLeave = useCallback(() => {
    if (!readOnly) setHoverValue(null);
  }, [readOnly]);

  if (readOnly) {
    return (
      <div className="inline-flex items-center gap-0.5" role="img"
        aria-label={`${value} out of 5 stars`}>
        {[1, 2, 3, 4, 5].map((star) => (
          <StarSvg key={star} fill={getFill(displayValue, star)} sizeClass={sizeClass} starIdx={star} />
        ))}
      </div>
    );
  }

  return (
    <div className="inline-flex items-center gap-0.5" role="radiogroup"
      aria-label="Rating" onMouseLeave={handleMouseLeave}>
      {[1, 2, 3, 4, 5].map((star) => (
        <button key={star} type="button"
          aria-label={`${star} star${star > 1 ? 's' : ''}`}
          className="cursor-pointer border-0 bg-transparent p-0"
          onMouseMove={(e) => {
            const rect = e.currentTarget.getBoundingClientRect();
            const isHalf = e.clientX - rect.left < rect.width / 2;
            setHoverValue(isHalf ? star - 0.5 : star);
          }}
          onClick={(e) => {
            if (!onChange) return;
            const rect = e.currentTarget.getBoundingClientRect();
            const isHalf = e.clientX - rect.left < rect.width / 2;
            onChange(isHalf ? star - 0.5 : star);
          }}>
          <StarSvg fill={getFill(displayValue, star)} sizeClass={sizeClass} starIdx={star} />
        </button>
      ))}
      <span className="ml-1.5 text-sm font-medium text-stone-600 dark:text-stone-400">
        {displayValue.toFixed(1)}
      </span>
    </div>
  );
}
