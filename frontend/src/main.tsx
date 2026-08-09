import { StrictMode } from 'react';
import { createRoot } from 'react-dom/client';
import { Provider } from 'react-redux';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { BrowserRouter } from 'react-router-dom';
import '@fontsource-variable/inter';
import '@fontsource-variable/fraunces';
import '@/styles/index.css';
import App from './App';
import { store } from './store';
import { Toaster } from './components/ui/Toast';
import { startSessionRefresher } from './services/sessionRefresher';

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 15_000,
      retry: 1,
      refetchOnWindowFocus: false,
    },
  },
});

// Rotate the session token ahead of expiry and on tab refocus, so long-lived
// admin sessions survive without forcing re-login (server TTL is 24h).
startSessionRefresher();

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <Provider store={store}>
      <QueryClientProvider client={queryClient}>
        <BrowserRouter>
          <App />
          <Toaster />
        </BrowserRouter>
      </QueryClientProvider>
    </Provider>
  </StrictMode>,
);
