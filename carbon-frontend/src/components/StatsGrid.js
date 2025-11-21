import React from 'react';
import './StatsGrid.css';

export const StatsGrid = ({ metrics = {}, loading }) => {
  const items = [
    {
      label: 'Dépenses analysées',
      value: metrics.total_spend
        ? `${metrics.total_spend.toFixed(2)} €`
        : '--',
    },
    {
      label: 'Empreinte totale',
      value: metrics.total_co2
        ? `${metrics.total_co2.toFixed(2)} tCO2e`
        : '--',
    },
    {
      label: 'Factures traitées',
      value: metrics.invoice_count ?? '--',
    },
    {
      label: 'Intensité carbone',
      value: metrics.avg_co2_per_euro
        ? `${metrics.avg_co2_per_euro.toFixed(3)} t/€`
        : '--',
    },
  ];

  return (
    <section className="ct-stats-grid">
      {items.map((stat) => (
        <div key={stat.label} className="ct-stat-card">
          <p className="ct-stat-label">{stat.label}</p>
          <p className="ct-stat-value">
            {loading ? <span className="ct-skeleton" /> : stat.value}
          </p>
        </div>
      ))}
    </section>
  );
};

