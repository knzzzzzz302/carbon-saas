import React from 'react';
import './Sidebar.css';

const NAV_ITEMS = [
  { label: 'Tableau de bord', icon: '📊' },
  { label: 'Données', icon: '📁' },
  { label: 'Analyses', icon: '📈' },
  { label: 'Fournisseurs', icon: '🏭' },
  { label: 'Calculateur', icon: '🧮' },
  { label: 'Compliance', icon: '✅' },
  { label: 'Export', icon: '📤' },
  { label: 'Paramètres', icon: '⚙️' },
];

export const Sidebar = ({ onLogout }) => (
  <aside className="ct-sidebar">
    <div className="ct-brand">
      <span className="ct-logo">🌱</span>
      <div>
        <p className="ct-product">CarbonTracker</p>
        <p className="ct-tagline">Smart Climate Copilot</p>
      </div>
    </div>
    <nav>
      {NAV_ITEMS.map((item) => (
        <button key={item.label} className="ct-nav-item">
          <span>{item.icon}</span>
          <span>{item.label}</span>
        </button>
      ))}
    </nav>
    <button className="ct-logout" onClick={onLogout}>
      🚪 Déconnexion
    </button>
  </aside>
);

