import { CONFIG, endpoints } from '../config';

const jsonHeaders = (token) => ({
  'Content-Type': 'application/json',
  ...(token ? { Authorization: `Bearer ${token}` } : {}),
});

const handleResponse = async (res) => {
  if (!res.ok) {
    let message = `Erreur API (${res.status})`;
    try {
      const data = await res.json();
      if (data?.error) {
        message = data.error;
      }
    } catch (e) {
      // ignore parsing error
    }
    throw new Error(message);
  }
  return res.json();
};

export const aiClient = {
  persistToken(value) {
    localStorage.setItem(CONFIG.tokenStorageKey, value);
  },
  getToken() {
    return localStorage.getItem(CONFIG.tokenStorageKey) || '';
  },
  removeToken() {
    localStorage.removeItem(CONFIG.tokenStorageKey);
  },
};

export const fetchAIStatus = (token) =>
  fetch(endpoints.status, {
    headers: jsonHeaders(token),
  }).then(handleResponse);

export const fetchAnalytics = (token) =>
  fetch(endpoints.analytics, {
    headers: jsonHeaders(token),
  }).then(handleResponse);

export const fetchSuppliers = (token) =>
  fetch(endpoints.suppliers, {
    headers: jsonHeaders(token),
  }).then(handleResponse);

export const sendChatPrompt = (token, prompt) =>
  fetch(endpoints.chat, {
    method: 'POST',
    headers: jsonHeaders(token),
    body: JSON.stringify({ prompt }),
  }).then(handleResponse);

