import { Component, type ErrorInfo, type ReactNode } from "react";

type Props = {
  children: ReactNode;
  label: string;
};

type State = {
  failed: boolean;
};

export class WorkspaceErrorBoundary extends Component<Props, State> {
  state: State = { failed: false };

  static getDerivedStateFromError() {
    return { failed: true };
  }

  componentDidCatch(error: Error, info: ErrorInfo) {
    console.error(`${this.props.label} could not be shown`, error, info.componentStack);
  }

  render() {
    if (this.state.failed) {
      return <section className="inline-error" role="alert">
        <div><strong>{this.props.label} could not be shown</strong><p>The rest of the workspace is still available. Try again after reloading this view.</p></div>
        <button className="secondary-button" type="button" onClick={() => this.setState({ failed: false })}>Try again</button>
      </section>;
    }
    return this.props.children;
  }
}
