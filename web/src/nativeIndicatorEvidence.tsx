import { IndicatorDetail } from "./components/indicators/IndicatorDetail";
import { sampleIndicatorHistory, sampleMobileSuccessIndicator } from "./indicatorEvidenceData";

export function NativeIndicatorEvidencePage() {
  return <main className="native-indicator-evidence">
    <header className="topbar">
      <div>
        <span className="eyebrow">Sample data · Digital channels</span>
        <h1>Risk indicator</h1>
        <p>Sample observations dated 5 October 2026.</p>
      </div>
    </header>
    <IndicatorDetail
      indicator={sampleMobileSuccessIndicator}
      loadResults={async () => sampleIndicatorHistory}
    />
  </main>;
}
