import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import './index.css'
import App from './App'
import { WebSocketProvider } from './hooks/WebSocketProvider'

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <WebSocketProvider>
      <App />
    </WebSocketProvider>
  </StrictMode>,
)
