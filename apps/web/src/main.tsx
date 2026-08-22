import { StrictMode } from 'react';
import { createRoot } from 'react-dom/client';
import { QueryClientProvider } from '@tanstack/react-query';

import './design-system/styles/index.css';
import './application/i18n';
import { App } from './App';
import { client } from './api/generated/client.gen';
import { queryClient } from './application/query';

const container = document.getElementById('root');

if (!container) {
  throw new Error('The #root element is missing from index.html.');
}

client.setConfig({ baseUrl: window.location.origin, credentials: 'include' });

createRoot(container).render(
  <StrictMode>
    <QueryClientProvider client={queryClient}>
      <App />
    </QueryClientProvider>
  </StrictMode>,
);
