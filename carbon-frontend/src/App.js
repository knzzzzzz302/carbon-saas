import React, { useCallback, useEffect, useMemo, useState } from 'react';
import './App.css';
import { Sidebar } from './components/Sidebar';
import { StatsGrid } from './components/StatsGrid';
import { InsightsPanel } from './components/InsightsPanel';
import { ChatPanel } from './components/ChatPanel';
import { SuppliersTable } from './components/SuppliersTable';
import { ScopeBreakdown } from './components/ScopeBreakdown';
import {
  aiClient,
  fetchAIStatus,
  fetchAnalytics,
  fetchSuppliers,
  sendChatPrompt,
} from './services/ai';

function App() {
  const [token, setToken] = useState(() => aiClient.getToken());
  const [statusReady, setStatusReady] = useState(false);
  const [analytics, setAnalytics] = useState(null);
  const [suppliers, setSuppliers] = useState([]);
  const [loadingAnalytics, setLoadingAnalytics] = useState(false);
  const [loadingSuppliers, setLoadingSuppliers] = useState(false);
  const [chatLoading, setChatLoading] = useState(false);
  const [prompt, setPrompt] = useState('');
  const [conversation, setConversation] = useState([]);
  const [chatError, setChatError] = useState('');
  const [globalError, setGlobalError] = useState('');

  const hasToken = useMemo(() => Boolean(token && token.trim().length > 10), [token]);

  const refreshStatus = useCallback(async () => {
    try {
      const status = await fetchAIStatus(token);
      setStatusReady(Boolean(status.ready));
    } catch (error) {
      setStatusReady(false);
    }
  }, [token]);

  const loadAnalytics = useCallback(async () => {
    if (!hasToken) return;
    setLoadingAnalytics(true);
    setGlobalError('');
    try {
      const data = await fetchAnalytics(token);
      setAnalytics(data);
    } catch (error) {
      setGlobalError(error.message);
    } finally {
      setLoadingAnalytics(false);
    }
  }, [hasToken, token]);

  const loadSuppliers = useCallback(async () => {
    if (!hasToken) return;
    setLoadingSuppliers(true);
    try {
      const data = await fetchSuppliers(token);
      setSuppliers(data.suppliers || data.Suppliers || []);
    } catch (error) {
      setGlobalError(error.message);
    } finally {
      setLoadingSuppliers(false);
    }
  }, [hasToken, token]);

  useEffect(() => {
    if (!hasToken) {
      setStatusReady(false);
      setAnalytics(null);
      setSuppliers([]);
      return;
    }
    aiClient.persistToken(token);
    refreshStatus();
    loadAnalytics();
    loadSuppliers();
  }, [hasToken, token, refreshStatus, loadAnalytics, loadSuppliers]);

  const handleChatSend = async () => {
    if (!prompt || !hasToken) return;
    const userMessage = { role: 'user', content: prompt };
    setConversation((prev) => [...prev, userMessage]);
    setPrompt('');
    setChatError('');
    setChatLoading(true);
    try {
      const response = await sendChatPrompt(token, userMessage.content);
      const assistantMessage = {
        role: 'assistant',
        content: response.message || response.Message || 'Pas de réponse IA',
      };
      setConversation((prev) => [...prev, assistantMessage]);
    } catch (error) {
      setChatError(error.message);
    } finally {
      setChatLoading(false);
    }
  };

  const handleLogout = () => {
    aiClient.removeToken();
    setToken('');
    setConversation([]);
    setAnalytics(null);
    setSuppliers([]);
    setStatusReady(false);
  };

  return (
    <div className="ct-app">
      <Sidebar onLogout={handleLogout} />
      <main className="ct-main">
        <header>
          <h1>Hub Climat IA</h1>
          <p>Calculs, analyses et recommandations propulsées par Mistral.</p>
        </header>

        <div className="ct-token-bar">
          <span>Token JWT</span>
          <input
            type="password"
            placeholder="Collez le token obtenu après connexion"
            value={token}
            onChange={(e) => setToken(e.target.value)}
          />
          <button onClick={() => {
            if (hasToken) {
              loadAnalytics();
              loadSuppliers();
              refreshStatus();
            }
          }}>
            Synchroniser
          </button>
        </div>

        {globalError && (
          <div className="ct-error-banner">
            ⚠️ {globalError}
          </div>
        )}

        <StatsGrid
          metrics={analytics?.metrics}
          loading={loadingAnalytics}
        />

        <div className="ct-grid">
          <InsightsPanel
            title="Synthèse IA Greenly-like"
            narrative={analytics?.narrative}
            findings={analytics?.key_findings || []}
            recommendations={analytics?.recommendations || []}
            loading={loadingAnalytics}
          />
          <ChatPanel
            conversation={conversation}
            prompt={prompt}
            setPrompt={setPrompt}
            onSend={handleChatSend}
            loading={chatLoading}
            statusReady={statusReady}
            error={chatError}
          />
        </div>

        <div className="ct-grid-narrow">
          <SuppliersTable suppliers={suppliers} loading={loadingSuppliers} />
          <ScopeBreakdown scopes={analytics?.scopes || {}} />
        </div>
      </main>
    </div>
  );
}

export default App;
