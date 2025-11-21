import React from 'react';
import './ScopeBreakdown.css';

export const ScopeBreakdown = ({ scopes = {} }) => {
  const entries = Object.entries(scopes);
  if (!entries.length) {
    return null;
  }

  const total = entries.reduce((acc, [, value]) => acc + value, 0);

  return (
    <section className="ct-panel">
      <header>
        <h3>Scopes GHG</h3>
      </header>
      <div className="ct-scope-bars">
        {entries.map(([scope, value]) => {
          const pct = total ? ((value / total) * 100).toFixed(1) : 0;
          return (
            <div key={scope} className="ct-scope-row">
              <span>{scope.toUpperCase()}</span>
              <div className="ct-scope-track">
                <div
                  className="ct-scope-fill"
                  style={{ width: `${pct}%` }}
                />
              </div>
              <span>{pct}%</span>
            </div>
          );
        })}
      </div>
    </section>
  );
};

