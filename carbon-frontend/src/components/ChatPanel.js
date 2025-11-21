import React from 'react';
import './ChatPanel.css';

export const ChatPanel = ({
  conversation,
  prompt,
  setPrompt,
  onSend,
  loading,
  statusReady,
  error,
}) => (
  <section className="ct-chat-panel">
    <header>
      <div>
        <p className="ct-chat-title">Assistant Écologie IA</p>
        <p className="ct-chat-subtitle">
          Analyse vos données, calcule l'empreinte carbone précise et propose
          des plans d'action.
        </p>
      </div>
      <span
        className={`ct-status-dot ${statusReady ? 'online' : 'offline'}`}
      />
    </header>
    <div className="ct-chat-messages">
      {conversation.length === 0 && (
        <p className="ct-chat-placeholder">
          Posez votre première question : “Quels sont mes fournisseurs les plus
          émetteurs ?”
        </p>
      )}
      {conversation.map((entry, idx) => (
        <div key={idx} className={`ct-chat-bubble ${entry.role}`}>
          <p>{entry.content}</p>
        </div>
      ))}
      {error && <div className="ct-error-banner">{error}</div>}
    </div>
    <form
      onSubmit={(e) => {
        e.preventDefault();
        onSend();
      }}
      className="ct-chat-input"
    >
      <input
        type="text"
        placeholder="Posez votre question..."
        value={prompt}
        onChange={(e) => setPrompt(e.target.value)}
        disabled={!statusReady || loading}
      />
      <button type="submit" disabled={!prompt || loading || !statusReady}>
        {loading ? 'Envoi...' : 'Envoyer'}
      </button>
    </form>
  </section>
);

