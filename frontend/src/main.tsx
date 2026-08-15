import React, { Component, type ErrorInfo, type ReactNode } from 'react';
import { createRoot } from 'react-dom/client';
import App from './App';
import './styles.css';

class AppErrorBoundary extends Component<{ children: ReactNode }, { error: Error | null }> {
  state = { error: null as Error | null };

  static getDerivedStateFromError(error: Error) { return { error }; }

  componentDidCatch(error: Error, info: ErrorInfo) {
    console.error('tabpo render error', error, info.componentStack);
  }

  render() {
    if (this.state.error) {
      return <div className="fatal-error"><h2>テーブルの表示に失敗しました</h2><p>{this.state.error.message}</p><p className="fatal-error-help">「更新」を押すか、別のテーブルを選択してください。</p><button onClick={() => window.location.reload()}>アプリを再読み込み</button></div>;
    }
    return this.props.children;
  }
}

createRoot(document.getElementById('root')!).render(<React.StrictMode><AppErrorBoundary><App /></AppErrorBoundary></React.StrictMode>);
