import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import './styles.css'
import { App } from './App'
import { initTelegram, tg } from './telegram'

initTelegram()
const applyTheme = () => {
  document.documentElement.dataset.theme = tg?.colorScheme === 'dark' ? 'dark' : 'light'
}
applyTheme()
tg?.onEvent('themeChanged', applyTheme)

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <App />
  </StrictMode>,
)
