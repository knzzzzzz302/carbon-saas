import React from 'react';
import './InsightsPanel.css';

export const InsightsPanel = ({
  title,
  narrative,
  findings = [],
  recommendations = [],
  loading,
}) => (
  <section className="ct-panel">
    <header>
      <h3>{title}</h3>
      {loading && <span className="ct-pill">Calcul IA...</span>}
    </header>
    {narrative ? (
      <p className="ct-narrative">{narrative}</p>
    ) : (
      <p className="ct-placeholder">
        Les insights IA apparaîtront dès que des factures auront été analysées.
      </p>
    )}
    <div className="ct-list-columns">
      <div>
        <h4>Constats clés</h4>
        <ul>
          {findings.map((item, idx) => (
            <li key={idx}>{item}</li>
          ))}
        </ul>
      </div>
      <div>
        <h4>Actions recommandées</h4>
        <ul>
          {recommendations.map((item, idx) => (
            <li key={idx}>{item}</li>
          ))}
        </ul>
      </div>
    </div>
  </section>
);

