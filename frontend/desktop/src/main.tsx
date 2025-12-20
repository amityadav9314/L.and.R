import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import 'bootstrap/dist/css/bootstrap.min.css'
import './index.css'
import App from './App.tsx'
import { GoogleOAuthProvider } from '@react-oauth/google'
import { GOOGLE_WEB_CLIENT_ID } from './utils/config'

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <GoogleOAuthProvider clientId={GOOGLE_WEB_CLIENT_ID}>
      <App />
    </GoogleOAuthProvider>
  </StrictMode>,
)
