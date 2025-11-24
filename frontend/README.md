# LLM Council - Frontend

React-based web interface for the LLM Council application. Displays multi-stage LLM deliberations with tabbed response viewing, peer review visualization, and final synthesis.

## Quick Start

```bash
# Install dependencies
npm install

# Start development server
npm run dev
```

Open http://localhost:5173 (requires backend running on port 8001)

## Prerequisites

- **Node.js** 18+ and npm
- **Backend server** running on http://localhost:8001

## Available Commands

```bash
npm run dev      # Development server (port 5173)
npm run build    # Production build
npm run preview  # Preview production build
npm run lint     # Run ESLint
```

## Project Structure

```
frontend/
├── src/
│   ├── App.jsx                 # Main app, conversation management
│   ├── components/
│   │   ├── ChatInterface.jsx   # Message input with multiline support
│   │   ├── ConversationList.jsx
│   │   ├── Stage1.jsx          # Individual model responses (tabs)
│   │   ├── Stage2.jsx          # Peer review and rankings
│   │   └── Stage3.jsx          # Final synthesis (green background)
│   ├── api.js                  # Backend API client
│   ├── index.css               # Global styles
│   └── main.jsx                # React entry point
├── public/                     # Static assets
├── index.html
├── package.json
└── vite.config.js
```

## Architecture

### Core Components

**App.jsx** - Main container
- Manages conversation list and current conversation state
- Handles navigation between conversations
- Orchestrates message submission

**ChatInterface.jsx** - User input
- Multiline textarea with Enter to send, Shift+Enter for newlines
- Loading states during API requests
- Message history display

**Stage1.jsx** - Individual responses
- Tabbed interface for each council member's response
- Syntax-highlighted markdown rendering
- Model name and response time display

**Stage2.jsx** - Peer review
- Shows each model's raw evaluation text
- Client-side de-anonymization (models received anonymous labels)
- Extracted ranking validation
- Aggregate ranking with average positions

**Stage3.jsx** - Final answer
- Chairman's synthesized response
- Distinctive green-tinted background (#f0fff0)
- Full markdown rendering

### Styling System

- **Primary color:** #4a90e2 (blue)
- **Theme:** Light mode only
- **Markdown:** All content wrapped in `<div className="markdown-content">` for proper spacing

### API Integration

The frontend communicates with the backend via `api.js`:

```javascript
// Key endpoints
GET    /api/conversations              // List conversations
POST   /api/conversations              // Create new
GET    /api/conversations/:id          // Get conversation
POST   /api/conversations/:id/message  // Send message (batch)
POST   /api/conversations/:id/message/stream  // SSE streaming
```

**Batch mode** returns all stages at once after processing completes.
**Streaming mode** sends real-time updates as each stage completes.

## Configuration

### Backend URL

Edit `src/api.js` to change the backend URL (default: http://localhost:8001):

```javascript
const API_BASE_URL = 'http://localhost:8001';
```

### Port

Edit `vite.config.js` to change the dev server port (default: 5173):

```javascript
export default defineConfig({
  server: {
    port: 5173
  }
})
```

## Development

### Hot Module Replacement

Vite provides instant HMR - changes appear immediately without page refresh.

### Adding New Components

1. Create component in `src/components/`
2. Import in parent component
3. Use standard React patterns (hooks recommended)

### Styling

- Global styles in `src/index.css`
- Component-specific styles can be inline or CSS modules
- Markdown content requires `className="markdown-content"` wrapper

## Troubleshooting

### "Failed to fetch" errors

- Verify backend is running: `curl http://localhost:8001/`
- Check browser console for CORS errors
- Ensure backend is on port 8001

### Port 5173 already in use

```bash
# Kill existing process
lsof -i :5173
kill -9 <PID>

# Or use a different port in vite.config.js
```

### Build errors

```bash
# Clear node_modules and reinstall
rm -rf node_modules package-lock.json
npm install
```

### Markdown not rendering

Ensure content is wrapped: `<div className="markdown-content">{content}</div>`

## Tech Stack

- **React 19** - UI framework
- **Vite** - Build tool and dev server
- **react-markdown** - Markdown rendering with syntax highlighting
- **ESLint** - Code quality and linting

## Production Build

```bash
# Build for production
npm run build

# Output in dist/ directory
# Deploy dist/ to any static hosting (Vercel, Netlify, S3, etc.)
```

The production build is optimized and minified. Remember to set the correct API_BASE_URL for your production backend.

## Browser Support

Modern browsers with ES6+ support:
- Chrome 90+
- Firefox 88+
- Safari 14+
- Edge 90+
