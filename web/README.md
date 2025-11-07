# eBPF Microsegmentation Web UI

Modern web-based management interface for the eBPF Microsegmentation system.

## Tech Stack

- **Frontend Framework**: React 19 + TypeScript
- **Build Tool**: Vite 7
- **UI Library**: Ant Design 5
- **Routing**: React Router DOM 7
- **State Management**: Zustand 5
- **Data Fetching**: TanStack Query 5
- **HTTP Client**: Axios 1
- **Code Quality**: ESLint + Prettier

## Features

- 📊 Real-time Dashboard with system overview
- 🖥️ Agent Management and monitoring
- 🌐 Network Flow visualization and analytics
- 🔒 Security Policy configuration
- 📱 Responsive design for mobile and desktop
- ⚡ Fast HMR (Hot Module Replacement) with Vite
- 🎨 Modern UI with Ant Design components

## Prerequisites

- Node.js >= 18.0.0
- npm >= 9.0.0
- Backend Server running on http://localhost:8080

## Getting Started

### Installation

```bash
# Install dependencies
npm install
```

### Development

```bash
# Start development server (http://localhost:3000)
npm run dev
```

The development server includes:
- Hot Module Replacement (HMR)
- API proxy to backend (configured in vite.config.ts)
- Source maps for debugging

### Building for Production

```bash
# Build for production
npm run build

# Preview production build
npm run preview
```

Build output will be in the `dist/` directory.

### Code Quality

```bash
# Run ESLint
npm run lint

# Format code with Prettier
npm run format

# Check code formatting
npm run format:check

# Type checking
npx tsc --noEmit
```

## Project Structure

```
src/
├── api/              # API client and endpoints
│   ├── client.ts     # Axios configuration
│   ├── agents.ts     # Agent API
│   ├── flows.ts      # Flow API
│   └── policies.ts   # Policy API
├── components/       # React components
│   ├── layout/       # Layout components
│   │   ├── Header.tsx
│   │   ├── Sidebar.tsx
│   │   └── MainLayout.tsx
│   └── common/       # Reusable components
├── pages/            # Page components
│   ├── Dashboard/    # Dashboard page
│   ├── Agents/       # Agents management page
│   ├── Flows/        # Network flows page
│   └── Policies/     # Security policies page
├── hooks/            # Custom React hooks
│   └── useAgents.ts  # TanStack Query hooks
├── store/            # Zustand stores (future use)
├── types/            # TypeScript type definitions
│   ├── common.ts     # Common types
│   ├── agent.ts      # Agent types
│   ├── flow.ts       # Flow types
│   └── policy.ts     # Policy types
├── utils/            # Utility functions
├── router.tsx        # Route configuration
└── main.tsx          # Application entry point
```

## Environment Variables

Create a `.env.development` file for development:

```env
VITE_API_BASE_URL=http://localhost:8080
```

For production, create a `.env.production` file:

```env
VITE_API_BASE_URL=http://your-server-address:8080
```

## API Configuration

The application expects the backend API to be available at the configured `VITE_API_BASE_URL`.

In development mode, the Vite dev server proxies `/api/*` requests to the backend server to avoid CORS issues.

## Available Scripts

| Script | Description |
|--------|-------------|
| `npm run dev` | Start development server |
| `npm run build` | Build for production |
| `npm run preview` | Preview production build |
| `npm run lint` | Run ESLint |
| `npm run format` | Format code with Prettier |
| `npm run format:check` | Check code formatting |

## Browser Support

- Chrome (latest)
- Firefox (latest)
- Safari (latest)
- Edge (latest)

## Performance

- **Bundle Size**: ~250 KB gzip (production)
- **Dev Server Start**: < 3 seconds
- **Production Build**: < 10 seconds
- **First Paint**: < 1 second

## Development Guidelines

### Code Style

- Use TypeScript for all new files
- Follow ESLint and Prettier rules
- Use functional components with hooks
- Prefer `const` over `let`
- Use arrow functions for callbacks

### Component Guidelines

- One component per file
- Use named exports for components
- Props should have TypeScript interfaces
- Handle loading and error states
- Use Ant Design components when possible

### API Integration

- Use TanStack Query for data fetching
- Define API types in `src/types/`
- Handle errors gracefully
- Show loading states to users

## Troubleshooting

### Port Already in Use

If port 3000 is already in use, Vite will automatically try the next available port.

### CORS Errors

Make sure the backend server has CORS configured to allow requests from `http://localhost:3000`.

### Build Errors

If you encounter build errors:

1. Clear node_modules: `rm -rf node_modules`
2. Clear package-lock.json: `rm package-lock.json`
3. Reinstall dependencies: `npm install`
4. Try building again: `npm run build`

## Future Enhancements

- [ ] WebSocket support for real-time updates
- [ ] Dark mode theme
- [ ] Internationalization (i18n)
- [ ] User authentication
- [ ] Advanced data visualization charts
- [ ] Export reports functionality

## License

See the main project LICENSE file.

## Contributing

Contributions are welcome! Please follow the project's coding standards and submit pull requests for review.
