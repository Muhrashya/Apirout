import React from 'react'
import ReactDOM from 'react-dom/client'
import App from './App'
import './index.css'

// Auto-recover when Vite asset chunks are updated in a new deployment
window.addEventListener('vite:preloadError', (event) => {
    console.warn('Vite asset chunk update detected, reloading application...', event);
    window.location.reload();
});

ReactDOM.createRoot(document.getElementById('root')!).render(
    <React.StrictMode>
        <App />
    </React.StrictMode>,
)

