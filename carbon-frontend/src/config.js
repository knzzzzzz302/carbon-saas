const API_BASE_URL =
  process.env.REACT_APP_API_URL || 'http://localhost:3030';

export const CONFIG = {
  apiBaseUrl: API_BASE_URL,
  tokenStorageKey: 'carbon_token',
};

export const endpoints = {
  analytics: `${API_BASE_URL}/ai/analytics`,
  suppliers: `${API_BASE_URL}/ai/suppliers`,
  chat: `${API_BASE_URL}/ai/chat`,
  status: `${API_BASE_URL}/ai/status`,
};

