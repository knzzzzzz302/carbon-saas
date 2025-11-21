import React from 'react';
import './SuppliersTable.css';

export const SuppliersTable = ({ suppliers = [], loading }) => (
  <section className="ct-panel">
    <header>
      <h3>Fournisseurs critiques</h3>
      {loading && <span className="ct-pill">Analyse...</span>}
    </header>
    {suppliers.length === 0 ? (
      <p className="ct-placeholder">
        Dès qu'une facture est traitée, l'IA classe automatiquement vos
        fournisseurs par priorité carbone.
      </p>
    ) : (
      <div className="ct-table-wrapper">
        <table>
          <thead>
            <tr>
              <th>Fournisseur</th>
              <th>Dépenses (€)</th>
              <th>tCO2e</th>
              <th>Priorité</th>
              <th>Action IA</th>
            </tr>
          </thead>
          <tbody>
            {suppliers.map((supplier) => (
              <tr key={supplier.name}>
                <td>{supplier.name}</td>
                <td>{supplier.spend.toFixed(2)}</td>
                <td>{supplier.co2.toFixed(2)}</td>
                <td>
                  <span className={`ct-priority ${supplier.priority}`}>
                    {supplier.priority}
                  </span>
                </td>
                <td>{supplier.recommended_step}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    )}
  </section>
);

