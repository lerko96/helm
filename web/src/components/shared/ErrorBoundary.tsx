import { Component, type ReactNode } from 'react'

interface Props {
  widgetTitle: string
  children: ReactNode
}

interface State {
  hasError: boolean
  message: string
}

export default class ErrorBoundary extends Component<Props, State> {
  state: State = { hasError: false, message: '' }

  static getDerivedStateFromError(error: Error): State {
    return { hasError: true, message: error.message }
  }

  componentDidCatch() {}

  render() {
    if (this.state.hasError) {
      return (
        <div style={{ padding: '12px' }}>
          <span
            className="status status-alert"
            style={{ fontSize: 'var(--text-xs)', letterSpacing: 'var(--letter-spacing-label)' }}
          >
            {this.props.widgetTitle.toUpperCase()} — WIDGET ERROR
          </span>
        </div>
      )
    }
    return this.props.children
  }
}
