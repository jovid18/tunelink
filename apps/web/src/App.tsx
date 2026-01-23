import { useState } from 'react'
import './App.css'

// Production: use relative path (Ingress routes /api to API service)
// Development: use VITE_API_URL if set, otherwise relative path
const API_URL = import.meta.env.VITE_API_URL || ''

function App() {
  const [url, setUrl] = useState('')
  const [shortUrl, setShortUrl] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError('')
    setShortUrl('')
    setLoading(true)

    try {
      const response = await fetch(`${API_URL}/api/urls`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ originalUrl: url }),
      })

      if (!response.ok) {
        throw new Error('Failed to shorten URL')
      }

      const data = await response.json()
      setShortUrl(data.fullUrl)
    } catch (err) {
      setError('Failed to shorten URL. Please try again.')
    } finally {
      setLoading(false)
    }
  }

  const copyToClipboard = () => {
    navigator.clipboard.writeText(shortUrl)
  }

  return (
    <div className="container">
      <h1>TuneLink</h1>
      <p className="subtitle">URL Shortener</p>

      <form onSubmit={handleSubmit} className="form">
        <input
          type="url"
          value={url}
          onChange={(e) => setUrl(e.target.value)}
          placeholder="Enter your long URL here..."
          required
          className="input"
        />
        <button type="submit" disabled={loading} className="button">
          {loading ? 'Shortening...' : 'Shorten'}
        </button>
      </form>

      {error && <p className="error">{error}</p>}

      {shortUrl && (
        <div className="result">
          <p>Your shortened URL:</p>
          <div className="short-url-container">
            <a href={shortUrl} target="_blank" rel="noopener noreferrer" className="short-url">
              {shortUrl}
            </a>
            <button onClick={copyToClipboard} className="copy-button">
              Copy
            </button>
          </div>
        </div>
      )}
    </div>
  )
}

export default App
